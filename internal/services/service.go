package services

import (
	"context"
	"errors"
	"fmt"
	clients "hephaestus/internal/clients"
	models "hephaestus/internal/models"
	repositories "hephaestus/internal/repositories"
	utils "hephaestus/internal/utils"
	"time"

	"github.com/jackc/pgx/v5"
)

type ServiceInterface interface {
	Validate(token string) (string, error)
	GetDomains(filters models.GetDomainsReq) (models.GetDomainsResp, error)
	CreateDomain(req models.CreateDomainReq) (string, error)
	DeleteDomain(filters models.DeleteDomainReq) error
}

type Service struct {
	client     []*clients.Client
	repository *repositories.Repository
	log        *utils.Logger
	cfg        *utils.Config
	ctx        context.Context
}

func NewService(cfg *utils.Config, clientsList []*clients.Client, repo *repositories.Repository, log *utils.Logger) (*Service, error) {
	ctx := context.Background()

	return &Service{
		client:     clientsList,
		repository: repo,
		log:        log,
		cfg:        cfg,
		ctx:        ctx,
	}, nil
}

func (s *Service) SelectClientByName(name string) (*clients.Client, error) {
	if name == "no" {
		return s.client[0], nil
	}
	for _, client := range s.client {
		// fmt.Printf("Component: %s ", name)
		// fmt.Printf("Client: %s ", client.Name)
		if client.Name == name {
			return client, nil
		}
	}
	return nil, errors.New("client not found")
}

func NewEntity(table string, params map[string]any) models.Entity {
	e := models.Entity{
		EntityName:        table,
		StringParameters:  map[string]string{},
		IntegerParameters: map[string]int{},
		TimeParameters:    map[string]time.Time{},
		BoolParameters:    map[string]bool{},
	}

	for k, v := range params {
		switch val := v.(type) {
		case string:
			e.StringParameters[k] = val
		case int:
			e.IntegerParameters[k] = val
		case time.Time:
			e.TimeParameters[k] = val
		case bool:
			e.BoolParameters[k] = val
		default:
			panic(fmt.Sprintf("unsupported type for key %s: %T", k, val))
		}
	}
	return e
}

func (s *Service) updateStatus(
	ctx context.Context, tx pgx.Tx, ID any, tableName, status, updatedBy string,
) error {
	var ids []string

	switch v := ID.(type) {
	case string:
		ids = []string{v}

	case []string:
		ids = v

	default:
		return fmt.Errorf("unsupported ID type: %T", ID)
	}

	entity := NewEntity(tableName, map[string]any{
		"status":     status,
		"updated_by": updatedBy,
	})
	for _, id := range ids {
		if err := s.repository.UpdateTx(ctx, tx, entity, id); err != nil {
			return fmt.Errorf("failed to update %s with id %s: %w", tableName, id, err)
		}
	}

	return nil
}

func (s *Service) safeWriteEvent(user string, domainID string, eventType string, details string) error {
	err := s.writeEvent(s.ctx, nil, domainID, eventType, details, user)
	if err != nil {
		s.log.Error("event writing failed (non-fatal):", err)
	}
	return err
}

func (s *Service) writeEvent(
	ctx context.Context,
	tx pgx.Tx,
	domainID string,
	eventType string,
	message string,
	createdBy string,
) error {

	entity := NewEntity("events", map[string]any{
		"domain_id":  domainID,
		"event_type": eventType,
		"message":    message,
		"created_by": createdBy,
	})

	if _, err := s.repository.InsertTx(ctx, tx, entity); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

func (s *Service) insertMany(
	ctx context.Context,
	tx pgx.Tx,
	entities []models.Entity,
) ([]string, error) {

	ids := make([]string, 0, len(entities))

	for _, e := range entities {
		id, err := s.repository.InsertTx(ctx, tx, e)
		if err != nil {
			return nil, fmt.Errorf("failed to insert into %s: %w", e.EntityName, err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (s *Service) updateMany(
	ctx context.Context,
	tx pgx.Tx,
	updateData map[string]models.Entity,
) error {
	for id, entity := range updateData {

		if err := s.repository.UpdateTx(ctx, tx, entity, id); err != nil {
			return fmt.Errorf(
				"failed to update %s id=%s: %w",
				entity.EntityName, id, err,
			)
		}
	}

	return nil
}
