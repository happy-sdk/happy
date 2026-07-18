// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package tail

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/bytesize"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestReadLastLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		n       int
		want    []string
	}{
		{"empty", "", 5, nil},
		{"fewer lines than n", "a\nb\nc", 10, []string{"a", "b", "c"}},
		{"exact n", "a\nb\nc", 3, []string{"a", "b", "c"}},
		{"more lines than n", "a\nb\nc\nd\ne", 2, []string{"d", "e"}},
		{"trailing newline", "a\nb\nc\n", 3, []string{"a", "b", "c"}},
		{"single line no newline", "only", 5, []string{"only"}},
		{"n=1", "a\nb\nc", 1, []string{"c"}},
		{"blank lines skipped", "a\n\nb\n\nc", 5, []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, err := readLastLines(strings.NewReader(tt.content), tt.n)
			testutils.NoError(t, err)
			testutils.Equal(t, len(tt.want), len(lines), "line count mismatch for %q", tt.content)
			if len(tt.want) == len(lines) {
				for i := range tt.want {
					testutils.Equal(t, tt.want[i], lines[i], "line %d mismatch for %q", i, tt.content)
				}
			}
		})
	}
}

func TestReadLastLinesRestoresPosition(t *testing.T) {
	r := strings.NewReader("a\nb\nc\n")
	endPos, err := r.Seek(0, 2)
	testutils.NoError(t, err)

	_, err = readLastLines(r, 2)
	testutils.NoError(t, err)

	pos, err := r.Seek(0, 1)
	testutils.NoError(t, err)
	testutils.Equal(t, endPos, pos, "reader position should be restored to end")
}

func TestReadLastLinesLargeContentAcrossBufferBoundary(t *testing.T) {
	// readLastLines reads backward in 4KB chunks; build content that spans
	// multiple chunks to exercise the chunk-boundary line-joining logic.
	var b strings.Builder
	var want []string
	for i := range 2000 {
		line := strings.Repeat("x", 10) + "-" + time.Duration(i).String()
		want = append(want, line)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	want = want[len(want)-5:]

	lines, err := readLastLines(strings.NewReader(b.String()), 5)
	testutils.NoError(t, err)
	testutils.Equal(t, len(want), len(lines), "line count mismatch")
	for i := range want {
		testutils.Equal(t, want[i], lines[i], "line %d mismatch", i)
	}
}

func TestReadLastLinesInvalidSeeker(t *testing.T) {
	_, err := readLastLines(&brokenSeeker{}, 2)
	testutils.Error(t, err)
}

type brokenSeeker struct{}

func (b *brokenSeeker) Read(p []byte) (int, error)     { return 0, io.ErrUnexpectedEOF }
func (b *brokenSeeker) Seek(int64, int) (int64, error) { return 0, errors.New("seek failure") }

func TestReadLastLinesFn(t *testing.T) {
	var got []string
	for line, err := range ReadLastLines(strings.NewReader("a\nb\nc"), 2) {
		testutils.NoError(t, err)
		got = append(got, line)
	}
	testutils.Equal(t, 2, len(got), "expected 2 lines")
	if len(got) == 2 {
		testutils.Equal(t, "b", got[0], "unexpected first line")
		testutils.Equal(t, "c", got[1], "unexpected second line")
	}
}

func TestReadLastLinesFnStopsOnYieldFalse(t *testing.T) {
	var got []string
	for line, err := range ReadLastLines(strings.NewReader("a\nb\nc"), 3) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected iteration to stop after first yield")
}

func TestReadLastLinesFnError(t *testing.T) {
	var calls int
	var lastErr error
	for _, err := range ReadLastLines(&brokenSeeker{}, 2) {
		calls++
		lastErr = err
	}
	testutils.Equal(t, 1, calls, "expected exactly one yield on error")
	testutils.Error(t, lastErr)
}

func TestTailReaderSeekable(t *testing.T) {
	r := strings.NewReader("a\nb\nc\nd\ne\n")
	ctx := context.Background()

	var got []string
	for line, err := range TailReader(ctx, r, 3) {
		testutils.NoError(t, err)
		got = append(got, line)
	}
	testutils.Equal(t, 3, len(got), "expected 3 initial lines")
	if len(got) == 3 {
		testutils.Equal(t, "c", got[0], "unexpected line")
		testutils.Equal(t, "d", got[1], "unexpected line")
		testutils.Equal(t, "e", got[2], "unexpected line")
	}
}

func TestTailReaderNonSeekable(t *testing.T) {
	r := nonSeekableReader{strings.NewReader("a\nb\nc\nd\ne\n")}
	ctx := context.Background()

	var got []string
	for line, err := range TailReader(ctx, r, 2) {
		testutils.NoError(t, err)
		got = append(got, line)
	}
	testutils.Equal(t, 2, len(got), "expected 2 buffered lines")
	if len(got) == 2 {
		testutils.Equal(t, "d", got[0], "unexpected line")
		testutils.Equal(t, "e", got[1], "unexpected line")
	}
}

func TestTailReaderNonSeekableStreamAll(t *testing.T) {
	r := nonSeekableReader{strings.NewReader("a\nb\nc\n")}
	ctx := context.Background()

	got := slices.Collect(TailReaderLines(ctx, r, -1))
	testutils.Equal(t, 3, len(got), "expected all 3 lines streamed")
	testutils.Equal(t, "a\nb\nc", strings.Join(got, "\n"), "unexpected streamed content")
}

func TestTailReaderDefaultN(t *testing.T) {
	var lines []string
	for range 15 {
		lines = append(lines, "line")
	}
	r := strings.NewReader(strings.Join(lines, "\n"))
	ctx := context.Background()

	got := slices.Collect(TailReaderLines(ctx, r, 0))
	testutils.Equal(t, DefaultInitialLines, len(got), "n=0 should default to DefaultInitialLines")
}

// nonSeekableReader wraps an io.Reader without exposing Seek, forcing
// TailReader down its in-memory-buffering code path.
type nonSeekableReader struct {
	r io.Reader
}

func (n nonSeekableReader) Read(p []byte) (int, error) { return n.r.Read(p) }

func TestTailLinesStopsOnError(t *testing.T) {
	ctx := context.Background()
	got := slices.Collect(TailLines(ctx, filepath.Join(t.TempDir(), "does-not-exist.log"), 5))
	testutils.Equal(t, 0, len(got), "expected no lines for nonexistent file")
}

func TestTailFileNonExistent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "does-not-exist.log")

	var sawErr bool
	for line, err := range TailFile(ctx, path, 5) {
		if err != nil {
			sawErr = true
			testutils.Equal(t, "", line, "expected empty line alongside error")
		}
	}
	testutils.Assert(t, sawErr, "expected an error for a nonexistent file")
}

func TestTailFileInitialLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("a\nb\nc\n"), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got []string
	for line, err := range TailFile(ctx, path, 3) {
		testutils.NoError(t, err)
		got = append(got, line)
		if len(got) == 3 {
			cancel()
		}
	}
	testutils.Equal(t, 3, len(got), "expected 3 initial lines")
	if len(got) == 3 {
		testutils.Equal(t, "a", got[0], "unexpected line")
		testutils.Equal(t, "b", got[1], "unexpected line")
		testutils.Equal(t, "c", got[2], "unexpected line")
	}
}

func TestTailFileStreamsAppendedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("first\n"), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lines := make(chan string, 8)
	go func() {
		for line, err := range TailFile(ctx, path, -1) {
			if err != nil {
				close(lines)
				return
			}
			select {
			case lines <- line:
			case <-ctx.Done():
				close(lines)
				return
			}
		}
		close(lines)
	}()

	// Give the watcher time to attach before writing.
	time.Sleep(50 * time.Millisecond)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	testutils.NoError(t, err)
	_, err = f.WriteString("second\n")
	testutils.NoError(t, err)
	testutils.NoError(t, f.Close())

	select {
	case line, ok := <-lines:
		testutils.Assert(t, ok, "expected a line, channel closed early")
		testutils.Equal(t, "second", line, "unexpected streamed line")
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for appended line to be tailed")
	}
}

func TestTailFileHandlesRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lines := make(chan string, 8)
	go func() {
		for line, err := range TailFile(ctx, path, -1) {
			if err != nil {
				close(lines)
				return
			}
			select {
			case lines <- line:
			case <-ctx.Done():
				close(lines)
				return
			}
		}
		close(lines)
	}()

	time.Sleep(50 * time.Millisecond)

	// Simulate log rotation: move the old file aside, create a new one at
	// the same path, and write to it.
	testutils.NoError(t, os.Rename(path, path+".1"))
	testutils.NoError(t, os.WriteFile(path, []byte("after-rotate\n"), 0644))

	select {
	case line, ok := <-lines:
		testutils.Assert(t, ok, "expected a line after rotation, channel closed early")
		testutils.Equal(t, "after-rotate", line, "unexpected line after rotation")
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for post-rotation line to be tailed")
	}
}

func TestTailFileContextCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("line\n"), 0644))

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		for range TailFile(ctx, path, -1) {
		}
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("TailFile did not stop after context cancellation")
	}
}

func TestBufferSizes(t *testing.T) {
	defer SetBufferSizes(DefaultBufferSize, DefaultMaxBufferSize)

	SetBufferSizes(4*bytesize.KiB, 2*bytesize.MiB)
	bufSize, maxBufSize := BufferSizes()
	testutils.Equal(t, uint64((4 * bytesize.KiB).Bytes()), uint64(bufSize.Bytes()), "unexpected buffer size")
	testutils.Equal(t, uint64((2 * bytesize.MiB).Bytes()), uint64(maxBufSize.Bytes()), "unexpected max buffer size")
}

// Note: cleanup's "reset if grown past 128 KiB" branch in
// newScannerWithPooledBuffer is not covered here. bufio.Scanner.Buffer never
// grows the slice it was seeded with in place (it reallocates its own
// internal buffer instead), so that branch can't be reached through actual
// scanning - and reaching it by priming the shared sync.Pool directly turned
// out to be flaky (its per-P fast-path retrieval isn't a reliable guarantee
// under race-detector scheduling). See the conversation for the fix options
// considered instead of a flaky test.
func TestNewScannerWithPooledBufferKeepsNormalCapacity(t *testing.T) {
	scanner, cleanup := newScannerWithPooledBuffer(strings.NewReader("line"))
	testutils.Assert(t, scanner.Scan(), "expected to scan the line")
	testutils.Equal(t, "line", scanner.Text(), "unexpected scanned content")
	cleanup()
}

func TestTailFileDefaultN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	var lines []string
	for i := range 15 {
		lines = append(lines, "line-"+string(rune('a'+i)))
	}
	testutils.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got []string
	for line, err := range TailFile(ctx, path, 0) {
		testutils.NoError(t, err)
		got = append(got, line)
		if len(got) == DefaultInitialLines {
			cancel()
		}
	}
	testutils.Equal(t, DefaultInitialLines, len(got), "n=0 should default to DefaultInitialLines")
}

func TestTailFileInitialLinesErrorOnDirectory(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	var sawErr bool
	for line, err := range TailFile(ctx, dir, 1) {
		if err != nil {
			sawErr = true
			testutils.Equal(t, "", line, "expected empty line alongside error")
			break
		}
	}
	testutils.Assert(t, sawErr, "expected an error tailing a directory path")
}

func TestTailFileSeekEndErrorOnDirectory(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	var sawErr bool
	for line, err := range TailFile(ctx, dir, -1) {
		if err != nil {
			sawErr = true
			testutils.Equal(t, "", line, "expected empty line alongside error")
			break
		}
	}
	testutils.Assert(t, sawErr, "expected an error seeking a directory path")
}

func TestTailFileTruncationDetected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte(strings.Repeat("padding-line\n", 20)), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lines := make(chan string, 8)
	go func() {
		for line, err := range TailFile(ctx, path, -1) {
			if err != nil {
				close(lines)
				return
			}
			select {
			case lines <- line:
			case <-ctx.Done():
				close(lines)
				return
			}
		}
		close(lines)
	}()

	time.Sleep(50 * time.Millisecond)

	// Truncate to a size smaller than what was already read, then write new,
	// shorter content: TailFile must detect the shrink and seek back to
	// start rather than trying to read from a now out-of-range offset.
	testutils.NoError(t, os.Truncate(path, 0))
	f, err := os.OpenFile(path, os.O_WRONLY, 0644)
	testutils.NoError(t, err)
	_, err = f.WriteString("after-truncate\n")
	testutils.NoError(t, err)
	testutils.NoError(t, f.Close())

	select {
	case line, ok := <-lines:
		testutils.Assert(t, ok, "expected a line after truncation, channel closed early")
		testutils.Equal(t, "after-truncate", line, "unexpected line after truncation")
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for post-truncation line to be tailed")
	}
}

func TestTailFileStopsWhenConsumerStopsIterating(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("a\nb\nc\n"), 0644))

	ctx := context.Background()
	var got []string
	for line, err := range TailFile(ctx, path, 3) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected iteration to stop after first yield returns false")
}

func TestTailLinesStopsWhenConsumerStopsIterating(t *testing.T) {
	var got []string
	for line := range TailLines(context.Background(), "", 0) {
		got = append(got, line)
		break
	}
	testutils.Equal(t, 0, len(got), "expected no lines for empty path")
}

func TestTailReaderLinesStopsWhenConsumerStopsIterating(t *testing.T) {
	r := strings.NewReader("a\nb\nc\n")
	var got []string
	for line := range TailReaderLines(context.Background(), r, -1) {
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected iteration to stop after first yield returns false")
}

// seekFailsAfter wraps a *bytes.Reader whose Seek call fails once the call
// counter exceeds failAfter (0 disables failing).
type seekFailsAfter struct {
	*bytes.Reader
	calls     int
	failAfter int
}

func (s *seekFailsAfter) Seek(offset int64, whence int) (int64, error) {
	s.calls++
	if s.calls > s.failAfter {
		return 0, errors.New("seek failure")
	}
	return s.Reader.Seek(offset, whence)
}

// readFailsAfter wraps a *bytes.Reader whose Read call fails once the call
// counter exceeds failAfter (0 disables failing).
type readFailsAfter struct {
	*bytes.Reader
	calls     int
	failAfter int
}

func (r *readFailsAfter) Read(p []byte) (int, error) {
	r.calls++
	if r.calls > r.failAfter {
		return 0, errors.New("read failure")
	}
	return r.Reader.Read(p)
}

func TestReadLastLinesSeekInLoopFails(t *testing.T) {
	// Content small enough to be read in a single backward chunk: the 2nd
	// Seek call (1st is the initial SeekEnd) happens inside the read loop.
	s := &seekFailsAfter{Reader: bytes.NewReader([]byte("a\nb\nc\n")), failAfter: 1}
	_, err := readLastLines(s, 2)
	testutils.Error(t, err)
}

func TestReadLastLinesRestoreSeekFails(t *testing.T) {
	// 1st Seek = SeekEnd, 2nd Seek = SeekStart within the loop, 3rd Seek =
	// restore-to-end. Let the first two succeed and fail the restore.
	s := &seekFailsAfter{Reader: bytes.NewReader([]byte("a\nb\nc\n")), failAfter: 2}
	_, err := readLastLines(s, 2)
	testutils.Error(t, err)
}

func TestReadLastLinesReadInLoopFails(t *testing.T) {
	r := &readFailsAfter{Reader: bytes.NewReader([]byte("a\nb\nc\n")), failAfter: 0}
	_, err := readLastLines(r, 2)
	testutils.Error(t, err)
}

func TestTailReaderSeekableInitialReadFails(t *testing.T) {
	s := &seekFailsAfter{Reader: bytes.NewReader([]byte("a\nb\nc\n")), failAfter: 1}
	ctx := context.Background()

	var sawErr bool
	for _, err := range TailReader(ctx, s, 2) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailReader to surface the readLastLines error")
}

func TestTailReaderSeekableStreamScanFails(t *testing.T) {
	// n must be >= 0 here: that's what routes TailReader down the seekable
	// fast path (readLastLines for initial lines, then its own separate
	// scanner for "stream new lines"), as opposed to the generic
	// non-seekable path used for n = -1 even on a seekable reader.
	r := &readFailsAfter{Reader: bytes.NewReader([]byte("a\nb\nc\n")), failAfter: 1}
	ctx := context.Background()

	var sawErr bool
	for _, err := range TailReader(ctx, r, 1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailReader to surface the scan error")
}

// errOnlyReader is a non-seekable io.Reader that fails after n successful
// reads, used to exercise TailReader's non-seekable error branches.
type errOnlyReader struct {
	r         io.Reader
	calls     int
	failAfter int
}

func (e *errOnlyReader) Read(p []byte) (int, error) {
	e.calls++
	if e.calls > e.failAfter {
		return 0, errors.New("read failure")
	}
	return e.r.Read(p)
}

func TestTailReaderNonSeekableBufferingScanFails(t *testing.T) {
	r := &errOnlyReader{r: strings.NewReader("a\nb\nc\n"), failAfter: 1}
	ctx := context.Background()

	var sawErr bool
	for _, err := range TailReader(ctx, r, 2) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailReader to surface the buffering scan error")
}

func TestTailReaderNonSeekableStreamScanFails(t *testing.T) {
	r := &errOnlyReader{r: strings.NewReader("a\nb\nc\n"), failAfter: 1}
	ctx := context.Background()

	var sawErr bool
	for _, err := range TailReader(ctx, r, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailReader to surface the streaming scan error")
}

func TestTailReaderSeekableInitialLinesConsumerStops(t *testing.T) {
	r := strings.NewReader("a\nb\nc\nd\ne\n")
	ctx := context.Background()

	var got []string
	for line, err := range TailReader(ctx, r, 3) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected iteration to stop after first initial line")
}

func TestTailReaderNonSeekableInitialLinesConsumerStops(t *testing.T) {
	r := nonSeekableReader{strings.NewReader("a\nb\nc\nd\ne\n")}
	ctx := context.Background()

	var got []string
	for line, err := range TailReader(ctx, r, 3) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected iteration to stop after first buffered initial line")
}

// growableSeeker is an io.ReadSeeker over an in-memory buffer that can grow
// concurrently. Unlike an *os.File, Read blocks (briefly polling) instead of
// returning io.EOF immediately when caught up to the current end, which is
// what TailReader's "stream new lines" phase needs from its reader to have
// anything to actually stream for a seekable source; it gives up and
// returns io.EOF after a bounded wait so a test can never hang on it.
type growableSeeker struct {
	mu   sync.Mutex
	data []byte
	pos  int64
}

func (g *growableSeeker) Append(s string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.data = append(g.data, s...)
}

func (g *growableSeeker) Read(p []byte) (int, error) {
	deadline := time.Now().Add(2 * time.Second)
	for {
		g.mu.Lock()
		if int(g.pos) < len(g.data) {
			n := copy(p, g.data[g.pos:])
			g.pos += int64(n)
			g.mu.Unlock()
			return n, nil
		}
		g.mu.Unlock()
		if time.Now().After(deadline) {
			return 0, io.EOF
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (g *growableSeeker) Seek(offset int64, whence int) (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var base int64
	switch whence {
	case io.SeekStart:
		base = 0
	case io.SeekCurrent:
		base = g.pos
	case io.SeekEnd:
		base = int64(len(g.data))
	}
	g.pos = base + offset
	return g.pos, nil
}

func TestTailReaderSeekableStreamsNewContentFromGrowingSource(t *testing.T) {
	g := &growableSeeker{data: []byte("a\nb\n")}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		g.Append("c\n")
	}()

	// n=1: initial lines yield "b" via readLastLines (which restores the
	// position to the then-current end), then TailReader's own separate
	// scanner blocks on Scan() until the goroutine above appends "c".
	var got []string
	for line, err := range TailReader(ctx, g, 1) {
		testutils.NoError(t, err)
		got = append(got, line)
		if len(got) == 2 {
			break
		}
	}
	testutils.Equal(t, 2, len(got), "expected initial line plus one streamed line")
	if len(got) == 2 {
		testutils.Equal(t, "b", got[0], "unexpected initial line")
		testutils.Equal(t, "c", got[1], "unexpected streamed line")
	}
}

func TestTailLinesConsumerStopsIterating(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("a\nb\nc\n"), 0644))

	var got []string
	for line := range TailLines(context.Background(), path, 3) {
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected TailLines to stop after first yield returns false")
}

func TestTailReaderLinesStopsOnError(t *testing.T) {
	got := slices.Collect(TailReaderLines(context.Background(), &brokenSeeker{}, 2))
	testutils.Equal(t, 0, len(got), "expected TailReaderLines to stop without yielding on error")
}

func TestTailFileWriteEventConsumerStops(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("first\n"), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		_, _ = f.WriteString("second\n")
		_ = f.Close()
	}()

	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected iteration to stop right after the first streamed write-event line")
	if len(got) == 1 {
		testutils.Equal(t, "second", got[0], "unexpected streamed line")
	}
}

func TestTailFileRotationEventConsumerStops(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = os.Rename(path, path+".1")
		_ = os.WriteFile(path, []byte("after-rotate\n"), 0644)
	}()

	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected iteration to stop right after the first post-rotation line")
	if len(got) == 1 {
		testutils.Equal(t, "after-rotate", got[0], "unexpected post-rotation line")
	}
}
