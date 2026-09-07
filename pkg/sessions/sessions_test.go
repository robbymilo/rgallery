package sessions_test

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/robbymilo/rgallery/pkg/sessions"
	"github.com/robbymilo/rgallery/pkg/types"
)

func TestConcurrentSessionLifecycle(t *testing.T) {
	c := types.Conf{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	var workers sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			username := fmt.Sprintf("user-%d", worker)
			for i := 0; i < 100; i++ {
				token := fmt.Sprintf("%s-%d", username, i)
				if err := sessions.CreateSession(username, "viewer", token, time.Now().Add(time.Hour), c); err != nil {
					t.Error(err)
					return
				}
				if session, ok := sessions.GetSession(token); !ok || session.UserName != username {
					t.Error("session was lost or assigned to another user")
				}
				sessions.DeleteSession(token, c)
				sessions.DeleteUserSessions(username)
				if _, ok := sessions.GetSession(token); ok {
					t.Error("deleted session remains accessible")
				}
			}
		}(worker)
	}
	workers.Wait()
}
