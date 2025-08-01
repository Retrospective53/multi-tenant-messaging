package types

import (
	"time"

	"github.com/google/uuid"
)

type CreateTenantRequest struct {
	Name string `json:"name" binding:"required"`
}

type TenantResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
