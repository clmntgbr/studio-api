package di

import (
	"log"

	authcmd "go-api/internal/application/command/auth"
	detectioncmd "go-api/internal/application/command/detection"
	identitycmd "go-api/internal/application/command/identity"
	mediacmd "go-api/internal/application/command/media"
	projectcmd "go-api/internal/application/command/project"
	timelinecmd "go-api/internal/application/command/timeline"
	usercmd "go-api/internal/application/command/user"
	querydetection "go-api/internal/application/query/detection"
	querymedia "go-api/internal/application/query/media"
	queryproject "go-api/internal/application/query/project"
	querytimeline "go-api/internal/application/query/timeline"
	querytranscript "go-api/internal/application/query/transcript"
	queryuser "go-api/internal/application/query/user"
	"go-api/internal/infrastructure/centrifugo"
	infraClerk "go-api/internal/infrastructure/clerk"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/messaging/studiojobs"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/read"
	"go-api/internal/infrastructure/persistence/write"
	s3store "go-api/internal/infrastructure/storage/s3"
	httphandler "go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/middleware"

	"gorm.io/gorm"
)

type Container struct {
	AuthenticateMiddleware *middleware.AuthenticateMiddleware
	UserWebhookMiddleware  *middleware.UserWebhookMiddleware
	UserWebhookHandler     *httphandler.UserWebhookHandler
	MinioWebhookMiddleware *middleware.MinioWebhookMiddleware
	MinioWebhookHandler    *httphandler.MinioWebhookHandler
	UserHandler            *httphandler.UserHandler
	RealtimeHandler        *httphandler.RealtimeHandler
	ProjectHandler         *httphandler.ProjectHandler
	MediaHandler           *httphandler.MediaHandler
	DetectionHandler       *httphandler.DetectionHandler
	OverrideHandler        *httphandler.OverrideHandler
	TranscriptHandler      *httphandler.TranscriptHandler
	TimelineHandler        *httphandler.TimelineHandler
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	jwksProvider, err := infraClerk.NewJWKSProvider(env)
	if err != nil {
		log.Fatalf("failed to create JWKS provider: %v", err)
	}

	userWriteRepo := write.NewUserWriteRepository(db)
	userReadRepo := read.NewUserReadRepository(db)
	outboxRepo := outbox.NewRepository(db)
	projectWriteRepo := write.NewProjectWriteRepository(db)
	projectReadRepo := read.NewProjectReadRepository(db)
	mediaWriteRepo := write.NewMediaWriteRepository(db)
	mediaReadRepo := read.NewMediaReadRepository(db)
	detectionWriteRepo := write.NewDetectionWriteRepository(db)
	detectionReadRepo := read.NewDetectionReadRepository(db)
	transcriptReadRepo := read.NewTranscriptReadRepository(db)
	timelineWriteRepo := write.NewTimelineWriteRepository(db)
	timelineReadRepo := read.NewTimelineReadRepository(db)

	storage, err := s3store.New(s3store.Config{
		Endpoint:  env.S3Endpoint,
		Region:    env.S3Region,
		AccessKey: env.S3AccessKey,
		SecretKey: env.S3SecretKey,
	})
	if err != nil {
		log.Fatalf("failed to create s3 client: %v", err)
	}

	jobPublisher, err := studiojobs.NewPublisher(env.RabbitMQURL)
	if err != nil {
		log.Fatalf("failed to create job publisher: %v", err)
	}

	createUserHandler := usercmd.NewCreateUserHandler(userWriteRepo, outboxRepo)
	updateUserHandler := usercmd.NewUpdateUserHandler(userWriteRepo, outboxRepo)
	getUserByExternalIDHandler := usercmd.NewGetUserByExternalIDHandler(userWriteRepo)
	deleteUserByExternalIDHandler := usercmd.NewDeleteUserByExternalIDHandler(userWriteRepo, outboxRepo)
	validateTokenHandler := authcmd.NewValidateTokenHandler(jwksProvider, userWriteRepo)
	fetchUserHandler := identitycmd.NewFetchUserHandler(infraClerk.NewUserGateway(env.ClerkSecretKey))
	getUserByIDHandler := queryuser.NewGetUserByIDHandler(userReadRepo)

	createProjectHandler := projectcmd.NewCreateProjectHandler(projectWriteRepo, outboxRepo)
	updateProjectHandler := projectcmd.NewUpdateProjectHandler(projectWriteRepo)
	deleteProjectHandler := projectcmd.NewDeleteProjectHandler(projectWriteRepo)
	getProjectHandler := queryproject.NewGetProjectHandler(projectReadRepo)
	listProjectsHandler := queryproject.NewListProjectsHandler(projectReadRepo)

	initiateUploadHandler := mediacmd.NewInitiateUploadHandler(
		projectWriteRepo, mediaWriteRepo, outboxRepo, storage, env.S3BucketMedia, env.S3PublicEndpoint,
	)
	presignPartsHandler := mediacmd.NewPresignPartsHandler(mediaWriteRepo, storage, env.S3BucketMedia, env.S3PublicEndpoint)
	markSourceReadyHandler := mediacmd.NewMarkSourceReadyHandler(mediaWriteRepo, outboxRepo)
	completeUploadHandler := mediacmd.NewCompleteUploadHandler(mediaWriteRepo, storage, env.S3BucketMedia, markSourceReadyHandler)
	fromURLHandler := mediacmd.NewFromURLHandler(projectWriteRepo, mediaWriteRepo, outboxRepo, jobPublisher)
	deleteMediaHandler := mediacmd.NewDeleteMediaHandler(mediaWriteRepo)
	getMediaHandler := querymedia.NewGetMediaHandler(mediaReadRepo)
	listMediaHandler := querymedia.NewListMediaHandler(mediaReadRepo)
	updateConfigHandler := mediacmd.NewUpdateConfigurationHandler(mediaWriteRepo)
	getConfigHandler := querymedia.NewGetConfigurationHandler(mediaReadRepo)
	reanalyzeHandler := mediacmd.NewReanalyzeHandler(mediaWriteRepo, jobPublisher)
	listStepRunsHandler := querymedia.NewListStepRunsHandler(mediaReadRepo)
	listDetectionsHandler := querydetection.NewListDetectionsHandler(detectionReadRepo)
	createOverrideHandler := detectioncmd.NewCreateOverrideHandler(mediaWriteRepo, detectionWriteRepo, outboxRepo)
	deleteOverrideHandler := detectioncmd.NewDeleteOverrideHandler(mediaWriteRepo, detectionWriteRepo, detectionReadRepo, outboxRepo)
	getTranscriptHandler := querytranscript.NewGetTranscriptHandler(transcriptReadRepo)
	getTranscriptSRTHandler := querytranscript.NewGetSRTHandler(transcriptReadRepo)

	previewTimeline := timelinecmd.NewPreviewHandler(mediaWriteRepo, mediaReadRepo, detectionReadRepo)
	validateTimeline := timelinecmd.NewValidateHandler(timelineWriteRepo, timelineReadRepo)
	getTimeline := querytimeline.NewGetTimelineHandler(timelineReadRepo)
	listTimelineVersions := querytimeline.NewListVersionsHandler(timelineReadRepo)

	return &Container{
		AuthenticateMiddleware: middleware.NewAuthenticateMiddleware(
			validateTokenHandler,
			fetchUserHandler,
			createUserHandler,
		),
		UserWebhookMiddleware: middleware.NewUserWebhookMiddleware(env.ClerkWebhookSecret),
		UserWebhookHandler: httphandler.NewUserWebhookHandler(
			getUserByExternalIDHandler,
			createUserHandler,
			updateUserHandler,
			deleteUserByExternalIDHandler,
		),
		MinioWebhookMiddleware: middleware.NewMinioWebhookMiddleware(env.MinioWebhookSecret),
		MinioWebhookHandler:    httphandler.NewMinioWebhookHandler(markSourceReadyHandler, env.S3BucketMedia),
		UserHandler:            httphandler.NewUserHandler(getUserByIDHandler),
		RealtimeHandler:        httphandler.NewRealtimeHandler(centrifugo.NewConnectionInfoCreator(env)),
		ProjectHandler: httphandler.NewProjectHandler(
			createProjectHandler,
			updateProjectHandler,
			deleteProjectHandler,
			getProjectHandler,
			listProjectsHandler,
		),
		MediaHandler: httphandler.NewMediaHandler(
			initiateUploadHandler,
			presignPartsHandler,
			completeUploadHandler,
			fromURLHandler,
			deleteMediaHandler,
			getMediaHandler,
			listMediaHandler,
			updateConfigHandler,
			getConfigHandler,
			reanalyzeHandler,
			listStepRunsHandler,
		),
		DetectionHandler:  httphandler.NewDetectionHandler(listDetectionsHandler),
		OverrideHandler:   httphandler.NewOverrideHandler(createOverrideHandler, deleteOverrideHandler),
		TranscriptHandler: httphandler.NewTranscriptHandler(getTranscriptHandler, getTranscriptSRTHandler),
		TimelineHandler: httphandler.NewTimelineHandler(
			previewTimeline,
			validateTimeline,
			getTimeline,
			listTimelineVersions,
		),
	}
}
