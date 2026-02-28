package storage

import (
	"context"
	"io"
	"log/slog"
)

// StreamedFileReader provides efficient streaming of large files.
type StreamedFileReader struct {
	reader io.ReadCloser
	log    *slog.Logger
}

// NewStreamedFileReader creates a new streaming file reader
func NewStreamedFileReader(reader io.ReadCloser, log *slog.Logger) *StreamedFileReader {
	return &StreamedFileReader{reader: reader, log: log}
}

// StreamToWriter efficiently streams file content to an output writer with buffering.
func (sfr *StreamedFileReader) StreamToWriter(ctx context.Context, writer io.Writer, bufferSize int) (int64, error) {
	// Use a 32KB buffer by default (balance between memory and throughput)
	if bufferSize <= 0 {
		bufferSize = 32 * 1024
	}

	// Create a buffered pipe to avoid loading entire file into memory
	buffer := make([]byte, bufferSize)
	var totalWritten int64

	for {
		select {
		case <-ctx.Done():
			return totalWritten, ctx.Err()
		default:
			// Read a chunk of the file
			n, err := sfr.reader.Read(buffer)
			if n > 0 {
				// Write the chunk to the destination
				written, writeErr := writer.Write(buffer[:n])
				totalWritten += int64(written)
				if writeErr != nil {
					sfr.log.Error("failed to write chunk", "error", writeErr, "chunk_size", n)
					return totalWritten, writeErr
				}
			}

			if err != nil {
				if err == io.EOF {
					return totalWritten, nil // Successful completion
				}
				sfr.log.Error("failed to read chunk", "error", err)
				return totalWritten, err
			}
		}
	}
}

// Close closes the underlying reader
func (sfr *StreamedFileReader) Close() error {
	if sfr.reader != nil {
		return sfr.reader.Close()
	}
	return nil
}
