// internal/service/message.go
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sqlc "github.com/retrospective53/multi-tenant/db/sqlc"
)

type MessageService struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewMessageService(db *sql.DB, q *sqlc.Queries) *MessageService {
	return &MessageService{db: db, queries: q}
}

func (s *MessageService) CreateMessage(ctx context.Context, tenantID uuid.UUID, payload json.RawMessage) error {
	argCreateMessageParams := sqlc.CreateMessageParams{
		ID:       uuid.New(),
		TenantID: tenantID,
		Payload:  payload,
	}
	return s.queries.CreateMessage(ctx, argCreateMessageParams)
}

func (s *MessageService) GetMessagesByTenant(ctx context.Context, tenantID uuid.UUID) ([]sqlc.Message, error) {
	return s.queries.GetMessagesByTenant(ctx, tenantID)
}

func (s *MessageService) SaveMessage(ctx context.Context, tenantID uuid.UUID, content []byte) error {
	tableName := fmt.Sprintf("messages_%s", tenantID.String())
	query := fmt.Sprintf(`INSERT INTO %s (content) VALUES ($1)`, pgx.Identifier{tableName}.Sanitize())

	_, err := s.db.Exec(query, string(content))
	return err
}
