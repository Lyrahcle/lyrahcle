package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
)

func TestFromContextAddsRequestAndIdentityFields(t *testing.T) {
	var output bytes.Buffer
	Configure(&output)
	t.Cleanup(func() { Configure(&bytes.Buffer{}) })

	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "req-123")
	ctx = auth.WithClaims(ctx, &auth.Claims{UserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), OrganizationID: uuid.MustParse("22222222-2222-2222-2222-222222222222")})
	FromContext(ctx).Info("test_event", "result", "ok")

	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("decode JSON log: %v", err)
	}
	for key, want := range map[string]string{
		"msg": "test_event", "request_id": "req-123", "user_id": "11111111-1111-1111-1111-111111111111", "tenant_id": "22222222-2222-2222-2222-222222222222", "result": "ok",
	} {
		if got := event[key]; got != want {
			t.Errorf("%s = %v, want %s", key, got, want)
		}
	}
}
