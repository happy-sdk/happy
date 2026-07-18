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

	"github.com/fsnotify/fsnotify"
	"github.com/happy-sdk/happy/pkg/bytesize"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

// fakeWatcher is a fully controllable fsWatcher: tests send exactly the
// events/errors they want, deterministically, instead of racing against real
// filesystem event timing. attached closes on the first Add() call, so a
// test can wait for TailFile to have actually reached (and passed) its
// initial Seek-to-end/watcher-attach step before mutating the file and
// queuing an event - otherwise a mutation made too early is already baked
// into that initial "end of file" position and the scanner finds nothing.
type fakeWatcher struct {
	mu           sync.Mutex
	addCalls     int
	failAddAt    int // 1-indexed Add() call to fail; 0 = never
	events       chan fsnotify.Event
	errors       chan error
	attached     chan struct{}
	attachedOnce sync.Once
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{
		events:   make(chan fsnotify.Event, 16),
		errors:   make(chan error, 16),
		attached: make(chan struct{}),
	}
}

func (f *fakeWatcher) Add(string) error {
	f.mu.Lock()
	f.addCalls++
	failNow := f.failAddAt != 0 && f.addCalls == f.failAddAt
	f.mu.Unlock()
	f.attachedOnce.Do(func() { close(f.attached) })
	if failNow {
		return errors.New("add failure")
	}
	return nil
}

func (f *fakeWatcher) Close() error                  { return nil }
func (f *fakeWatcher) Events() <-chan fsnotify.Event { return f.events }
func (f *fakeWatcher) Errors() <-chan error          { return f.errors }

// withFakeWatcher installs fake as TailFile's watcher for the duration of
// the test.
func withFakeWatcher(t *testing.T, fake *fakeWatcher) {
	t.Helper()
	old := newWatcher
	newWatcher = func() (fsWatcher, error) { return fake, nil }
	t.Cleanup(func() { newWatcher = old })
}

// flakyFile wraps a tailFile, failing its Stat/Seek/Read calls on demand.
type flakyFile struct {
	f          tailFile
	statCalls  int
	failStatAt int
	seekCalls  int
	failSeekAt int
	readCalls  int
	failReadAt int
}

func (w *flakyFile) Read(p []byte) (int, error) {
	w.readCalls++
	if w.failReadAt != 0 && w.readCalls == w.failReadAt {
		return 0, errors.New("read failure")
	}
	return w.f.Read(p)
}

func (w *flakyFile) Close() error { return w.f.Close() }

func (w *flakyFile) Stat() (os.FileInfo, error) {
	w.statCalls++
	if w.failStatAt != 0 && w.statCalls == w.failStatAt {
		return nil, errors.New("stat failure")
	}
	return w.f.Stat()
}

func (w *flakyFile) Seek(offset int64, whence int) (int64, error) {
	w.seekCalls++
	if w.failSeekAt != 0 && w.seekCalls == w.failSeekAt {
		return 0, errors.New("seek failure")
	}
	return w.f.Seek(offset, whence)
}

// withFlakyOpenFile installs an openFile that wraps every real file it opens
// in a flakyFile, failing Stat/Seek on the Nth *open call* (1-indexed: 1 is
// the initial open, 2 is a rotation reopen, ...) rather than the Nth
// Stat/Seek call overall, so failures can target a specific file.
func withFlakyOpenFile(t *testing.T, failStatOnOpenCall, failSeekOnOpenCall int) {
	t.Helper()
	old := openFile
	var openCalls int
	openFile = func(name string) (tailFile, error) {
		openCalls++
		real, err := old(name)
		if err != nil {
			return nil, err
		}
		fw := &flakyFile{f: real}
		if openCalls == failStatOnOpenCall {
			fw.failStatAt = 1
		}
		if openCalls == failSeekOnOpenCall {
			fw.failSeekAt = 1
		}
		return fw, nil
	}
	t.Cleanup(func() { openFile = old })
}

// appendToFileNoErr appends content to the file at path, for use from
// mutateAfterAttach's background goroutine (see its doc comment for why).
func appendToFileNoErr(path, content string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(content)
	_ = f.Close()
}

// mutateAfterAttach runs mutate in a goroutine once TailFile has attached
// its watcher (i.e. past its initial Seek-to-end). A mutation applied any
// earlier would already be baked into that initial "end of file" position,
// so the scanner would never see it as new content. mutate must not call
// any *testing.T method - by the time it observes fake.attached, the main
// test goroutine may already be consuming (or have finished consuming) the
// resulting event, so a T failure recorded here could race the test
// completing.
func mutateAfterAttach(fake *fakeWatcher, mutate func()) {
	go func() {
		<-fake.attached
		mutate()
	}()
}

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

func TestTailFileWriteEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("first\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		appendToFileNoErr(path, "second\n")
		fake.events <- fsnotify.Event{Op: fsnotify.Write}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected exactly one streamed line")
	if len(got) == 1 {
		testutils.Equal(t, "second", got[0], "unexpected streamed line")
	}
}

// TestTailFileDebounceSkipsRapidWrites exercises the "rapid writes within
// 10ms are debounced" branch: two Write events delivered back-to-back
// (deterministically, no sleeps) fall well inside that window.
func TestTailFileDebounceSkipsRapidWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("first\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		appendToFileNoErr(path, "second\n")
		fake.events <- fsnotify.Event{Op: fsnotify.Write}
		fake.events <- fsnotify.Event{Op: fsnotify.Write}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Don't break on the first line: the second (debounced) event is still
	// queued and only gets pulled off the channel once this iteration
	// finishes and the outer loop goes back to select. Schedule the cancel
	// asynchronously so that happens instead of returning immediately.
	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		time.AfterFunc(20*time.Millisecond, cancel)
	}
	testutils.Equal(t, 1, len(got), "expected exactly one streamed line despite two rapid write events")
}

func TestTailFileWriteEventStatFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("first\n"), 0644))

	// The 1st Stat() call (right after Add, to seed lastSize) tolerates
	// errors silently; only the 2nd (inside the write-event handler) is
	// fatal, so that's the one that needs to fail here.
	old := openFile
	openFile = func(name string) (tailFile, error) {
		real, err := old(name)
		if err != nil {
			return nil, err
		}
		return &flakyFile{f: real, failStatAt: 2}, nil
	}
	t.Cleanup(func() { openFile = old })

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		fake.events <- fsnotify.Event{Op: fsnotify.Write}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the Stat error during write handling")
}

func TestTailFileTruncationDetected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte(strings.Repeat("padding-line\n", 20)), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Truncate(path, 0)
		appendToFileNoErr(path, "after-truncate\n")
		fake.events <- fsnotify.Event{Op: fsnotify.Write}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected exactly one post-truncation line")
	if len(got) == 1 {
		testutils.Equal(t, "after-truncate", got[0], "unexpected line after truncation")
	}
}

func TestTailFileTruncationSeekFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte(strings.Repeat("padding-line\n", 20)), 0644))

	// The 1st Seek() call is the initial Seek-to-end (n=-1 startup); the
	// truncation-detected seek-to-start is the 2nd.
	old := openFile
	openFile = func(name string) (tailFile, error) {
		real, err := old(name)
		if err != nil {
			return nil, err
		}
		return &flakyFile{f: real, failSeekAt: 2}, nil
	}
	t.Cleanup(func() { openFile = old })

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Truncate(path, 0)
		appendToFileNoErr(path, "after-truncate\n")
		fake.events <- fsnotify.Event{Op: fsnotify.Write}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the Seek error during truncation handling")
}

func TestTailFileRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Rename(path, path+".1")
		_ = os.WriteFile(path, []byte("after-rotate\n"), 0644)
		fake.events <- fsnotify.Event{Op: fsnotify.Rename}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Don't break, and don't cancel synchronously: the content-reading loop
	// checks ctx.Done() again immediately after every yield, so a cancel()
	// called from directly within this callback races that same check and
	// (if it wins, which it reliably does, being on the same goroutine) exits
	// before the re-Add call below it ever runs. Schedule the cancel
	// asynchronously instead, comfortably after the synchronous work
	// (finishing the scan loop, then re-Add) has had time to complete.
	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		time.AfterFunc(20*time.Millisecond, cancel)
	}
	testutils.Equal(t, 1, len(got), "expected exactly one post-rotation line")
	if len(got) == 1 {
		testutils.Equal(t, "after-rotate", got[0], "unexpected line after rotation")
	}
	testutils.Equal(t, 2, fake.addCalls, "expected watcher.Add to be called again after rotation")
}

func TestTailFileRotationReopenFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	old := openFile
	var calls int
	openFile = func(name string) (tailFile, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("reopen failure")
		}
		return old(name)
	}
	t.Cleanup(func() { openFile = old })

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Rename(path, path+".1")
		_ = os.WriteFile(path, []byte("after-rotate\n"), 0644)
		fake.events <- fsnotify.Event{Op: fsnotify.Rename}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the reopen error after rotation")
}

func TestTailFileRotationStatFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	withFlakyOpenFile(t, 2, 0) // fail Stat on the reopened (2nd) file only
	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Rename(path, path+".1")
		_ = os.WriteFile(path, []byte("after-rotate\n"), 0644)
		fake.events <- fsnotify.Event{Op: fsnotify.Rename}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the Stat error after rotation reopen")
}

func TestTailFileRotationReAddFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	fake := newFakeWatcher()
	fake.failAddAt = 2 // 1st Add (initial attach) succeeds, 2nd (post-rotation) fails
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Rename(path, path+".1")
		_ = os.WriteFile(path, []byte("after-rotate\n"), 0644)
		fake.events <- fsnotify.Event{Op: fsnotify.Rename}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the re-Add error after rotation")
}

func TestTailFileRotationConsumerStops(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Rename(path, path+".1")
		_ = os.WriteFile(path, []byte("after-rotate\n"), 0644)
		fake.events <- fsnotify.Event{Op: fsnotify.Rename}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		break
	}
	testutils.Equal(t, 1, len(got), "expected iteration to stop right after the first post-rotation line")
}

// TestTailFileWriteEventCtxCancelledMidScan exercises the ctx.Done() check
// that runs between individual yields within one Write event's scan burst.
// This is deterministic, not racy: TailFile's iterator and this test's
// consumer run on the same goroutine (range-over-func), so a cancel() called
// synchronously from inside the first yield is guaranteed to already be
// visible to the very next ctx.Done() check, before any second line is ever
// scanned.
func TestTailFileWriteEventCtxCancelledMidScan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("first\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		appendToFileNoErr(path, "second\nthird\n")
		fake.events <- fsnotify.Event{Op: fsnotify.Write}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		cancel()
	}
	testutils.Equal(t, 1, len(got), "expected only the first of two scanned lines before ctx cancellation was observed")
	if len(got) == 1 {
		testutils.Equal(t, "second", got[0], "unexpected line")
	}
}

func TestTailFileWriteEventScanFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("first\n"), 0644))

	old := openFile
	openFile = func(name string) (tailFile, error) {
		real, err := old(name)
		if err != nil {
			return nil, err
		}
		// 1st Read is the initial n=-1 Seek's implicit read-none; reads
		// actually start once the scanner in the write-handler runs, so
		// fail on the very first Read the scanner performs.
		return &flakyFile{f: real, failReadAt: 1}, nil
	}
	t.Cleanup(func() { openFile = old })

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		appendToFileNoErr(path, "second\n")
		fake.events <- fsnotify.Event{Op: fsnotify.Write}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the scan error during write handling")
}

// TestTailFileRotationCtxCancelledMidScan is TestTailFileWriteEventCtxCancelledMidScan's
// counterpart for the rotation content-read loop.
func TestTailFileRotationCtxCancelledMidScan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Rename(path, path+".1")
		_ = os.WriteFile(path, []byte("after-rotate-1\nafter-rotate-2\n"), 0644)
		fake.events <- fsnotify.Event{Op: fsnotify.Rename}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got []string
	for line, err := range TailFile(ctx, path, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		cancel()
	}
	testutils.Equal(t, 1, len(got), "expected only the first of two post-rotation lines before ctx cancellation was observed")
	if len(got) == 1 {
		testutils.Equal(t, "after-rotate-1", got[0], "unexpected line")
	}
}

func TestTailFileRotationScanFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("before-rotate\n"), 0644))

	old := openFile
	var calls int
	openFile = func(name string) (tailFile, error) {
		calls++
		real, err := old(name)
		if err != nil {
			return nil, err
		}
		if calls == 2 {
			// Fail the reopened (post-rotation) file's first Read.
			return &flakyFile{f: real, failReadAt: 1}, nil
		}
		return real, nil
	}
	t.Cleanup(func() { openFile = old })

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	mutateAfterAttach(fake, func() {
		_ = os.Rename(path, path+".1")
		_ = os.WriteFile(path, []byte("after-rotate\n"), 0644)
		fake.events <- fsnotify.Event{Op: fsnotify.Rename}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the scan error after rotation reopen")
}

func TestFsnotifyWatcherAdapter(t *testing.T) {
	w, err := newWatcher()
	testutils.NoError(t, err)
	defer func() { _ = w.Close() }()

	testutils.NotNil(t, w.Events(), "expected a non-nil Events channel")
	testutils.NotNil(t, w.Errors(), "expected a non-nil Errors channel")
}

func TestTailFileEventsChannelClosed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("line\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	close(fake.events)

	ctx := context.Background()
	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface an error when the events channel closes")
}

func TestTailFileErrorsChannelClosed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("line\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	close(fake.errors)

	ctx := context.Background()
	var got []string
	var sawErr bool
	for line, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
			continue
		}
		got = append(got, line)
	}
	testutils.Assert(t, !sawErr, "expected no error when the errors channel simply closes")
	testutils.Equal(t, 0, len(got), "expected no lines")
}

func TestTailFileErrorsChannelDelivers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("line\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)
	fake.errors <- errors.New("watcher error")

	ctx := context.Background()
	var sawErr bool
	for _, err := range TailFile(ctx, path, -1) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the watcher's delivered error")
}

func TestTailFileContextCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("line\n"), 0644))

	fake := newFakeWatcher()
	withFakeWatcher(t, fake)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		for range TailFile(ctx, path, -1) {
		}
		close(done)
	}()

	// No events are ever sent; TailFile is blocked on its select. Give the
	// goroutine a moment to actually reach that select before canceling.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("TailFile did not stop after context cancellation")
	}
}

func TestTailFileNewWatcherFails(t *testing.T) {
	old := newWatcher
	newWatcher = func() (fsWatcher, error) { return nil, errors.New("new watcher failure") }
	t.Cleanup(func() { newWatcher = old })

	ctx := context.Background()
	var sawErr bool
	for _, err := range TailFile(ctx, "irrelevant", 0) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the watcher construction error")
}

func TestTailFileResolveAbsPathFails(t *testing.T) {
	old := resolveAbsPath
	resolveAbsPath = func(string) (string, error) { return "", errors.New("abs failure") }
	t.Cleanup(func() { resolveAbsPath = old })

	ctx := context.Background()
	var sawErr bool
	for _, err := range TailFile(ctx, "irrelevant", 0) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the resolveAbsPath error")
}

func TestTailFileInitialAddFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	testutils.NoError(t, os.WriteFile(path, []byte("a\n"), 0644))

	fake := newFakeWatcher()
	fake.failAddAt = 1
	withFakeWatcher(t, fake)

	ctx := context.Background()
	var sawErr bool
	for _, err := range TailFile(ctx, path, 0) {
		if err != nil {
			sawErr = true
		}
	}
	testutils.Assert(t, sawErr, "expected TailFile to surface the initial watcher.Add error")
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

// The following three tests exercise the ctx.Done() check that runs between
// individual yields within TailReader's various multi-line loops. This is
// deterministic, not racy: consumer and iterator run on the same goroutine
// (range-over-func), so a cancel() called synchronously from inside the
// first yield is guaranteed to already be visible to the very next
// ctx.Done() check, before a second line is ever yielded.

func TestTailReaderSeekableInitialLinesCtxCancelledMidLoop(t *testing.T) {
	r := strings.NewReader("a\nb\nc\nd\ne\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got []string
	for line, err := range TailReader(ctx, r, 3) {
		testutils.NoError(t, err)
		got = append(got, line)
		cancel()
	}
	testutils.Equal(t, 1, len(got), "expected only the first initial line before ctx cancellation was observed")
}

func TestTailReaderNonSeekableInitialLinesCtxCancelledMidLoop(t *testing.T) {
	r := nonSeekableReader{strings.NewReader("a\nb\nc\nd\ne\n")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got []string
	for line, err := range TailReader(ctx, r, 3) {
		testutils.NoError(t, err)
		got = append(got, line)
		cancel()
	}
	testutils.Equal(t, 1, len(got), "expected only the first buffered initial line before ctx cancellation was observed")
}

func TestTailReaderNonSeekableStreamCtxCancelledMidLoop(t *testing.T) {
	r := nonSeekableReader{strings.NewReader("a\nb\nc\n")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got []string
	for line, err := range TailReader(ctx, r, -1) {
		testutils.NoError(t, err)
		got = append(got, line)
		cancel()
	}
	testutils.Equal(t, 1, len(got), "expected only the first streamed line before ctx cancellation was observed")
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

func TestTailReaderSeekableStreamCtxCancelledMidLoop(t *testing.T) {
	g := &growableSeeker{data: []byte("a\nb\n")}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		g.Append("c\nd\n")
	}()

	// n=1: "b" is the initial line; "c" and "d" both become available in the
	// same growth burst. Canceling synchronously right after "c" (the first
	// streamed line) must stop the iteration before "d" is ever yielded -
	// deterministic for the same same-goroutine reason as the other
	// ctx-cancelled-mid-loop tests, once at least one streamed line exists.
	var got []string
	for line, err := range TailReader(ctx, g, 1) {
		testutils.NoError(t, err)
		got = append(got, line)
		if len(got) == 2 {
			cancel()
		}
	}
	testutils.Equal(t, 2, len(got), "expected initial line plus exactly one streamed line")
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
