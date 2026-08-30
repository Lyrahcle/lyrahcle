package auth

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestOrganizationIDRequiresTenant(t *testing.T) {
	if _, err := OrganizationID(nil); !errors.Is(err, ErrOrganizationContextRequired) {
		t.Fatalf("nil claims error = %v", err)
	}
	if _, err := OrganizationID(&Claims{}); !errors.Is(err, ErrOrganizationContextRequired) {
		t.Fatalf("empty claims error = %v", err)
	}
	want := uuid.New()
	got, err := OrganizationID(&Claims{OrganizationID: want})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
