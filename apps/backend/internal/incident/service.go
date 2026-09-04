package incident

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

var (
	ErrSatelliteNotFound    = errors.New("satellite not found")
	ErrAssignedUserNotFound = errors.New("assigned user not found")
)

type Repository interface {
	Create(context.Context, store.CreateIncidentInput) (store.Incident, error)
	ListByOrganization(context.Context, uuid.UUID, int, int, store.IncidentFilter) (store.Page[store.Incident], error)
	GetByOrganization(context.Context, uuid.UUID, uuid.UUID) (store.IncidentDetail, error)
	UpdateByOrganization(context.Context, uuid.UUID, uuid.UUID, store.UpdateIncidentInput) (store.Incident, error)
	AddNoteByOrganization(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) error
	AddEvidenceByOrganization(context.Context, uuid.UUID, uuid.UUID, store.CreateIncidentEvidenceInput) (store.IncidentEvidence, error)
}
type Satellites interface {
	Exists(context.Context, uuid.UUID) (bool, error)
}
type Users interface {
	GetByIDInOrganization(context.Context, uuid.UUID, uuid.UUID) (store.User, error)
}

type Service struct {
	incidents  Repository
	satellites Satellites
	users      Users
}

func NewService(incidents Repository, satellites Satellites, users Users) *Service {
	return &Service{incidents: incidents, satellites: satellites, users: users}
}

type CreateInput struct {
	OrganizationID                                        uuid.UUID
	ActorID                                               uuid.UUID
	Title, Description, Severity, AgentID, IdempotencyKey string
	SatelliteID, AssignedTo                               *uuid.UUID
}

func (s *Service) Create(ctx context.Context, input CreateInput) (store.Incident, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Severity = strings.ToLower(strings.TrimSpace(input.Severity))
	input.AgentID = strings.TrimSpace(input.AgentID)
	if err := validate.IncidentTitle(input.Title); err != nil {
		return store.Incident{}, err
	}
	if err := validate.IncidentDescription(input.Description); err != nil {
		return store.Incident{}, err
	}
	if err := validate.IncidentSeverity(input.Severity); err != nil {
		return store.Incident{}, err
	}
	if input.AgentID != "" {
		if err := validate.AgentID(input.AgentID); err != nil {
			return store.Incident{}, err
		}
	}
	if input.SatelliteID != nil {
		exists, err := s.satellites.Exists(ctx, *input.SatelliteID)
		if err != nil {
			return store.Incident{}, err
		}
		if !exists {
			return store.Incident{}, ErrSatelliteNotFound
		}
	}
	if input.AssignedTo != nil {
		if _, err := s.users.GetByIDInOrganization(ctx, input.OrganizationID, *input.AssignedTo); err != nil {
			if errors.Is(err, store.ErrUserNotFound) {
				return store.Incident{}, ErrAssignedUserNotFound
			}
			return store.Incident{}, err
		}
	}
	return s.incidents.Create(ctx, store.CreateIncidentInput{OrganizationID: input.OrganizationID, Title: input.Title, Description: input.Description, Severity: input.Severity, SourceType: "manual", SatelliteID: input.SatelliteID, AgentID: input.AgentID, AssignedTo: input.AssignedTo, CreatedBy: &input.ActorID, IdempotencyKey: input.IdempotencyKey})
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID, limit, offset int, filter store.IncidentFilter) (store.Page[store.Incident], error) {
	return s.incidents.ListByOrganization(ctx, organizationID, limit, offset, filter)
}

func (s *Service) Get(ctx context.Context, organizationID, incidentID uuid.UUID) (store.IncidentDetail, error) {
	return s.incidents.GetByOrganization(ctx, organizationID, incidentID)
}

func (s *Service) Update(ctx context.Context, organizationID, incidentID, actorID uuid.UUID, input store.UpdateIncidentInput) (store.Incident, error) {
	if input.Status != "" {
		input.Status = strings.ToLower(strings.TrimSpace(input.Status))
		if err := validate.IncidentStatus(input.Status); err != nil {
			return store.Incident{}, err
		}
	}
	if input.Severity != "" {
		input.Severity = strings.ToLower(strings.TrimSpace(input.Severity))
		if err := validate.IncidentSeverity(input.Severity); err != nil {
			return store.Incident{}, err
		}
	}
	if input.AssignedTo != nil {
		if _, err := s.users.GetByIDInOrganization(ctx, organizationID, *input.AssignedTo); err != nil {
			if errors.Is(err, store.ErrUserNotFound) {
				return store.Incident{}, ErrAssignedUserNotFound
			}
			return store.Incident{}, err
		}
	}
	input.ActorID = actorID
	return s.incidents.UpdateByOrganization(ctx, organizationID, incidentID, input)
}

func (s *Service) AddNote(ctx context.Context, organizationID, incidentID, actorID uuid.UUID, message string) error {
	message = strings.TrimSpace(message)
	if err := validate.IncidentNote(message); err != nil {
		return err
	}
	return s.incidents.AddNoteByOrganization(ctx, organizationID, incidentID, actorID, message)
}

func (s *Service) AddEvidence(ctx context.Context, organizationID, incidentID, actorID uuid.UUID, input store.CreateIncidentEvidenceInput) (store.IncidentEvidence, error) {
	input.EvidenceType = strings.ToLower(strings.TrimSpace(input.EvidenceType))
	input.EvidenceRef = strings.TrimSpace(input.EvidenceRef)
	input.Label = strings.TrimSpace(input.Label)
	if err := validate.IncidentEvidenceType(input.EvidenceType); err != nil {
		return store.IncidentEvidence{}, err
	}
	if err := validate.IncidentEvidenceRef(input.EvidenceRef); err != nil {
		return store.IncidentEvidence{}, err
	}
	if err := validate.IncidentEvidenceLabel(input.Label); err != nil {
		return store.IncidentEvidence{}, err
	}
	input.CreatedBy = &actorID
	return s.incidents.AddEvidenceByOrganization(ctx, organizationID, incidentID, input)
}
