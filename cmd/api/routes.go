package main

import (
	"go-api/cmd/api/di"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
)

func setupRoutes(app *fiber.App, container *di.Container) {
	setupHealthChecks(app)
	setupWebhooks(app, container)
	setupAPIRoutes(app, container)
}

func setupWebhooks(app *fiber.App, container *di.Container) {
	webhooks := app.Group("/webhooks")
	webhooks.Post("/clerk", container.UserWebhookMiddleware.Protected(), container.UserWebhookHandler.Execute)
	webhooks.Post("/minio/object-created", container.MinioWebhookMiddleware.Protected(), container.MinioWebhookHandler.ObjectCreated)
}

func setupHealthChecks(app *fiber.App) {
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())
}

func setupAPIRoutes(app *fiber.App, container *di.Container) {
	public := app.Group("/api")

	protected := public.Group("", container.AuthenticateMiddleware.Protected())
	setupUserRoutes(protected, container)
	setupRealtimeRoutes(protected, container)
	setupProjectRoutes(protected, container)
	setupMediaRoutes(protected, container)
	setupDetectionRoutes(protected, container)
	setupTimelineRoutes(protected, container)
}

func setupRealtimeRoutes(api fiber.Router, container *di.Container) {
	api.Get("/realtime/connection", container.RealtimeHandler.GetConnection)
}

func setupUserRoutes(api fiber.Router, container *di.Container) {
	api.Get("/users/me", container.UserHandler.GetUser)
}

func setupProjectRoutes(api fiber.Router, container *di.Container) {
	api.Get("/projects", container.ProjectHandler.List)
	api.Post("/projects", container.ProjectHandler.Create)
	api.Get("/projects/:id", container.ProjectHandler.GetByID)
	api.Patch("/projects/:id", container.ProjectHandler.Update)
	api.Delete("/projects/:id", container.ProjectHandler.Delete)

	api.Get("/projects/:id/media", container.MediaHandler.ListByProject)
	api.Post("/projects/:id/media/uploads", container.MediaHandler.InitiateUpload)
	api.Post("/projects/:id/media/from-url", container.MediaHandler.FromURL)
}

func setupMediaRoutes(api fiber.Router, container *di.Container) {
	api.Get("/media/:id", container.MediaHandler.GetByID)
	api.Delete("/media/:id", container.MediaHandler.Delete)
	api.Post("/media/:id/uploads/parts", container.MediaHandler.PresignParts)
	api.Post("/media/:id/uploads/complete", container.MediaHandler.CompleteUpload)
	api.Get("/media/:id/configuration", container.MediaHandler.GetConfiguration)
	api.Patch("/media/:id/configuration", container.MediaHandler.UpdateConfiguration)
	api.Post("/media/:id/reanalyze", container.MediaHandler.Reanalyze)
}

func setupDetectionRoutes(api fiber.Router, container *di.Container) {
	api.Get("/media/:id/detections", container.DetectionHandler.ListByMedia)
}

func setupTimelineRoutes(api fiber.Router, container *di.Container) {
	api.Get("/projects/:id/timeline", container.TimelineHandler.Get)
	api.Post("/projects/:id/timeline/preview", container.TimelineHandler.Preview)
	api.Post("/projects/:id/timeline/validate", container.TimelineHandler.Validate)
}
