package output

import (
	"bufio"
	"os"
	"sync"
)

// fileWriter is a concurrent file based output writer.
type fileWriter struct {
	file     *os.File
	writer   *bufio.Writer
	mu       sync.Mutex // Mutex for thread-safe flushing
	flushCnt int        // Counter for periodic flushing
}

// NewFileOutputWriter creates a new buffered writer for a file
func newFileOutputWriter(file string) (*fileWriter, error) {
	output, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	return &fileWriter{file: output, writer: bufio.NewWriter(output)}, nil
}

// WriteString writes an output to the underlying file
func (w *fileWriter) Write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := w.writer.Write(data)
	if err != nil {
		return err
	}
	_, err = w.writer.WriteRune('\n')

	// Periodic flush every 100 records to prevent buffer buildup on large scans
	w.flushCnt++
	if w.flushCnt >= 100 {
		w.flushCnt = 0
		if flushErr := w.writer.Flush(); flushErr != nil {
			return flushErr
		}
	}

	return err
}

// Flush flushes the underlying writer
func (w *fileWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Flush()
}

// Close closes the underlying writer flushing everything to disk
func (w *fileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.writer.Flush(); err != nil {
		return err
	}
	//nolint:errcheck // we don't care whether sync failed or succeeded.
	w.file.Sync()
	return w.file.Close()
}
