package rates

import (
	"context"
	"testing"
)

// TestGetElectricResidential_UnknownProvider ensures unknown providers return an error.
func TestGetElectricResidential_UnknownProvider(t *testing.T) {
	svc := NewService(Config{})
	ctx := context.Background()

	if _, err := svc.GetElectricResidential(ctx, "unknown"); err == nil {
		t.Fatalf("expected error for unknown provider")
	}
}
