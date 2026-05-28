// Package readlimit provides a hardened bounded file reader shared by all
// ecosystem scanners.
package readlimit

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ReadBounded opens path, confirms it is a regular file, and reads at most
// max bytes. It uses io.LimitReader to enforce the bound during the actual
// read rather than relying solely on a stat-before-read size comparison,
// which avoids a TOCTOU gap on files that change between stat and read.
//
// When max <= 0 the read is unbounded. When the file content exceeds max
// bytes a warning is emitted via diag (if non-nil) and an error is returned.
// The diag signature matches the ecosystem scanner convention:
//
//	func(level, path, msg string)
func ReadBounded(path string, max int64, diag func(level, path, msg string)) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	if max <= 0 {
		return io.ReadAll(f)
	}
	// Read at most max+1 bytes. If we get more than max the file exceeds
	// the limit regardless of what stat reported; the LimitReader is the
	// binding safety control here, not the stat size.
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		if diag != nil {
			diag("warn", path, fmt.Sprintf("skipping: size %d exceeds max %d", info.Size(), max))
		}
		return nil, fmt.Errorf("file exceeds max size %d bytes", max)
	}
	return data, nil
}
