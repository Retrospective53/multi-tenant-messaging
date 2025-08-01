package handler

import (
	"log"

	"github.com/google/uuid"
	"github.com/retrospective53/multi-tenant/internal/mq"
	"github.com/retrospective53/multi-tenant/internal/service"
	"github.com/retrospective53/multi-tenant/internal/types"

	"github.com/gofiber/fiber/v2"
)

type TenantHandler struct {
	svc *service.TenantService
	cm  *mq.ConsumerManager
}

func NewTenantHandler(svc *service.TenantService, cm *mq.ConsumerManager) *TenantHandler {
	return &TenantHandler{svc: svc, cm: cm}
}

func (h *TenantHandler) CreateTenant(c *fiber.Ctx) error {
	var req types.CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	tenant, err := h.svc.CreateTenant(c.Context(), req.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// internal/handler/tenant.go (inside CreateTenant)
	err = h.cm.StartConsumer(tenant.ID)
	if err != nil {
		log.Printf("Failed to start consumer for tenant: %v", err)
	}

	return c.Status(fiber.StatusCreated).JSON(tenant)
}

func (h *TenantHandler) DeleteTenant(c *fiber.Ctx) error {
	tenantIDStr := c.Params("id")
	tenantID, err := types.ParseUUID(tenantIDStr) // wrap uuid.Parse
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant ID"})
	}

	// Stop RabbitMQ consumer
	if err := h.cm.StopConsumer(tenantID); err != nil {
		log.Printf("Failed to stop consumer: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to stop consumer"})
	}

	// // Delete tenant from database
	// if err := h.svc.DeleteTenant(c.Context(), tenantID); err != nil {
	// 	log.Printf("Failed to delete tenant: %v", err)
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete tenant"})
	// }

	return c.SendStatus(fiber.StatusNoContent)
}

// PUT /tenants/:id/config/concurrency
func (h *TenantHandler) UpdateConcurrency(c *fiber.Ctx) error {
	tenantIDStr := c.Params("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant ID"})
	}

	var req struct {
		Workers int `json:"workers"`
	}
	if err := c.BodyParser(&req); err != nil || req.Workers <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid workers value"})
	}

	err = h.cm.UpdateConcurrency(tenantID, req.Workers)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}
