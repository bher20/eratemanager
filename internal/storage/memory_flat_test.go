package storage

import (
	"context"
	"testing"
)

func TestNewMemoryWithProviders_PreloadsProviders(t *testing.T) {
	ctx := context.Background()
	p := Provider{
		Key:            "testprov",
		Name:           "Test Provider",
		LandingURL:     "https://example.org",
		DefaultPDFPath: "/tmp/test.pdf",
		Notes:          "notes",
	}

	m := NewMemoryWithProviders([]Provider{p})
	defer m.Close()

	list, err := m.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(list))
	}
	if list[0].Key != p.Key || list[0].Name != p.Name {
		t.Fatalf("provider mismatch: want %+v got %+v", p, list[0])
	}
}
