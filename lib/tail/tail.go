package tail

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TailFile watches the specified file for changes, similar to 'tail -f', and streams
// new lines to a channel. It handles file rotation by detecting renames and re-opening
// the new file. The function stops when the context is canceled. The parameter n specifies
// the number of most recent lines to read initially (default 10). If n = -1, no initial
// lines are read, and only new lines are streamed. The returned channel receives lines
// as strings, and any error during watching is sent to the error channel.
func TailFile(ctx context.Context, filePath string, n int) (<-chan string, <-chan error, error) {
	lineCh := make(chan string, 100) // Buffered to prevent blocking
	errCh := make(chan error, 1)

	// Initialize file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	// Resolve absolute path to handle renames
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		_ = watcher.Close()
		return nil, nil, err
	}

	// Open the file
	file, err := os.Open(absPath)
	if err != nil {
		_ = watcher.Close()
		return nil, nil, err
	}

	// Read initial lines if n >= 0
	if n >= 0 {
		if n == 0 {
			n = 10 // Default to 10 lines
		}
		lines, err := readLastLines(file, n)
		if err != nil {
			_ = file.Close()
			_ = watcher.Close()
			return nil, nil, err
		}
		// Send initial lines asynchronously
		go func() {
			for _, line := range lines {
				select {
				case lineCh <- line:
				case <-ctx.Done():
					_ = file.Close()
					_ = watcher.Close()
					return
				}
			}
		}()
	} else {
		// Seek to the end if n = -1
		_, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			_ = file.Close()
			_ = watcher.Close()
			return nil, nil, err
		}
	}

	// Add file to watcher
	err = watcher.Add(absPath)
	if err != nil {
		_ = file.Close()
		_ = watcher.Close()
		return nil, nil, err
	}

	// Start goroutine to handle file watching and reading
	go func() {
		defer func() {
			_ = watcher.Close()
			close(lineCh)
			close(errCh)
		}()

		currentFile := file
		lastWrite := time.Now()
		var lastSize int64 // Track file size

		// Get initial file size
		if info, err := currentFile.Stat(); err == nil {
			lastSize = info.Size()
		}

		for {
			select {
			case <-ctx.Done():
				_ = currentFile.Close()
				return
			case event, ok := <-watcher.Events:
				if !ok {
					_ = currentFile.Close()
					errCh <- fsnotify.ErrEventOverflow
					return
				}
				if event.Has(fsnotify.Write) {
					// Debounce rapid writes
					if time.Since(lastWrite) < 10*time.Millisecond {
						continue
					}
					lastWrite = time.Now()

					// Check for truncation by comparing current file size
					info, err := currentFile.Stat()
					if err != nil {
						_ = currentFile.Close()
						errCh <- err
						return
					}
					currentSize := info.Size()
					if currentSize < lastSize {
						// Truncation detected: seek to start
						_, err := currentFile.Seek(0, io.SeekStart)
						if err != nil {
							_ = currentFile.Close()
							errCh <- err
							return
						}
					}
					lastSize = currentSize

					// Read new lines
					scanner := bufio.NewScanner(currentFile)
					for scanner.Scan() {
						select {
						case lineCh <- scanner.Text():
						case <-ctx.Done():
							_ = currentFile.Close()
							return
						}
					}
					if err := scanner.Err(); err != nil && err != io.EOF {
						_ = currentFile.Close()
						errCh <- err
						return
					}
					// Update position
					_, err = currentFile.Seek(0, io.SeekCurrent)
					if err != nil {
						_ = currentFile.Close()
						errCh <- err
						return
					}
				}
				if event.Has(fsnotify.Rename | fsnotify.Remove) {
					// Handle log rotation: re-open the new file
					_ = currentFile.Close()
					newFile, err := os.Open(absPath)
					if err != nil {
						errCh <- err
						return
					}
					currentFile = newFile
					// Seek to end and update size
					info, err := currentFile.Stat()
					if err != nil {
						_ = currentFile.Close()
						errCh <- err
						return
					}
					lastSize = info.Size()
					_, err = currentFile.Seek(0, io.SeekEnd)
					if err != nil {
						_ = currentFile.Close()
						errCh <- err
						return
					}
					err = watcher.Add(absPath)
					if err != nil {
						_ = currentFile.Close()
						errCh <- err
						return
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					_ = currentFile.Close()
					return
				}
				errCh <- err
				return
			}
		}
	}()

	return lineCh, errCh, nil
}

// TailReader tails lines from an io.Reader, streaming new lines to a channel.
// If the reader implements io.ReadSeeker, it seeks to read the last n lines efficiently.
// Otherwise, it buffers lines in memory. The parameter n specifies the number of most
// recent lines to send initially (default 10). If n = -1, no initial lines are sent.
func TailReader(ctx context.Context, r io.Reader, n int) (<-chan string, <-chan error, error) {
	lineCh := make(chan string, 100) // Buffered to prevent blocking
	errCh := make(chan error, 1)

	if n == 0 {
		n = 10 // Default to 10 lines
	}

	// Check if the reader is an io.ReadSeeker
	if rs, ok := r.(io.ReadSeeker); ok && n >= 0 {
		// Handle io.ReadSeeker (e.g., *os.File)
		lines, err := readLastLines(rs, n)
		if err != nil {
			return nil, nil, err
		}
		go func() {
			defer close(lineCh)
			defer close(errCh)

			// Send initial lines
			for _, line := range lines {
				select {
				case lineCh <- line:
				case <-ctx.Done():
					return
				}
			}

			// Stream new lines
			scanner := bufio.NewScanner(rs)
			for scanner.Scan() {
				select {
				case lineCh <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				errCh <- err
			}
		}()
		return lineCh, errCh, nil
	}

	// Handle non-seekable io.Reader
	go func() {
		defer close(lineCh)
		defer close(errCh)

		scanner := bufio.NewScanner(r)
		var lines []string
		if n >= 0 {
			// Buffer last n lines in memory
			lines = slices.Grow(lines, n) // Pre-allocate for efficiency
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
				if len(lines) > n {
					lines = lines[1:] // Slide window
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				errCh <- err
				return
			}
			// Send initial lines
			for _, line := range lines {
				select {
				case lineCh <- line:
				case <-ctx.Done():
					return
				}
			}
		} else {
			// Stream new lines only (n = -1)
			for scanner.Scan() {
				select {
				case lineCh <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				errCh <- err
			}
		}
	}()

	return lineCh, errCh, nil
}

// readLastLines reads the last n lines from an io.ReadSeeker. It seeks to the end
// and scans backward to collect lines, restoring the seeker to the end afterward.
// It assumes the seeker is positioned arbitrarily.
func readLastLines(rs io.ReadSeeker, n int) ([]string, error) {
	// Seek to end to get file size
	endPos, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if endPos == 0 {
		return nil, nil // Empty file
	}

	var lines []string
	buf := make([]byte, 4096) // 4KB buffer for reading backward
	pos := endPos
	remainder := ""

	for pos > 0 && len(lines) < n {
		// Determine how much to read (up to buffer size)
		readSize := int64(len(buf))
		readSize = min(readSize, pos)

		pos -= readSize

		// Seek to position
		_, err := rs.Seek(pos, io.SeekStart)
		if err != nil {
			return nil, err
		}

		// Read chunk
		nRead, err := rs.Read(buf[:readSize])
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Process chunk in reverse
		chunk := string(buf[:nRead])
		linesChunk := strings.Split(remainder+chunk, "\n")
		remainder = linesChunk[0]  // Partial line at start
		newLines := linesChunk[1:] // Complete lines

		// Prepend new lines (since reading backward)
		for i := len(newLines) - 1; i >= 0 && len(lines) < n; i-- {
			if newLines[i] != "" { // Skip empty lines from split
				lines = append([]string{newLines[i]}, lines...)
			}
		}
	}

	// Handle case where file has fewer lines than n
	if remainder != "" && len(lines) < n {
		lines = append([]string{remainder}, lines...)
	}

	// Seek back to end
	_, err = rs.Seek(endPos, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Reverse lines to correct order (since we read backward)
	slices.Reverse(lines)
	return lines[:min(len(lines), n)], nil
}
