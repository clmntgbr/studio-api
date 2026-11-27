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
	setupOverrideRoutes(protected, container)
	setupSubtitleStyleRoutes(protected, container)
	setupTranscriptRoutes(protected, container)
	setupTimelineRoutes(protected, container)
	setupOutputPresetRoutes(protected, container)
	setupRenderRoutes(protected, container)
	setupClipRoutes(protected, container)
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
	api.Get("/media/:id/steps", container.MediaHandler.ListSteps)
	api.Get("/media/:id/playback", container.PlaybackHandler.Get)
	api.Get("/media/:id/stream", container.PlaybackHandler.Stream)
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

func setupOverrideRoutes(api fiber.Router, container *di.Container) {
	api.Post("/media/:id/overrides", container.OverrideHandler.Create)
	api.Delete("/overrides/:id", container.OverrideHandler.Delete)
}

func setupSubtitleStyleRoutes(api fiber.Router, container *di.Container) {
	api.Get("/subtitle-styles", container.SubtitleStyleHandler.List)
	api.Post("/subtitle-styles", container.SubtitleStyleHandler.Create)
	api.Get("/subtitle-styles/:id", container.SubtitleStyleHandler.GetByID)
	api.Patch("/subtitle-styles/:id", container.SubtitleStyleHandler.Update)
	api.Delete("/subtitle-styles/:id", container.SubtitleStyleHandler.Delete)
}

func setupTranscriptRoutes(api fiber.Router, container *di.Container) {
	api.Get("/media/:id/transcript.srt", container.TranscriptHandler.GetSRT)
	api.Get("/media/:id/transcript", container.TranscriptHandler.GetByMedia)
}

func setupTimelineRoutes(api fiber.Router, container *di.Container) {
	api.Get("/media/:id/timeline", container.TimelineHandler.Get)
	api.Get("/media/:id/timeline/versions", container.TimelineHandler.ListVersions)
	api.Get("/media/:id/timeline/subtitles.ass", container.TimelineHandler.GetSubtitlesASS)
	api.Get("/media/:id/timeline/subtitles.srt", container.TimelineHandler.GetSubtitlesSRT)
	api.Get("/media/:id/timeline/export.fcpxml", container.TimelineHandler.GetExportFCPXML)
	api.Get("/media/:id/timeline/export.edl", container.TimelineHandler.GetExportEDL)
	api.Get("/media/:id/timeline/export.otio", container.TimelineHandler.GetExportOTIO)
	api.Get("/media/:id/timeline/export.premiere.xml", container.TimelineHandler.GetExportPremiere)
	api.Post("/media/:id/timeline/preview", container.TimelineHandler.Preview)
	api.Post("/media/:id/timeline/validate", container.TimelineHandler.Validate)
}

func setupOutputPresetRoutes(api fiber.Router, container *di.Container) {
	api.Get("/output-presets", container.OutputPresetHandler.List)
}

func setupRenderRoutes(api fiber.Router, container *di.Container) {
	api.Post("/media/:id/renders", container.RenderHandler.Create)
	api.Get("/media/:id/renders", container.RenderHandler.ListByMedia)
	api.Get("/renders/:id", container.RenderHandler.GetByID)
	api.Post("/renders/:id/cancel", container.RenderHandler.Cancel)
	api.Get("/renders/:id/download", container.RenderHandler.Download)
}

func setupClipRoutes(api fiber.Router, container *di.Container) {
	api.Get("/projects/:id/clips", container.ClipHandler.ListByProject)
	api.Get("/projects/:id/clips/numbered-pack", container.ClipHandler.NumberedPack)
	api.Post("/projects/:id/clips", container.ClipHandler.Create)
	api.Post("/projects/:id/clips/analyze", container.ClipHandler.Analyze)
	api.Patch("/clips/:id", container.ClipHandler.Update)
	api.Post("/clips/:id/render", container.ClipHandler.Render)
	api.Get("/media/:id/llm-artifacts", container.ClipHandler.ListArtifactsByMedia)
}
