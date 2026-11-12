package plugin

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

type stubPlugin struct {
	typ string
}

func (s *stubPlugin) Type() string                                              { return s.typ }
func (s *stubPlugin) Models() []any                                             { return nil }
func (s *stubPlugin) Configure(_ *config.EdgeConfig)                            {}
func (s *stubPlugin) RegisterRoutes(_ *gin.RouterGroup, _ *gorm.DB)             {}
func (s *stubPlugin) OnPLCEvent(_ string, _ map[string]any) *types.EdgeEvent    { return nil }
func (s *stubPlugin) SyncSpec() SyncSpec                                        { return SyncSpec{} }
func (s *stubPlugin) OnCommand(_ types.Command) error                           { return nil }

func TestRegisterAndGet(t *testing.T) {
	p := &stubPlugin{typ: "test-alpha"}
	Register(p)

	got, err := Get("test-alpha")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got != p {
		t.Fatal("Get returned different instance")
	}
}

func TestGet_UnknownType(t *testing.T) {
	_, err := Get("nonexistent-type-xyz")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestAll_ContainsRegistered(t *testing.T) {
	p1 := &stubPlugin{typ: "test-beta"}
	p2 := &stubPlugin{typ: "test-gamma"}
	Register(p1)
	Register(p2)

	all := All()
	found := map[string]bool{}
	for _, p := range all {
		found[p.Type()] = true
	}
	if !found["test-beta"] {
		t.Error("test-beta not found in All()")
	}
	if !found["test-gamma"] {
		t.Error("test-gamma not found in All()")
	}
}

func TestRegister_OverwritesSameType(t *testing.T) {
	p1 := &stubPlugin{typ: "test-delta"}
	p2 := &stubPlugin{typ: "test-delta"}
	Register(p1)
	Register(p2)

	got, err := Get("test-delta")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got != p2 {
		t.Fatal("expected second registered plugin to win")
	}
}
