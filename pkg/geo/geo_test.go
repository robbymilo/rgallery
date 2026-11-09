package geo

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robbymilo/rgallery/pkg/types"
)

func TestReverseGeocodeContextStopsCanceledRequest(t *testing.T) {
	called := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(called)
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ReverseGeocodeContext(ctx, 1, 2, types.Conf{
		LocationService: server.URL,
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
	select {
	case <-called:
		t.Fatal("canceled request reached the server")
	default:
	}
}
