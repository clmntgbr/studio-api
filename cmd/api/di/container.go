package di

import (
	"log"

	authcmd "go-api/internal/application/command/auth"
	clipcmd "go-api/internal/application/command/clip"
	detectioncmd "go-api/internal/application/command/detection"
	identitycmd "go-api/internal/application/command/identity"
	mediacmd "go-api/internal/application/command/media"
	rendercmd "go-api/internal/application/command/render"
	subtitlestylecmd "go-api/internal/application/command/subtitlestyle"
	timelinecmd "go-api/internal/application/command/timeline"
	usercmd "go-api/internal/application/command/user"
	queryclip "go-api/internal/application/query/clip"
	querydetection "go-api/internal/application/query/detection"
	queryllm "go-api/internal/application/query/llmartifact"
	querymedia "go-api/internal/application/query/media"
	queryoutputpreset "go-api/internal/application/query/outputpreset"
	queryrender "go-api/internal/application/query/render"
	querysubtitlestyle "go-api/internal/application/query/subtitlestyle"
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
	MediaHandler           *httphandler.MediaHandler
	PlaybackHandler        *httphandler.PlaybackHandler
	DetectionHandler       *httphandler.DetectionHandler
	OverrideHandler        *httphandler.OverrideHandler
	SubtitleStyleHandler   *httphandler.SubtitleStyleHandler
	TranscriptHandler      *httphandler.TranscriptHandler
	TimelineHandler        *httphandler.TimelineHandler
	OutputPresetHandler    *httphandler.OutputPresetHandler
	RenderHandler          *httphandler.RenderHandler
	ClipHandler            *httphandler.ClipHandler
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	jwksProvider, err := infraClerk.NewJWKSProvider(env)
	if err != nil {
		log.Fatalf("failed to create JWKS provider: %v", err)
	}

	userWriteRepo := write.NewUserWriteRepository(db)
	userReadRepo := read.NewUserReadRepository(db)
	outboxRepo := outbox.NewRepository(db)
	mediaWriteRepo := write.NewMediaWriteRepository(db)
	mediaReadRepo := read.NewMediaReadRepository(db)
	detectionWriteRepo := write.NewDetectionWriteRepository(db)
	detectionReadRepo := read.NewDetectionReadRepository(db)
	transcriptReadRepo := read.NewTranscriptReadRepository(db)
	timelineWriteRepo := write.NewTimelineWriteRepository(db)
	timelineReadRepo := read.NewTimelineReadRepository(db)
	subtitleStyleWriteRepo := write.NewSubtitleStyleWriteRepository(db)
	subtitleStyleReadRepo := read.NewSubtitleStyleReadRepository(db)

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

	initiateUploadHandler := mediacmd.NewInitiateUploadHandler(
		mediaWriteRepo, outboxRepo, storage, env.S3BucketMedia, env.S3PublicEndpoint,
	)
	presignPartsHandler := mediacmd.NewPresignPartsHandler(mediaWriteRepo, storage, env.S3BucketMedia, env.S3PublicEndpoint)
	markSourceReadyHandler := mediacmd.NewMarkSourceReadyHandler(mediaWriteRepo, outboxRepo)
	completeUploadHandler := mediacmd.NewCompleteUploadHandler(mediaWriteRepo, storage, env.S3BucketMedia, markSourceReadyHandler)
	fromURLHandler := mediacmd.NewFromURLHandler(mediaWriteRepo, outboxRepo, jobPublisher)
	deleteMediaHandler := mediacmd.NewDeleteMediaHandler(mediaWriteRepo)
	getMediaHandler := querymedia.NewGetMediaHandler(mediaReadRepo)
	listMediaHandler := querymedia.NewListMediaHandler(mediaReadRepo)
	updateConfigHandler := mediacmd.NewUpdateConfigurationHandler(mediaWriteRepo)
	getConfigHandler := querymedia.NewGetConfigurationHandler(mediaReadRepo)
	reanalyzeHandler := mediacmd.NewReanalyzeHandler(mediaWriteRepo, detectionWriteRepo, outboxRepo, jobPublisher)
	listStepRunsHandler := querymedia.NewListStepRunsHandler(mediaReadRepo)
	getPlaybackHandler := querymedia.NewGetPlaybackHandler(mediaWriteRepo, storage, env.S3BucketMedia, env.S3PublicEndpoint)
	getStreamHandler := querymedia.NewGetStreamHandler(mediaWriteRepo, storage, env.S3BucketMedia)
	listDetectionsHandler := querydetection.NewListDetectionsHandler(detectionReadRepo)
	createOverrideHandler := detectioncmd.NewCreateOverrideHandler(mediaWriteRepo, detectionWriteRepo, outboxRepo)
	deleteOverrideHandler := detectioncmd.NewDeleteOverrideHandler(mediaWriteRepo, detectionWriteRepo, detectionReadRepo, outboxRepo)
	getTranscriptHandler := querytranscript.NewGetTranscriptHandler(transcriptReadRepo)
	getTranscriptSRTHandler := querytranscript.NewGetSRTHandler(transcriptReadRepo)

	previewTimeline := timelinecmd.NewPreviewHandler(mediaWriteRepo, mediaReadRepo, detectionReadRepo)
	validateTimeline := timelinecmd.NewValidateHandler(timelineWriteRepo, timelineReadRepo)
	getTimeline := querytimeline.NewGetTimelineHandler(timelineReadRepo)
	listTimelineVersions := querytimeline.NewListVersionsHandler(timelineReadRepo)
	getSubtitles := querytimeline.NewGetSubtitlesHandler(timelineReadRepo, subtitleStyleReadRepo, transcriptReadRepo)
	getExport := querytimeline.NewGetExportHandler(timelineReadRepo, mediaWriteRepo)

	createSubtitleStyle := subtitlestylecmd.NewCreateHandler(subtitleStyleWriteRepo)
	updateSubtitleStyle := subtitlestylecmd.NewUpdateHandler(subtitleStyleWriteRepo)
	deleteSubtitleStyle := subtitlestylecmd.NewDeleteHandler(subtitleStyleWriteRepo)
	getSubtitleStyle := querysubtitlestyle.NewGetHandler(subtitleStyleReadRepo)
	listSubtitleStyles := querysubtitlestyle.NewListHandler(subtitleStyleReadRepo)

	outputPresetReadRepo := read.NewOutputPresetReadRepository(db)
	renderWriteRepo := write.NewRenderWriteRepository(db)
	renderReadRepo := read.NewRenderReadRepository(db)
	listOutputPresets := queryoutputpreset.NewListHandler(outputPresetReadRepo)
	clipWriteRepo := write.NewClipWriteRepository(db)
	clipReadRepo := read.NewClipReadRepository(db)
	createRender := rendercmd.NewCreateHandler(
		renderWriteRepo,
		timelineReadRepo,
		outputPresetReadRepo,
		mediaWriteRepo,
		clipReadRepo,
		jobPublisher,
		storage,
		getSubtitles,
		env.S3BucketRenders,
	)
	cancelRender := rendercmd.NewCancelHandler(renderWriteRepo, mediaWriteRepo)
	getRender := queryrender.NewGetHandler(renderReadRepo)
	listRenders := queryrender.NewListHandler(renderReadRepo)
	downloadRender := queryrender.NewDownloadHandler(renderReadRepo, storage, env.S3BucketRenders)

	llmArtifactReadRepo := read.NewLLMArtifactReadRepository(db)
	listClips := queryclip.NewListHandler(clipReadRepo)
	getClip := queryclip.NewGetHandler(clipReadRepo)
	createClip := clipcmd.NewCreateHandler(clipWriteRepo, mediaWriteRepo)
	updateClip := clipcmd.NewUpdateHandler(clipWriteRepo, clipReadRepo, mediaWriteRepo)
	analyzeClips := clipcmd.NewAnalyzeHandler(mediaWriteRepo, jobPublisher)
	listLLMArtifacts := queryllm.NewListHandler(llmArtifactReadRepo)

	userWebhookMiddleware, err := middleware.NewUserWebhookMiddleware(env.ClerkWebhookSecret)
	if err != nil {
		log.Fatalf("invalid CLERK_WEBHOOK_SECRET: %v", err)
	}

	return &Container{
		AuthenticateMiddleware: middleware.NewAuthenticateMiddleware(
			validateTokenHandler,
			fetchUserHandler,
			createUserHandler,
		),
		UserWebhookMiddleware: userWebhookMiddleware,
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
		PlaybackHandler:  httphandler.NewPlaybackHandler(getPlaybackHandler, getStreamHandler),
		DetectionHandler: httphandler.NewDetectionHandler(listDetectionsHandler),
		OverrideHandler:  httphandler.NewOverrideHandler(createOverrideHandler, deleteOverrideHandler),
		SubtitleStyleHandler: httphandler.NewSubtitleStyleHandler(
			createSubtitleStyle,
			updateSubtitleStyle,
			deleteSubtitleStyle,
			getSubtitleStyle,
			listSubtitleStyles,
		),
		TranscriptHandler: httphandler.NewTranscriptHandler(getTranscriptHandler, getTranscriptSRTHandler),
		TimelineHandler: httphandler.NewTimelineHandler(
			previewTimeline,
			validateTimeline,
			getTimeline,
			listTimelineVersions,
			getSubtitles,
			getExport,
		),
		OutputPresetHandler: httphandler.NewOutputPresetHandler(listOutputPresets),
		RenderHandler: httphandler.NewRenderHandler(
			createRender,
			cancelRender,
			getRender,
			listRenders,
			downloadRender,
		),
		ClipHandler: httphandler.NewClipHandler(
			listClips,
			createClip,
			updateClip,
			analyzeClips,
			listLLMArtifacts,
			getClip,
			createRender,
		),
	}
}