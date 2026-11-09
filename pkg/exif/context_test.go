package exif

import (
	"context"
	"errors"
	"testing"
)

func TestGetImageExifContextChecksCancellationBeforeExifTool(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := GetImageExifContext(ctx, "image", "photo.jpg", "photo.jpg", nil, nil, Conf{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
}
