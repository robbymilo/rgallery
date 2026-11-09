package scanner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

type statErrorKind string

const (
	statNotFound   statErrorKind = "not-found"
	statPermission statErrorKind = "permission"
	statTransient  statErrorKind = "transient"
)

// classifyStatError keeps discovery errors actionable without treating every
// failed stat as a deleted file.
func classifyStatError(err error) statErrorKind {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return statNotFound
	case errors.Is(err, fs.ErrPermission):
		return statPermission
	default:
		return statTransient
	}
}

type fileSignature struct {
	modTime time.Time
	size    int64
}

func signatureFromInfo(info fs.FileInfo) fileSignature {
	return fileSignature{modTime: info.ModTime(), size: info.Size()}
}

func signatureFromDirEntry(entry fs.DirEntry) (fileSignature, error) {
	info, err := entry.Info()
	if err != nil {
		return fileSignature{}, err
	}
	return signatureFromInfo(info), nil
}

func readFileSignature(path string) (fileSignature, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileSignature{}, err
	}
	return signatureFromInfo(info), nil
}

func (s fileSignature) equal(other fileSignature) bool {
	return s.size == other.size && s.modTime.Equal(other.modTime)
}

func unstableFileError(path string, before, after fileSignature) error {
	return fmt.Errorf("file changed while being scanned: %s (before size=%d modified=%s, after size=%d modified=%s)",
		path, before.size, before.modTime.Format(time.RFC3339Nano), after.size, after.modTime.Format(time.RFC3339Nano))
}

func ensureFileStable(path string, before fileSignature) error {
	after, err := readFileSignature(path)
	if err != nil {
		return fmt.Errorf("checking file stability for %s: %w", path, err)
	}
	if !before.equal(after) {
		return unstableFileError(path, before, after)
	}
	return nil
}
