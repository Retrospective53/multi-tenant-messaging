// internal/handler/message.go
package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/retrospective53/multi-tenant/internal/service"
)

type MessageHandler struct {
	svc *service.MessageService
}

func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

type PostMessageRequest struct {
	TenantID string          `json:"tenant_id"`
	Payload  json.RawMessage `json:"payload"`
}

func (h *MessageHandler) PostMessage(c *fiber.Ctx) error {
	var req PostMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant_id"})
	}

	err = h.svc.CreateMessage(c.Context(), tenantID, req.Payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to insert message"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "ok"})
}

func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	tenantIDStr := c.Query("tenant_id")
	if tenantIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant_id is required"})
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant_id"})
	}

	messages, err := h.svc.GetMessagesByTenant(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not fetch messages"})
	}

	return c.JSON(messages)
}
