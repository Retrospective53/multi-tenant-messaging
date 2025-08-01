package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/retrospective53/multi-tenant/db"
	sqlc "github.com/retrospective53/multi-tenant/db/sqlc"
	"github.com/retrospective53/multi-tenant/internal/handler"
	"github.com/retrospective53/multi-tenant/internal/mq"
	"github.com/retrospective53/multi-tenant/internal/service"
	"github.com/streadway/amqp"
)

func main() {
	app := fiber.New()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// Initialize the database
	if err := db.InitDB("postgresql://user:pass@localhost:5432/app?sslmode=disable"); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	conn := db.DB
	queries := sqlc.New(conn)

	// RabbitMQ
	mqConn, err := amqp.Dial("amqp://user:pass@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer mqConn.Close()

	transmitterMng := mq.NewTransmitterManager()
	msgService := service.NewMessageService(conn, queries)
	consumerManager := mq.NewConsumerManager(mqConn, msgService, transmitterMng)
	tenantService := service.NewTenantService(conn, queries)
	tenantHandler := handler.NewTenantHandler(tenantService, consumerManager)
	app.Post("/tenants", tenantHandler.CreateTenant)
	app.Delete("/tenants/:id", tenantHandler.DeleteTenant)
	app.Put("/tenants/:id/config/concurrency", tenantHandler.UpdateConcurrency)

	messageService := service.NewMessageService(conn, queries)
	messageHandler := handler.NewMessageHandler(messageService)
	app.Post("/messages", messageHandler.PostMessage)
	app.Get("/messages", messageHandler.GetMessages)

	// Run server in a goroutine
	go func() {
		if err := app.Listen(":3000"); err != nil {
			log.Fatalf("Fiber server failed: %v", err)
		}
	}()
	log.Println("Server is running on http://localhost:3000")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	<-ctx.Done() // Wait for CTRL+C or SIGTERM
	log.Println("Shutdown signal received")

	// stop components
	consumerManager.StopAllConsumers()
	transmitterMng.ShutdownAndWait()

	// Gracefully shutdown Fiber
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Clean up DB
	conn.Close()
	log.Println("✅ Server gracefully stopped")
}
