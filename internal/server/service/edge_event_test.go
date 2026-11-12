package service

import (
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	logger.Init("debug")
	os.Exit(m.Run())
}

func testEventDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.EdgeEventRecord{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func makeEvent(edgeID, eventID string, seq int64) types.EdgeEvent {
	return types.EdgeEvent{
		EdgeID:    edgeID,
		EventID:   eventID,
		Seq:       seq,
		EventType: "doffing",
		Timestamp: time.Now(),
		Payload:   map[string]any{"line_code": "L01", "doff_no": 42},
	}
}

func TestIngest_FirstTimeIsNew(t *testing.T) {
	svc := NewEdgeEventService(testEventDB(t))

	rec, isNew, err := svc.Ingest(makeEvent("edge-1", "evt-1", 1))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if !isNew {
		t.Fatal("first ingest should be new")
	}
	if rec.EdgeID != "edge-1" || rec.EventID != "evt-1" {
		t.Fatalf("unexpected record: %+v", rec)
	}
	if rec.Status != "received" {
		t.Fatalf("status = %q, want received", rec.Status)
	}
}

func TestIngest_DuplicateIsIgnored(t *testing.T) {
	svc := NewEdgeEventService(testEventDB(t))

	if _, isNew, err := svc.Ingest(makeEvent("edge-1", "evt-1", 1)); err != nil || !isNew {
		t.Fatalf("first ingest: isNew=%v err=%v", isNew, err)
	}
	rec, isNew, err := svc.Ingest(makeEvent("edge-1", "evt-1", 1))
	if err != nil {
		t.Fatalf("duplicate ingest: %v", err)
	}
	if isNew {
		t.Fatal("duplicate ingest must not be new")
	}
	if rec.EventID != "evt-1" {
		t.Fatalf("duplicate should return existing record, got %+v", rec)
	}

	n, err := svc.Count("edge-1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1 (duplicate must not create a row)", n)
	}
}

func TestIngest_SameEventIDDifferentEdge(t *testing.T) {
	svc := NewEdgeEventService(testEventDB(t))

	if _, isNew, _ := svc.Ingest(makeEvent("edge-1", "evt-1", 1)); !isNew {
		t.Fatal("edge-1 should be new")
	}
	if _, isNew, _ := svc.Ingest(makeEvent("edge-2", "evt-1", 1)); !isNew {
		t.Fatal("edge-2 with same event_id must be independent and new")
	}
	if n, _ := svc.Count(""); n != 2 {
		t.Fatalf("total count = %d, want 2", n)
	}
}

func TestIngest_FallsBackToSeq(t *testing.T) {
	svc := NewEdgeEventService(testEventDB(t))

	rec, isNew, err := svc.Ingest(makeEvent("edge-1", "", 7))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if !isNew {
		t.Fatal("should be new")
	}
	if rec.EventID != "edge-1:7" {
		t.Fatalf("event_id = %q, want edge-1:7", rec.EventID)
	}

	// 同样的 seq 再来一次 → 幂等
	if _, isNew, _ := svc.Ingest(makeEvent("edge-1", "", 7)); isNew {
		t.Fatal("same seq must be treated as duplicate")
	}
}

func TestIngest_RejectsMissingKey(t *testing.T) {
	svc := NewEdgeEventService(testEventDB(t))

	if _, _, err := svc.Ingest(makeEvent("edge-1", "", 0)); err == nil {
		t.Fatal("expected error for missing event_id and seq")
	}
}

func TestIngest_RejectsMissingEdgeID(t *testing.T) {
	svc := NewEdgeEventService(testEventDB(t))

	if _, _, err := svc.Ingest(makeEvent("", "evt-1", 1)); err == nil {
		t.Fatal("expected error for missing edge_id")
	}
}

func TestMarkProcessed(t *testing.T) {
	svc := NewEdgeEventService(testEventDB(t))

	rec, _, err := svc.Ingest(makeEvent("edge-1", "evt-1", 1))
	if err != nil {
		t.Fatal(err)
	}
	svc.MarkProcessed(rec, "failed", errTestSentinel)

	// 通过再次读取验证落库
	var got model.EdgeEventRecord
	if err := svc.db.First(&got, rec.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" {
		t.Fatalf("status = %q, want failed", got.Status)
	}
	if got.ProcessedAt == nil {
		t.Fatal("processed_at should be set")
	}
	if got.Error == "" {
		t.Fatal("error should be recorded")
	}
}

var errTestSentinel = &testError{}

type testError struct{}

func (*testError) Error() string { return "boom" }
