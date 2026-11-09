package scanner

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/robbymilo/rgallery/pkg/database"
)

func TestSuccessfulRetryClearsScanError(t *testing.T) {
	c := Conf{
		Data:   filepath.Join(t.TempDir(), "data"),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	database.CreateDB(c)

	path := "photos/retry.jpg"
	if err := TrackScanError(path, time.Now(), assertErr("first attempt"), c); err != nil {
		t.Fatal(err)
	}
	before, err := GetScanErrors(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := before[path]; !ok {
		t.Fatalf("scan error for %q was not recorded", path)
	}

	if err := ClearScanError(path, c); err != nil {
		t.Fatal(err)
	}
	after, err := GetScanErrors(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := after[path]; ok {
		t.Fatalf("scan error for %q remained after successful retry", path)
	}
}

func TestScanErrorsCanBePagedAndResetAll(t *testing.T) {
	c := Conf{
		Data:   filepath.Join(t.TempDir(), "data"),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	database.CreateDB(c)
	for _, path := range []string{"photos/b.jpg", "photos/a.jpg"} {
		if err := TrackScanError(path, time.Now(), assertErr("bad file"), c); err != nil {
			t.Fatal(err)
		}
	}

	page, err := GetScanErrorsPage(c, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Path != "photos/a.jpg" {
		t.Fatalf("unexpected first page: %+v", page)
	}

	if err := ResetAllScanErrors(c); err != nil {
		t.Fatal(err)
	}
	page, err = GetScanErrorsPage(c, 1, 25)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("scan errors remained after reset all: %+v", page)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
