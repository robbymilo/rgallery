//go:build windows

package transcode

import (
	"context"
	"fmt"
)

// fileLock reports that video cache locking is unsupported on Windows.
func fileLock(context.Context, string, bool, bool) (func(), error) {
	return nil, fmt.Errorf("video cache locking requires Linux or macOS")
}
