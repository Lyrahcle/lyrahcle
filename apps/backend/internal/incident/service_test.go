package incident

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

type fakeRepository struct{ created store.CreateIncidentInput }

func (f *fakeRepository) Create(_ context.Context, in store.CreateIncidentInput) (store.Incident, error) {
	f.created = in
	return store.Incident{Title: in.Title, Severity: in.Severity}, nil
}
func (*fakeRepository) ListByOrganization(context.Context, uuid.UUID, int, int, store.IncidentFilter) (store.Page[store.Incident], error) {
	return store.Page[store.Incident]{}, nil
}
func (*fakeRepository) GetByOrganization(context.Context, uuid.UUID, uuid.UUID) (store.IncidentDetail, error) {
	return store.IncidentDetail{}, nil
}
func (*fakeRepository) UpdateByOrganization(context.Context, uuid.UUID, uuid.UUID, store.UpdateIncidentInput) (store.Incident, error) {
	return store.Incident{}, nil
}
func (*fakeRepository) AddNoteByOrganization(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) error {
	return nil
}
func (*fakeRepository) AddEvidenceByOrganization(context.Context, uuid.UUID, uuid.UUID, store.CreateIncidentEvidenceInput) (store.IncidentEvidence, error) {
	return store.IncidentEvidence{}, nil
}

type fakeSatellites struct{ exists bool }

func (f fakeSatellites) Exists(context.Context, uuid.UUID) (bool, error) { return f.exists, nil }

type fakeUsers struct{}

func (fakeUsers) GetByIDInOrganization(context.Context, uuid.UUID, uuid.UUID) (store.User, error) {
	return store.User{}, nil
}

func TestCreateNormalizesAndDelegates(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, fakeSatellites{exists: true}, fakeUsers{})
	actor := uuid.New()
	got, err := service.Create(context.Background(), CreateInput{OrganizationID: uuid.New(), ActorID: actor, Title: "  Test incident  ", Description: "description", Severity: " HIGH ", AgentID: " agent-1 "})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Test incident" || repo.created.Severity != "high" || repo.created.AgentID != "agent-1" || repo.created.CreatedBy == nil || *repo.created.CreatedBy != actor {
		t.Fatalf("unexpected normalized input: %+v", repo.created)
	}
}

func TestCreateRejectsUnknownSatellite(t *testing.T) {
	service := NewService(&fakeRepository{}, fakeSatellites{}, fakeUsers{})
	satelliteID := uuid.New()
	_, err := service.Create(context.Background(), CreateInput{OrganizationID: uuid.New(), ActorID: uuid.New(), Title: "Test incident", Description: "description", Severity: "high", SatelliteID: &satelliteID})
	if err != ErrSatelliteNotFound {
		t.Fatalf("error = %v, want %v", err, ErrSatelliteNotFound)
	}
}
