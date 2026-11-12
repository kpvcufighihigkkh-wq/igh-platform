package queue

import (
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
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
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func TestNew_EmptyQueue(t *testing.T) {
	q := New(testDB(t))
	if d := q.Depth(); d != 0 {
		t.Fatalf("expected empty queue, got depth %d", d)
	}
}

func TestEnqueue_IncreasesDepth(t *testing.T) {
	q := New(testDB(t))
	q.Enqueue(types.EdgeEvent{
		EdgeID:    "e1",
		EventType: "doffing",
		Timestamp: time.Now(),
		Payload:   map[string]any{"line": "L01"},
	})
	if d := q.Depth(); d != 1 {
		t.Fatalf("expected depth 1, got %d", d)
	}
}

func TestDrain_ReturnsAllAndClearsQueue(t *testing.T) {
	q := New(testDB(t))
	for i := 0; i < 3; i++ {
		q.Enqueue(types.EdgeEvent{
			EdgeID:    "e1",
			EventType: "doffing",
			Timestamp: time.Now(),
			Payload:   map[string]any{"i": i},
		})
	}

	events := q.Drain()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if d := q.Depth(); d != 0 {
		t.Fatalf("expected empty after drain, got %d", d)
	}
}

func TestDrain_PreservesOrder(t *testing.T) {
	q := New(testDB(t))
	for i := 0; i < 5; i++ {
		q.Enqueue(types.EdgeEvent{
			EdgeID:    "e1",
			EventType: "test",
			Timestamp: time.Now(),
			Payload:   map[string]any{"seq": float64(i)},
		})
	}
	events := q.Drain()
	for i, e := range events {
		seq, _ := e.Payload["seq"].(float64)
		if int(seq) != i {
			t.Fatalf("event %d has seq %v", i, seq)
		}
	}
}

func TestDrain_Empty(t *testing.T) {
	q := New(testDB(t))
	events := q.Drain()
	if events != nil {
		t.Fatalf("expected nil for empty drain, got %v", events)
	}
}

func TestEnqueue_AfterClose_Noop(t *testing.T) {
	q := New(testDB(t))
	q.Close()
	q.Enqueue(types.EdgeEvent{
		EdgeID:    "e1",
		EventType: "test",
		Timestamp: time.Now(),
		Payload:   map[string]any{},
	})
	if d := q.Depth(); d != 0 {
		t.Fatalf("expected 0 after closed enqueue, got %d", d)
	}
}

func TestDrain_Limit500(t *testing.T) {
	q := New(testDB(t))
	for i := 0; i < 510; i++ {
		q.Enqueue(types.EdgeEvent{
			EdgeID:    "e1",
			EventType: "test",
			Timestamp: time.Now(),
			Payload:   map[string]any{},
		})
	}
	events := q.Drain()
	if len(events) != 500 {
		t.Fatalf("expected 500 (batch limit), got %d", len(events))
	}
	if d := q.Depth(); d != 10 {
		t.Fatalf("expected 10 remaining, got %d", d)
	}
}

func TestEnqueue_PayloadRoundTrip(t *testing.T) {
	q := New(testDB(t))
	q.Enqueue(types.EdgeEvent{
		EdgeID:    "edge-01",
		EventType: "alarm",
		Timestamp: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
		Payload:   map[string]any{"level": "critical", "code": float64(42)},
	})

	events := q.Drain()
	if len(events) != 1 {
		t.Fatal("expected 1 event")
	}
	e := events[0]
	if e.EdgeID != "edge-01" {
		t.Errorf("EdgeID = %q", e.EdgeID)
	}
	if e.EventType != "alarm" {
		t.Errorf("EventType = %q", e.EventType)
	}
	if e.Payload["level"] != "critical" {
		t.Errorf("payload level = %v", e.Payload["level"])
	}
	if e.Payload["code"] != float64(42) {
		t.Errorf("payload code = %v", e.Payload["code"])
	}
}
