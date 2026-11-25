package studiojobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	detectioncmd "go-api/internal/application/command/detection"
	mediacmd "go-api/internal/application/command/media"
	rendercmd "go-api/internal/application/command/render"
	transcriptcmd "go-api/internal/application/command/transcript"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/pipeline"
	"go-api/internal/studio/envelope"
	"go-api/internal/studio/topology"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ResultsConsumer consumes studio.results (q.results) and runs shared post-source orchestration.
type ResultsConsumer struct {
	url             string
	media           domainmedia.WriteRepository
	markReady       *mediacmd.MarkSourceReadyHandler
	markFail        *mediacmd.MarkFetchFailedHandler
	applyProbe      *mediacmd.ApplyProbeResultHandler
	applyAudio      *mediacmd.ApplyAudioResultHandler
	applyProxy      *mediacmd.ApplyProxyResultHandler
	applyWave       *mediacmd.ApplyWaveformResultHandler
	applySilence    *mediacmd.ApplySilenceResultHandler
	applyTranscript *transcriptcmd.ApplyResultHandler
	applyText       *detectioncmd.ApplyTextResultHandler
	applyLLM        *detectioncmd.ApplyLLMResultHandler
	applyScenes     *detectioncmd.ApplyScenesResultHandler
	applyReframe    *mediacmd.ApplyReframeResultHandler
	applyRender     *rendercmd.ApplyResultHandler
}

func NewResultsConsumer(
	rabbitURL string,
	media domainmedia.WriteRepository,
	markReady *mediacmd.MarkSourceReadyHandler,
	markFail *mediacmd.MarkFetchFailedHandler,
	applyProbe *mediacmd.ApplyProbeResultHandler,
	applyAudio *mediacmd.ApplyAudioResultHandler,
	applyProxy *mediacmd.ApplyProxyResultHandler,
	applyWave *mediacmd.ApplyWaveformResultHandler,
	applySilence *mediacmd.ApplySilenceResultHandler,
	applyTranscript *transcriptcmd.ApplyResultHandler,
	applyText *detectioncmd.ApplyTextResultHandler,
	applyLLM *detectioncmd.ApplyLLMResultHandler,
	applyScenes *detectioncmd.ApplyScenesResultHandler,
	applyReframe *mediacmd.ApplyReframeResultHandler,
	applyRender *rendercmd.ApplyResultHandler,
) *ResultsConsumer {
	return &ResultsConsumer{
		url:             rabbitURL,
		media:           media,
		markReady:       markReady,
		markFail:        markFail,
		applyProbe:      applyProbe,
		applyAudio:      applyAudio,
		applyProxy:      applyProxy,
		applyWave:       applyWave,
		applySilence:    applySilence,
		applyTranscript: applyTranscript,
		applyText:       applyText,
		applyLLM:       applyLLM,
		applyScenes:     applyScenes,
		applyReframe:    applyReframe,
		applyRender:     applyRender,
	}
}

func (c *ResultsConsumer) Start(ctx context.Context) error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := topology.DeclareExchanges(ch); err != nil {
		return err
	}
	if err := topology.DeclareResultsQueue(ch, 30000); err != nil {
		return err
	}
	if err := ch.Qos(8, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume("q.results", "core-results", false, false, false, false, nil)
	if err != nil {
		return err
	}

	bus, err := topology.NewBus(ch, 8)
	if err != nil {
		return err
	}

	log.Println("studio results consumer started (q.results)")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("results channel closed")
			}
			if err := c.handle(ctx, bus, d); err != nil {
				log.Printf("results handle error: %v", err)
			}
		}
	}
}

func (c *ResultsConsumer) handle(ctx context.Context, bus *topology.Bus, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		log.Printf("poison results envelope: %v", err)
		return bus.Nack(d)
	}

	var handleErr error
	switch msg.Type {
	case "result.fetch.succeeded":
		handleErr = c.onFetchSucceeded(ctx, msg)
	case "result.fetch.failed":
		handleErr = c.onFetchFailed(ctx, msg)
	case "result.probe.succeeded":
		handleErr = c.onProbeSucceeded(ctx, msg)
	case "result.audio.succeeded":
		handleErr = c.onAudioSucceeded(ctx, msg)
	case "result.proxy.succeeded":
		handleErr = c.onProxySucceeded(ctx, msg)
	case "result.waveform.succeeded":
		handleErr = c.onWaveformSucceeded(ctx, msg)
	case "result.silence.succeeded":
		handleErr = c.onSilenceSucceeded(ctx, msg)
	case "result.transcribe.succeeded":
		handleErr = c.onTranscribeSucceeded(ctx, msg)
	case "result.text.succeeded":
		handleErr = c.onTextSucceeded(ctx, msg)
	case "result.llm.succeeded":
		handleErr = c.onLLMSucceeded(ctx, msg)
	case "result.scenes.succeeded":
		handleErr = c.onScenesSucceeded(ctx, msg)
	case "result.reframe.succeeded":
		handleErr = c.onReframeSucceeded(ctx, msg)
	case "result.render.succeeded":
		handleErr = c.onRenderSucceeded(ctx, msg)
	case "result.render.failed":
		handleErr = c.onRenderFailed(ctx, msg)
	case "result.probe.failed", "result.audio.failed", "result.proxy.failed", "result.waveform.failed", "result.silence.failed", "result.transcribe.failed", "result.text.failed", "result.llm.failed", "result.scenes.failed", "result.reframe.failed":
		handleErr = c.onStepFailed(ctx, msg)
	default:
		return bus.Ack(d)
	}
	if handleErr != nil {
		log.Printf("results handling failed type=%s: %v", msg.Type, handleErr)
		return bus.Nack(d)
	}
	return bus.Ack(d)
}

type fetchOutput struct {
	SourceKey string `json:"sourceKey"`
	SizeBytes int64  `json:"sizeBytes"`
	Title     string `json:"title"`
}

type probeOutput struct {
	DurationMs    int64   `json:"durationMs"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	FPSNum        int     `json:"fpsNum"`
	FPSDen        int     `json:"fpsDen"`
	VideoCodec    string  `json:"videoCodec"`
	AudioCodec    string  `json:"audioCodec"`
	AudioChannels int     `json:"audioChannels"`
}

type audioOutput struct {
	AnalysisKey      string `json:"analysisKey"`
	TranscriptionKey string `json:"transcriptionKey"`
	MasterKey        string `json:"masterKey"`
}

type proxyOutput struct {
	ProxyKey string `json:"proxyKey"`
}

type waveformOutput struct {
	PeaksKey string `json:"peaksKey"`
}

type silenceInterval struct {
	StartMs int64 `json:"startMs"`
	EndMs   int64 `json:"endMs"`
}

type silenceOutput struct {
	DetectionsKey         string            `json:"detectionsKey"`
	NoiseFloorDb          float64           `json:"noiseFloorDb"`
	CalculatedThresholdDb float64           `json:"calculatedThresholdDb"`
	Intervals             []silenceInterval `json:"intervals"`
}

func (c *ResultsConsumer) onFetchSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[fetchOutput](msg)
	if err != nil {
		return err
	}
	var sizePtr *int64
	if out.SizeBytes > 0 {
		sz := out.SizeBytes
		sizePtr = &sz
	}
	var titlePtr *string
	if out.Title != "" {
		t := out.Title
		titlePtr = &t
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.markReady.Handle(hCtx, mediacmd.MarkSourceReadyCommand{
		MediaID:   mediaID,
		SourceKey: out.SourceKey,
		SizeBytes: sizePtr,
		Title:     titlePtr,
	})
}

func (c *ResultsConsumer) onFetchFailed(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	hCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return c.markFail.Handle(hCtx, mediacmd.MarkFetchFailedCommand{MediaID: mediaID})
}

func (c *ResultsConsumer) onProbeSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[probeOutput](msg)
	if err != nil {
		return err
	}
	fps := 0.0
	if out.FPSDen > 0 {
		fps = float64(out.FPSNum) / float64(out.FPSDen)
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyProbe.Handle(hCtx, mediacmd.ApplyProbeResultCommand{
		MediaID:    mediaID,
		DurationMs: out.DurationMs,
		Width:      out.Width,
		Height:     out.Height,
		FPS:        fps,
		VideoCodec: out.VideoCodec,
		AudioCodec: out.AudioCodec,
		HasAudio:   out.AudioChannels > 0,
	})
}

func (c *ResultsConsumer) onAudioSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[audioOutput](msg)
	if err != nil {
		return err
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyAudio.Handle(hCtx, mediacmd.ApplyAudioResultCommand{
		MediaID:          mediaID,
		AnalysisKey:      out.AnalysisKey,
		TranscriptionKey: out.TranscriptionKey,
		MasterKey:        out.MasterKey,
	})
}

func (c *ResultsConsumer) onProxySucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[proxyOutput](msg)
	if err != nil {
		return err
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyProxy.Handle(hCtx, mediacmd.ApplyProxyResultCommand{
		MediaID:  mediaID,
		ProxyKey: out.ProxyKey,
	})
}

func (c *ResultsConsumer) onWaveformSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[waveformOutput](msg)
	if err != nil {
		return err
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyWave.Handle(hCtx, mediacmd.ApplyWaveformResultCommand{
		MediaID:  mediaID,
		PeaksKey: out.PeaksKey,
	})
}

func (c *ResultsConsumer) onSilenceSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[silenceOutput](msg)
	if err != nil {
		return err
	}
	intervals := make([]pipeline.SilenceInterval, 0, len(out.Intervals))
	for _, iv := range out.Intervals {
		intervals = append(intervals, pipeline.SilenceInterval{StartMs: iv.StartMs, EndMs: iv.EndMs})
	}
	hCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return c.applySilence.Handle(hCtx, mediacmd.ApplySilenceResultCommand{
		MediaID:               mediaID,
		Attempt:               msg.Attempt,
		DetectionsKey:         out.DetectionsKey,
		NoiseFloorDb:          out.NoiseFloorDb,
		CalculatedThresholdDb: out.CalculatedThresholdDb,
		Intervals:             intervals,
	})
}

type transcriptWordOut struct {
	Text          string   `json:"text"`
	SourceStartMs int64    `json:"sourceStartMs"`
	SourceEndMs   int64    `json:"sourceEndMs"`
	Confidence    *float64 `json:"confidence"`
	Speaker       *string  `json:"speaker"`
}

type transcriptOutput struct {
	WordsKey      string              `json:"wordsKey"`
	Language      string              `json:"language"`
	Text          string              `json:"text"`
	Provider      string              `json:"provider"`
	ProviderJobID string              `json:"providerJobId"`
	Words         []transcriptWordOut `json:"words"`
}

func (c *ResultsConsumer) onTranscribeSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[transcriptOutput](msg)
	if err != nil {
		return err
	}
	words := make([]transcriptcmd.WordInput, 0, len(out.Words))
	for _, w := range out.Words {
		words = append(words, transcriptcmd.WordInput{
			Text:          w.Text,
			SourceStartMs: w.SourceStartMs,
			SourceEndMs:   w.SourceEndMs,
			Confidence:    w.Confidence,
			Speaker:       w.Speaker,
		})
	}
	hCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	return c.applyTranscript.Handle(hCtx, transcriptcmd.ApplyResultCommand{
		MediaID:       mediaID,
		WordsKey:      out.WordsKey,
		Language:      out.Language,
		Text:          out.Text,
		Provider:      out.Provider,
		ProviderJobID: out.ProviderJobID,
		Words:         words,
	})
}

type textIntervalOut struct {
	Kind          string   `json:"kind"`
	SourceStartMs int64    `json:"sourceStartMs"`
	SourceEndMs   int64    `json:"sourceEndMs"`
	FirstWord     *int     `json:"firstWord"`
	LastWord      *int     `json:"lastWord"`
	Confidence    *float64 `json:"confidence"`
}

type textOutput struct {
	DetectionsKey string            `json:"detectionsKey"`
	Detector      string            `json:"detector"`
	Intervals     []textIntervalOut `json:"intervals"`
}

func (c *ResultsConsumer) onTextSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[textOutput](msg)
	if err != nil {
		return err
	}
	intervals := make([]detectioncmd.TextInterval, 0, len(out.Intervals))
	for _, iv := range out.Intervals {
		intervals = append(intervals, detectioncmd.TextInterval{
			Kind:          iv.Kind,
			SourceStartMs: iv.SourceStartMs,
			SourceEndMs:   iv.SourceEndMs,
			FirstWord:     iv.FirstWord,
			LastWord:      iv.LastWord,
			Confidence:    iv.Confidence,
		})
	}
	hCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return c.applyText.Handle(hCtx, detectioncmd.ApplyTextResultCommand{
		MediaID:       mediaID,
		Attempt:       msg.Attempt,
		DetectionsKey: out.DetectionsKey,
		Detector:      out.Detector,
		Intervals:     intervals,
	})
}

type llmSponsorOut struct {
	SourceStartMs int64   `json:"sourceStartMs"`
	SourceEndMs   int64   `json:"sourceEndMs"`
	FirstWord     *int    `json:"firstWord"`
	LastWord      *int    `json:"lastWord"`
	Confidence    float64 `json:"confidence"`
	Reason        string  `json:"reason"`
}

type llmChapterOut struct {
	Title         string `json:"title"`
	SourceStartMs int64  `json:"sourceStartMs"`
	SourceEndMs   int64  `json:"sourceEndMs"`
}

type llmClipOut struct {
	SourceStartMs int64    `json:"sourceStartMs"`
	SourceEndMs   int64    `json:"sourceEndMs"`
	Score         float64  `json:"score"`
	Hook          string   `json:"hook"`
	Reason        string   `json:"reason"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Hashtags      []string `json:"hashtags"`
}

type llmOutput struct {
	ArtifactsKey string          `json:"artifactsKey"`
	Provider     string          `json:"provider"`
	Sponsors     []llmSponsorOut `json:"sponsors"`
	Chapters     []llmChapterOut `json:"chapters"`
	Clips        []llmClipOut    `json:"clips"`
}

func (c *ResultsConsumer) onLLMSucceeded(ctx context.Context, msg *envelope.Message) error {
	if c.applyLLM == nil {
		return nil
	}
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[llmOutput](msg)
	if err != nil {
		return err
	}
	sponsors := make([]detectioncmd.LLMSponsorInterval, 0, len(out.Sponsors))
	for _, s := range out.Sponsors {
		conf := s.Confidence
		sponsors = append(sponsors, detectioncmd.LLMSponsorInterval{
			SourceStartMs: s.SourceStartMs,
			SourceEndMs:   s.SourceEndMs,
			FirstWord:     s.FirstWord,
			LastWord:      s.LastWord,
			Confidence:    &conf,
			Reason:        s.Reason,
		})
	}
	chapters := make([]detectioncmd.LLMChapter, 0, len(out.Chapters))
	for _, ch := range out.Chapters {
		chapters = append(chapters, detectioncmd.LLMChapter{
			Title:         ch.Title,
			SourceStartMs: ch.SourceStartMs,
			SourceEndMs:   ch.SourceEndMs,
		})
	}
	clips := make([]detectioncmd.LLMClipCandidate, 0, len(out.Clips))
	for _, cl := range out.Clips {
		score := cl.Score
		clips = append(clips, detectioncmd.LLMClipCandidate{
			SourceStartMs: cl.SourceStartMs,
			SourceEndMs:   cl.SourceEndMs,
			Score:         &score,
			Hook:          cl.Hook,
			Reason:        cl.Reason,
			Title:         cl.Title,
			Description:   cl.Description,
			Hashtags:      cl.Hashtags,
		})
	}
	hCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return c.applyLLM.Handle(hCtx, detectioncmd.ApplyLLMResultCommand{
		MediaID:      mediaID,
		ArtifactsKey: out.ArtifactsKey,
		Provider:     out.Provider,
		Sponsors:     sponsors,
		Chapters:     chapters,
		Clips:        clips,
	})
}

type scenesAdCandidate struct {
	SourceStartMs int64    `json:"sourceStartMs"`
	SourceEndMs   int64    `json:"sourceEndMs"`
	Confidence    *float64 `json:"confidence"`
	Reason        string   `json:"reason"`
	CutCount      int      `json:"cutCount"`
}

type scenesOutput struct {
	ScenesKey       string               `json:"scenesKey"`
	AdCandidatesKey string               `json:"adCandidatesKey"`
	Detector        string               `json:"detector"`
	SceneCount      int                  `json:"sceneCount"`
	AdCandidates    []scenesAdCandidate  `json:"adCandidates"`
}

func (c *ResultsConsumer) onScenesSucceeded(ctx context.Context, msg *envelope.Message) error {
	if c.applyScenes == nil {
		return nil
	}
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[scenesOutput](msg)
	if err != nil {
		return err
	}
	ads := make([]detectioncmd.ScenesAdCandidate, 0, len(out.AdCandidates))
	for _, a := range out.AdCandidates {
		ads = append(ads, detectioncmd.ScenesAdCandidate{
			SourceStartMs: a.SourceStartMs,
			SourceEndMs:   a.SourceEndMs,
			Confidence:    a.Confidence,
			Reason:        a.Reason,
			CutCount:      a.CutCount,
		})
	}
	hCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return c.applyScenes.Handle(hCtx, detectioncmd.ApplyScenesResultCommand{
		MediaID:         mediaID,
		ScenesKey:       out.ScenesKey,
		AdCandidatesKey: out.AdCandidatesKey,
		Detector:        out.Detector,
		SceneCount:      out.SceneCount,
		Attempt:         msg.Attempt,
		AdCandidates:    ads,
	})
}

type reframeOutput struct {
	CropPathKey string `json:"cropPathKey"`
	SampleCount int    `json:"sampleCount"`
	Detector    string `json:"detector"`
}

func (c *ResultsConsumer) onReframeSucceeded(ctx context.Context, msg *envelope.Message) error {
	if c.applyReframe == nil {
		return nil
	}
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[reframeOutput](msg)
	if err != nil {
		return err
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyReframe.Handle(hCtx, mediacmd.ApplyReframeResultCommand{
		MediaID:     mediaID,
		CropPathKey: out.CropPathKey,
		SampleCount: out.SampleCount,
		Detector:    out.Detector,
	})
}

func (c *ResultsConsumer) onRenderSucceeded(ctx context.Context, msg *envelope.Message) error {
	if c.applyRender == nil {
		return nil
	}
	renderID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return err
	}
	type renderOut struct {
		OutputKey  string `json:"outputKey"`
		DurationMs int64  `json:"durationMs"`
		SizeBytes  int64  `json:"sizeBytes"`
	}
	out, err := decodeResult[renderOut](msg)
	if err != nil {
		return err
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyRender.Handle(hCtx, rendercmd.ApplyResultCommand{
		RenderID:   renderID,
		OutputKey:  out.OutputKey,
		DurationMs: out.DurationMs,
		SizeBytes:  out.SizeBytes,
	})
}

func (c *ResultsConsumer) onRenderFailed(ctx context.Context, msg *envelope.Message) error {
	log.Printf("render failed correlationId=%s", msg.CorrelationID)
	if c.applyRender == nil {
		return nil
	}
	renderID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return nil
	}
	errMsg := "render failed"
	var rp envelope.ResultPayload
	if err := json.Unmarshal(msg.Payload, &rp); err == nil && rp.Error != nil && rp.Error.Message != "" {
		errMsg = rp.Error.Message
	}
	hCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return c.applyRender.Fail(hCtx, rendercmd.FailCommand{RenderID: renderID, Message: errMsg})
}

func (c *ResultsConsumer) onStepFailed(ctx context.Context, msg *envelope.Message) error {
	log.Printf("pipeline step failed type=%s correlationId=%s", msg.Type, msg.CorrelationID)
	if c.media == nil {
		return nil
	}
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return nil
	}
	kind, ok := failedResultStepKind(msg.Type)
	if !ok {
		return nil
	}
	errMsg := "step failed"
	var rp envelope.ResultPayload
	if err := json.Unmarshal(msg.Payload, &rp); err == nil && rp.Error != nil && rp.Error.Message != "" {
		errMsg = rp.Error.Message
	}
	hCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return mediacmd.FailStep(hCtx, c.media, mediaID, kind, time.Now().UTC(), errMsg)
}

func failedResultStepKind(msgType string) (domainmedia.StepKind, bool) {
	switch msgType {
	case "result.probe.failed":
		return domainmedia.StepProbe, true
	case "result.audio.failed":
		return domainmedia.StepAudio, true
	case "result.proxy.failed":
		return domainmedia.StepProxy, true
	case "result.waveform.failed":
		return domainmedia.StepWaveform, true
	case "result.silence.failed":
		return domainmedia.StepSilence, true
	case "result.transcribe.failed":
		return domainmedia.StepTranscribe, true
	case "result.text.failed":
		return domainmedia.StepText, true
	case "result.llm.failed":
		return domainmedia.StepLLM, true
	case "result.scenes.failed":
		return domainmedia.StepScenes, true
	case "result.reframe.failed":
		return domainmedia.StepReframe, true
	default:
		return "", false
	}
}

func decodeResult[T any](msg *envelope.Message) (T, error) {
	var zero T
	var rp envelope.ResultPayload
	if err := json.Unmarshal(msg.Payload, &rp); err != nil {
		return zero, err
	}
	if len(rp.Output) == 0 {
		return zero, nil
	}
	var out T
	if err := json.Unmarshal(rp.Output, &out); err != nil {
		return zero, err
	}
	return out, nil
}
