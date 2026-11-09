package scanner

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMediaExtensionsAreCaseInsensitive(t *testing.T) {
	for _, path := range []string{"photo.JpG", "photo.JPEG", "photo.HeIc", "photo.GiF", "photo.PnG"} {
		if !isImage(path) {
			t.Errorf("isImage(%q) = false, want true", path)
		}
	}
	for _, path := range []string{"clip.Mp4", "clip.MoV"} {
		if !isVideo(path) {
			t.Errorf("isVideo(%q) = false, want true", path)
		}
	}
	if isImage(".JPG") || isVideo(".MOV") {
		t.Error("hidden extension-only files must remain unsupported")
	}
}

func TestClassifyStatError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want statErrorKind
	}{
		{name: "not found", err: fs.ErrNotExist, want: statNotFound},
		{name: "wrapped not found", err: errors.New("walk: file does not exist"), want: statTransient},
		{name: "permission", err: fs.ErrPermission, want: statPermission},
		{name: "transient", err: errors.New("input/output error"), want: statTransient},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyStatError(tt.err); got != tt.want {
				t.Fatalf("classifyStatError(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestMediaModifiedHandlesMissingFileInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	c := Conf{Logger: logger}
	if mediaModified(Media{Path: "missing.jpg"}, nil, c) {
		t.Fatal("mediaModified returned true without file info")
	}
}

func TestFileSignatureIncludesNanosecondsAndSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(path, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := readFileSignature(path)
	if err != nil {
		t.Fatal(err)
	}

	secondTime := first.modTime.Add(1 * time.Nanosecond)
	if err := os.Chtimes(path, secondTime, secondTime); err != nil {
		t.Fatal(err)
	}
	second, err := readFileSignature(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.equal(second) {
		t.Fatal("nanosecond modification change was not detected")
	}

	if err := os.WriteFile(path, []byte("different size"), 0o600); err != nil {
		t.Fatal(err)
	}
	third, err := readFileSignature(path)
	if err != nil {
		t.Fatal(err)
	}
	if second.equal(third) {
		t.Fatal("size change was not detected")
	}
}

func TestEnsureFileStableDetectsUnstableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "copying.jpg")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := readFileSignature(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after metadata extraction"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = ensureFileStable(path, before)
	if err == nil || !strings.Contains(err.Error(), "file changed while being scanned") {
		t.Fatalf("ensureFileStable error = %v, want unstable-file error", err)
	}
}

func TestEnsureFileStableHandlesDisappearedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "disappearing.jpg")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := readFileSignature(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := ensureFileStable(path, before); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ensureFileStable error = %v, want not-exist error", err)
	}
}
