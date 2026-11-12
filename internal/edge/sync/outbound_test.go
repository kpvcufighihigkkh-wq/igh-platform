package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/igh-platform/igh-platform/internal/edge/handler"
	"github.com/igh-platform/igh-platform/internal/edge/queue"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	logger.Init("debug")
	os.Exit(m.Run())
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB, _ := db.DB(); sqlDB.Close() })
	return db
}

func newTestOutbound(t *testing.T, serverURL string) (*Outbound, *queue.Queue) {
	t.Helper()
	q := queue.New(testDB(t))
	tr := handler.NewTransport(&config.EdgeConfig{
		Edge: config.EdgeSection{
			ID:     "test-edge",
			Server: config.EdgeServer{URL: serverURL, APIKey: "sk-test"},
		},
	})
	return NewOutbound(tr, q), q
}

func TestFlush_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call server on empty flush")
	}))
	defer srv.Close()

	ob, _ := newTestOutbound(t, srv.URL)
	n := ob.Flush()
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestFlush_Success(t *testing.T) {
	var receivedEvents []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		receivedEvents, _ = body["events"].([]any)
		w.WriteHeader(200)
		w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	ob, q := newTestOutbound(t, srv.URL)
	q.Enqueue(types.EdgeEvent{EdgeID: "e1", EventType: "test", Timestamp: time.Now(), Payload: map[string]any{"x": 1}})
	q.Enqueue(types.EdgeEvent{EdgeID: "e1", EventType: "test", Timestamp: time.Now(), Payload: map[string]any{"x": 2}})

	n := ob.Flush()
	if n != 2 {
		t.Fatalf("expected 2 flushed, got %d", n)
	}
	if len(receivedEvents) != 2 {
		t.Fatalf("server got %d events", len(receivedEvents))
	}
	if q.Depth() != 0 {
		t.Fatalf("queue should be empty after successful flush, depth = %d", q.Depth())
	}
}

func TestFlush_Failure_ReQueues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	srv.Close() // close immediately to simulate unreachable server

	ob, q := newTestOutbound(t, srv.URL)
	q.Enqueue(types.EdgeEvent{EdgeID: "e1", EventType: "test", Timestamp: time.Now(), Payload: map[string]any{"x": 1}})

	n := ob.Flush()
	if n != 0 {
		t.Fatalf("expected 0 on failure, got %d", n)
	}
	if d := q.Depth(); d != 1 {
		t.Fatalf("expected re-queued event, depth = %d", d)
	}
}
