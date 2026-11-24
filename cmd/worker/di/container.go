package di

import (
	"log"

	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/application/event/dedup"
	eventmedia "go-api/internal/application/event/media"
	eventuser "go-api/internal/application/event/user"
	"go-api/internal/application/registry"
	domainmedia "go-api/internal/domain/media"
	domainuser "go-api/internal/domain/user"
	"go-api/internal/infrastructure/centrifugo"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/messaging/rabbitmq"
	"go-api/internal/infrastructure/messaging/studiojobs"
	"go-api/internal/infrastructure/notification"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/read"
	"go-api/internal/infrastructure/persistence/write"

	"gorm.io/gorm"
)

type Container struct {
	Relay            *outbox.Relay
	Consumer         *rabbitmq.Consumer
	ResultsConsumer  *studiojobs.ResultsConsumer
	ProgressConsumer *studiojobs.ProgressConsumer
	Conn             *rabbitmq.Connection
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	topology := rabbitmq.DefaultTopology(
		env.RabbitMQExchange,
		env.RabbitMQQueue,
		env.RabbitMQRoutingKey,
		env.RabbitMQRetryTTLMS,
	)

	conn, err := rabbitmq.Connect(env.RabbitMQURL, topology)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	publisher := rabbitmq.NewPublisher(conn, env.RabbitMQExchange)
	outboxRepo := outbox.NewRepository(db)
	relay := outbox.NewRelay(outboxRepo, publisher, env.OutboxPollInterval, 50)

	dedupRepo := processed.NewRepository(db)
	notifier := notification.NewLogNotifier()
	realtimePublisher := centrifugo.NewPublisher(env)
	publishUserRealtime := eventuser.NewPublishRealtimeHandler(realtimePublisher)
	publishMediaRealtime := eventmedia.NewPublishRealtimeHandler(realtimePublisher)
	reg := registry.NewHandlerRegistry()

	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"user_created",
		eventuser.NewUserCreatedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"notify_user_on_created",
		eventuser.NewNotifyUserOnCreatedHandler(notifier).Handle,
	))
	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"publish_user_created_realtime",
		publishUserRealtime.OnCreated,
	))
	reg.Register(domainuser.EventTypeUserUpdated, dedup.With(
		dedupRepo,
		"user_updated",
		eventuser.NewUserUpdatedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserUpdated, dedup.With(
		dedupRepo,
		"publish_user_updated_realtime",
		publishUserRealtime.OnUpdated,
	))
	reg.Register(domainuser.EventTypeUserDeleted, dedup.With(
		dedupRepo,
		"user_deleted",
		eventuser.NewUserDeletedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserDeleted, dedup.With(
		dedupRepo,
		"publish_user_deleted_realtime",
		publishUserRealtime.OnDeleted,
	))

	jobPublisher, err := studiojobs.NewPublisher(env.RabbitMQURL)
	if err != nil {
		log.Fatalf("failed to create job publisher: %v", err)
	}
	reg.Register(domainmedia.EventTypeMediaSourceReady, dedup.With(
		dedupRepo,
		"enqueue_probe_on_source_ready",
		eventmedia.NewEnqueueProbeHandler(jobPublisher).Handle,
	))
	reg.Register(domainmedia.EventTypeMediaSourceReady, dedup.With(
		dedupRepo,
		"publish_media_source_ready_realtime",
		publishMediaRealtime.OnSourceReady,
	))
	reg.Register(domainmedia.EventTypeMediaProbed, dedup.With(
		dedupRepo,
		"enqueue_audio_proxy_on_probed",
		eventmedia.NewEnqueueAfterProbedHandler(jobPublisher).Handle,
	))
	reg.Register(domainmedia.EventTypeMediaProbed, dedup.With(
		dedupRepo,
		"publish_media_probed_realtime",
		publishMediaRealtime.OnProbed,
	))
	reg.Register(domainmedia.EventTypeMediaAudioReady, dedup.With(
		dedupRepo,
		"enqueue_waveform_silence_on_audio_ready",
		eventmedia.NewEnqueueAfterAudioReadyHandler(jobPublisher).Handle,
	))
	reg.Register(domainmedia.EventTypeMediaAudioReady, dedup.With(
		dedupRepo,
		"publish_media_audio_ready_realtime",
		publishMediaRealtime.OnAudioReady,
	))
	reg.Register(domainmedia.EventTypeMediaProxyReady, dedup.With(
		dedupRepo,
		"publish_media_proxy_ready_realtime",
		publishMediaRealtime.OnProxyReady,
	))
	reg.Register(domainmedia.EventTypeMediaWaveformReady, dedup.With(
		dedupRepo,
		"publish_media_waveform_ready_realtime",
		publishMediaRealtime.OnWaveformReady,
	))
	reg.Register(domainmedia.EventTypeMediaReady, dedup.With(
		dedupRepo,
		"publish_media_ready_realtime",
		publishMediaRealtime.OnReady,
	))
	reg.Register(domainmedia.EventTypeMediaDetectionsUpdated, dedup.With(
		dedupRepo,
		"publish_detections_updated_realtime",
		publishMediaRealtime.OnDetectionsUpdated,
	))

	consumer := rabbitmq.NewConsumer(conn, reg, env.WorkerConcurrency, env.WorkerMaxRetries)

	mediaWriteRepo := write.NewMediaWriteRepository(db)
	detectionWriteRepo := write.NewDetectionWriteRepository(db)
	markReady := mediacmd.NewMarkSourceReadyHandler(mediaWriteRepo, outboxRepo)
	markFail := mediacmd.NewMarkFetchFailedHandler(mediaWriteRepo)
	applyProbe := mediacmd.NewApplyProbeResultHandler(mediaWriteRepo, outboxRepo)
	applyAudio := mediacmd.NewApplyAudioResultHandler(mediaWriteRepo, outboxRepo)
	applyProxy := mediacmd.NewApplyProxyResultHandler(mediaWriteRepo, outboxRepo)
	applyWave := mediacmd.NewApplyWaveformResultHandler(mediaWriteRepo, outboxRepo)
	applySilence := mediacmd.NewApplySilenceResultHandler(mediaWriteRepo, detectionWriteRepo, outboxRepo)
	resultsConsumer := studiojobs.NewResultsConsumer(
		env.RabbitMQURL, markReady, markFail, applyProbe, applyAudio, applyProxy, applyWave, applySilence,
	)
	progressConsumer := studiojobs.NewProgressConsumer(
		env.RabbitMQURL,
		read.NewMediaProjectResolver(db),
		realtimePublisher,
	)

	return &Container{
		Relay:            relay,
		Consumer:         consumer,
		ResultsConsumer:  resultsConsumer,
		ProgressConsumer: progressConsumer,
		Conn:             conn,
	}
}
