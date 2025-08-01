package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	db "github.com/retrospective53/multi-tenant/db/sqlc"
	"github.com/retrospective53/multi-tenant/internal/types"

	"github.com/google/uuid"
)

type TenantService struct {
	queries *db.Queries
	db      *sql.DB
}

func NewTenantService(db *sql.DB, queries *db.Queries) *TenantService {
	return &TenantService{
		db:      db,
		queries: queries,
	}
}

func (s *TenantService) CreateTenant(ctx context.Context, name string) (*types.TenantResponse, error) {
	id := uuid.New()

	tenant, err := s.queries.CreateTenant(ctx, db.CreateTenantParams{
		ID:   id,
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	// Create partition for this tenant
	err = s.createTenantPartition(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant partition: %w", err)
	}

	var createdAt *time.Time
	if tenant.CreatedAt.Valid {
		createdAt = &tenant.CreatedAt.Time
	}

	return &types.TenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		CreatedAt: *createdAt,
	}, nil
}

func (s *TenantService) createTenantPartition(ctx context.Context, tenantID uuid.UUID) error {
	cleanTenantID := strings.ReplaceAll(tenantID.String(), "-", "")
	tableName := fmt.Sprintf("messages_tenant_%s", cleanTenantID)

	sql := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s PARTITION OF messages
        FOR VALUES IN ('%s');
    `, tableName, tenantID.String())

	_, err := s.db.ExecContext(ctx, sql)
	return err
}
