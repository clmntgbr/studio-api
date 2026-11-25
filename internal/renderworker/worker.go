package renderworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	s3store "go-api/internal/infrastructure/storage/s3"
	"go-api/internal/mastersplit"
	"go-api/internal/renderplan"
	"go-api/internal/studio/envelope"
	"go-api/internal/studio/topology"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	RabbitURL    string
	MediaBucket  string
	RenderBucket string
	WorkDir      string
	S3           s3store.Config
	FontsDir     string
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		RabbitURL:    os.Getenv("RABBITMQ_URL"),
		MediaBucket:  envOr("S3_BUCKET_MEDIA", "media"),
		RenderBucket: envOr("S3_BUCKET_RENDERS", "renders"),
		WorkDir:      envOr("WORK_DIR", "/tmp/work"),
		FontsDir:     envOr("FONTS_DIR", "/usr/share/fonts/dejavu"),
		S3: s3store.Config{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			Region:    envOr("S3_REGION", "us-east-1"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
		},
	}
	if cfg.RabbitURL == "" {
		return cfg, fmt.Errorf("RABBITMQ_URL is required")
	}
	if cfg.S3.Endpoint == "" || cfg.S3.AccessKey == "" || cfg.S3.SecretKey == "" {
		return cfg, fmt.Errorf("S3_ENDPOINT, S3_ACCESS_KEY and S3_SECRET_KEY are required")
	}
	return cfg, nil
}

func Run(ctx context.Context, cfg Config) error {
	store, err := s3store.New(cfg.S3)
	if err != nil {
		return err
	}
	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	defer conn.Close()

	setup, err := conn.Channel()
	if err != nil {
		return err
	}
	if err := topology.DeclareExchanges(setup); err != nil {
		_ = setup.Close()
		return err
	}
	if err := topology.DeclareJobQueue(setup, "render", 7200000); err != nil {
		_ = setup.Close()
		return err
	}
	_ = setup.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	bus, err := topology.NewBus(ch, 1)
	if err != nil {
		return err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume("q.render", "worker-render", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Printf("worker-render started")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("deliveries closed")
			}
			if err := handle(ctx, cfg, store, bus, d); err != nil {
				log.Printf("worker-render error: %v", err)
			}
		}
	}
}

func handle(ctx context.Context, cfg Config, store *s3store.Client, bus *topology.Bus, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	var plan renderplan.Plan
	if err := json.Unmarshal(msg.Payload, &plan); err != nil || plan.RenderID == "" || len(plan.Segments) == 0 {
		return finish(ctx, bus, d, msg, false, nil, "invalid_payload", "invalid render plan", false)
	}

	workDir := filepath.Join(cfg.WorkDir, "render-"+plan.RenderID)
	_ = os.RemoveAll(workDir)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "workdir", err.Error(), true)
	}
	defer os.RemoveAll(workDir)

	_ = publishProgress(ctx, bus, msg, 5, "download", "Downloading source media")
	inputs, inputIdx, err := downloadUniqueInputs(ctx, store, cfg.MediaBucket, workDir, plan.Segments)
	if err != nil {
		return finish(ctx, bus, d, msg, false, nil, "download", err.Error(), true)
	}

	assPath := ""
	fontsDir := cfg.FontsDir
	if plan.Subtitles != nil && plan.Subtitles.ASSKey != "" {
		_ = publishProgress(ctx, bus, msg, 25, "subtitles", "Downloading burn-in track")
		assPath = filepath.Join(workDir, "subs.ass")
		if err := store.DownloadFile(ctx, cfg.RenderBucket, plan.Subtitles.ASSKey, assPath); err != nil {
			if err2 := store.DownloadFile(ctx, cfg.MediaBucket, plan.Subtitles.ASSKey, assPath); err2 != nil {
				return finish(ctx, bus, d, msg, false, nil, "ass_download", err.Error(), true)
			}
		}
		if plan.Subtitles.FontsDir != "" {
			fontsDir = plan.Subtitles.FontsDir
		}
	}

	var cropDoc *cropPathDoc
	mode := strings.ToLower(plan.Crop.Mode)
	if (mode == "face_track" || mode == "speaker_track") && plan.Crop.PathKey != nil && *plan.Crop.PathKey != "" {
		_ = publishProgress(ctx, bus, msg, 30, "crop_path", "Downloading face track")
		local := filepath.Join(workDir, "crop_path.json")
		key := strings.TrimPrefix(*plan.Crop.PathKey, "/")
		if err := store.DownloadFile(ctx, cfg.MediaBucket, key, local); err != nil {
			return finish(ctx, bus, d, msg, false, nil, "crop_path", err.Error(), true)
		}
		doc, err := loadCropPath(local)
		if err != nil {
			return finish(ctx, bus, d, msg, false, nil, "crop_path", err.Error(), false)
		}
		cropDoc = doc
	}

	_ = publishProgress(ctx, bus, msg, 40, "ffmpeg", "Encoding output")
	outLocal := filepath.Join(workDir, "out.mp4")
	if err := runFFmpeg(ctx, workDir, inputs, inputIdx, plan, outLocal, assPath, fontsDir, cropDoc); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "ffmpeg", err.Error(), true)
	}

	info, err := os.Stat(outLocal)
	if err != nil {
		return finish(ctx, bus, d, msg, false, nil, "stat", err.Error(), false)
	}
	key := plan.OutputKey
	_ = publishProgress(ctx, bus, msg, 90, "upload", "Uploading render output")
	if err := store.UploadFile(ctx, cfg.RenderBucket, key, outLocal, "video/mp4"); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "upload", err.Error(), true)
	}

	var partKeys []string
	if plan.Kind == "master" && strings.TrimSpace(plan.MasterSplit) != "" {
		_ = publishProgress(ctx, bus, msg, 93, "split", "Splitting master for multi-upload")
		keys, err := splitAndUploadMaster(ctx, store, cfg.RenderBucket, workDir, outLocal, plan)
		if err != nil {
			return finish(ctx, bus, d, msg, false, nil, "master_split", err.Error(), true)
		}
		partKeys = keys
	}

	_ = publishProgress(ctx, bus, msg, 100, "done", "Render completed")
	payload := map[string]any{
		"outputKey":  key,
		"durationMs": estimateDurationMs(plan),
		"sizeBytes":  info.Size(),
	}
	if len(partKeys) > 0 {
		payload["outputParts"] = partKeys
	}
	return finish(ctx, bus, d, msg, true, payload, "", "", false)
}

// maxFilterSegments caps filter_complex size to avoid ffmpeg OOM on dense cut lists.
const maxFilterSegments = 48

// uniqueMediaKeys collapses repeated keys while preserving first-seen order.
func uniqueMediaKeys(keys []string) (unique []string, idx []int) {
	seen := make(map[string]int, len(keys))
	idx = make([]int, len(keys))
	for i, key := range keys {
		key = strings.TrimPrefix(key, "/")
		if j, ok := seen[key]; ok {
			idx[i] = j
			continue
		}
		seen[key] = len(unique)
		idx[i] = len(unique)
		unique = append(unique, key)
	}
	return unique, idx
}

// downloadUniqueInputs fetches each distinct mediaKey once and returns
// (unique local paths, per-segment index into that slice).
func downloadUniqueInputs(ctx context.Context, store *s3store.Client, bucket, workDir string, segs []renderplan.Segment) ([]string, []int, error) {
	keys := make([]string, len(segs))
	for i, seg := range segs {
		keys[i] = seg.MediaKey
	}
	unique, inputIdx := uniqueMediaKeys(keys)
	inputs := make([]string, len(unique))
	for i, key := range unique {
		local := filepath.Join(workDir, fmt.Sprintf("in-%d.mp4", i))
		if err := store.DownloadFile(ctx, bucket, key, local); err != nil {
			if err2 := store.DownloadFile(ctx, bucket, "/"+key, local); err2 != nil {
				return nil, nil, err
			}
		}
		inputs[i] = local
	}
	return inputs, inputIdx, nil
}

func runFFmpeg(ctx context.Context, workDir string, inputs []string, inputIdx []int, plan renderplan.Plan, outPath, assPath, fontsDir string, cropDoc *cropPathDoc) error {
	if len(inputs) == 0 || len(plan.Segments) == 0 {
		return fmt.Errorf("no inputs")
	}
	if len(inputIdx) != len(plan.Segments) {
		return fmt.Errorf("input index mismatch")
	}

	headlinePath, err := writeHeadlineFile(workDir, plan)
	if err != nil {
		return err
	}

	n := len(plan.Segments)
	needPost := plan.Audio.LoudnessLUFS != 0 || assPath != "" || headlinePath != ""

	if n <= maxFilterSegments {
		return runFFmpegConcat(ctx, workDir, inputs, inputIdx, plan, plan.Segments, 0, outPath, assPath, fontsDir, headlinePath, cropDoc, true)
	}

	// Dense cut lists: encode in batches then concat demuxer, then optional post pass.
	parts := make([]string, 0, (n+maxFilterSegments-1)/maxFilterSegments)
	for start := 0; start < n; start += maxFilterSegments {
		end := start + maxFilterSegments
		if end > n {
			end = n
		}
		partPath := filepath.Join(workDir, fmt.Sprintf("part-%d.mp4", start))
		subSegs := plan.Segments[start:end]
		subIdx := inputIdx[start:end]
		if err := runFFmpegConcat(ctx, workDir, inputs, subIdx, plan, subSegs, start, partPath, "", fontsDir, "", cropDoc, false); err != nil {
			return err
		}
		parts = append(parts, partPath)
	}

	concatPath := filepath.Join(workDir, "concat.mp4")
	if err := concatDemuxer(ctx, workDir, parts, concatPath); err != nil {
		return err
	}
	if !needPost {
		return os.Rename(concatPath, outPath)
	}
	return runFFmpegPost(ctx, workDir, concatPath, plan, outPath, assPath, fontsDir, headlinePath)
}

// runFFmpegConcat builds one filter_complex concat for segs.
// applyFinish enables loudnorm/ass/overlays when encoding the full plan in one shot.
func runFFmpegConcat(
	ctx context.Context,
	workDir string,
	inputs []string,
	inputIdx []int,
	plan renderplan.Plan,
	segs []renderplan.Segment,
	segOffset int,
	outPath, assPath, fontsDir, headlinePath string,
	cropDoc *cropPathDoc,
	applyFinish bool,
) error {
	args := []string{"-y", "-hide_banner", "-loglevel", "error"}
	for _, in := range inputs {
		args = append(args, "-i", in)
	}

	fade := plan.Audio.FadeMs
	if fade <= 0 {
		fade = 15
	}
	fadeSec := float64(fade) / 1000.0
	var filter strings.Builder
	n := len(segs)
	mode := strings.ToLower(plan.Crop.Mode)
	for i, seg := range segs {
		start := float64(seg.SrcStartMs) / 1000.0
		dur := float64(seg.SrcEndMs-seg.SrcStartMs) / 1000.0
		if dur <= 0 {
			dur = 0.04
		}
		inI := inputIdx[i]
		geom := videoGeometry(plan.Crop.Mode, plan.Output.Width, plan.Output.Height)
		if cropDoc != nil && (mode == "face_track" || mode == "speaker_track") {
			srcW, srcH, err := probeVideoSize(inputs[inI])
			if err != nil {
				return err
			}
			cmdPath, err := writeFaceTrackSendcmd(workDir, segOffset+i, cropDoc, seg.SrcStartMs, seg.SrcEndMs, srcW, srcH, plan.Output.Width, plan.Output.Height)
			if err != nil {
				return err
			}
			geom = faceTrackGeometry(cmdPath, plan.Output.Width, plan.Output.Height)
		}
		fmt.Fprintf(&filter,
			"[%d:v]trim=start=%f:duration=%f,setpts=PTS-STARTPTS,%s[v%d];",
			inI, start, dur, geom, i,
		)
		fmt.Fprintf(&filter,
			"[%d:a]atrim=start=%f:duration=%f,asetpts=PTS-STARTPTS,afade=t=in:st=0:d=%f,afade=t=out:st=%f:d=%f[a%d];",
			inI, start, dur, fadeSec, maxFloat(0, dur-fadeSec), fadeSec, i,
		)
	}
	for i := 0; i < n; i++ {
		fmt.Fprintf(&filter, "[v%d][a%d]", i, i)
	}
	fmt.Fprintf(&filter, "concat=n=%d:v=1:a=1[vout][aout]", n)

	vout := "[vout]"
	aout := "[aout]"
	if applyFinish {
		if plan.Audio.LoudnessLUFS != 0 {
			fmt.Fprintf(&filter, ";[aout]loudnorm=I=%0.1f:TP=-1.5:LRA=11[aout2]", plan.Audio.LoudnessLUFS)
			aout = "[aout2]"
		}
		if assPath != "" {
			escaped := escapeFilterPath(assPath)
			fonts := escapeFilterPath(fontsDir)
			fmt.Fprintf(&filter, ";%sass=%s:fontsdir=%s[vburn]", vout, escaped, fonts)
			vout = "[vburn]"
		}
		if overlay := headlineDrawtext(plan, fontsDir, headlinePath); overlay != "" {
			// Chain setsar after drawtext so enable=… cannot swallow the [vhead] pad.
			fmt.Fprintf(&filter, ";%s%s,setsar=1[vhead]", vout, overlay)
			vout = "[vhead]"
		}
	}

	scriptPath := filepath.Join(workDir, fmt.Sprintf("filter-%d.txt", segOffset))
	if err := os.WriteFile(scriptPath, []byte(filter.String()), 0o644); err != nil {
		return err
	}
	args = append(args, "-filter_complex_script", scriptPath)
	args = append(args, "-map", vout, "-map", aout)
	args = append(args, encodeArgs(plan)...)
	args = append(args, outPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, string(out))
	}
	return nil
}

func concatDemuxer(ctx context.Context, workDir string, parts []string, outPath string) error {
	listPath := filepath.Join(workDir, "concat.txt")
	var b strings.Builder
	for _, p := range parts {
		// Basename: ffmpeg runs with cmd.Dir=workDir (works for relative WORK_DIR too).
		fmt.Fprintf(&b, "file '%s'\n", filepath.Base(p))
	}
	if err := os.WriteFile(listPath, []byte(b.String()), 0o644); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "concat", "-safe", "0", "-i", listPath,
		"-c", "copy", "-movflags", "+faststart",
		outPath,
	)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg concat: %w: %s", err, string(out))
	}
	return nil
}

func runFFmpegPost(ctx context.Context, workDir, inPath string, plan renderplan.Plan, outPath, assPath, fontsDir, headlinePath string) error {
	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-i", inPath}
	var vf, af []string
	if assPath != "" {
		vf = append(vf, fmt.Sprintf("ass=%s:fontsdir=%s", escapeFilterPath(assPath), escapeFilterPath(fontsDir)))
	}
	if overlay := headlineDrawtext(plan, fontsDir, headlinePath); overlay != "" {
		vf = append(vf, strings.TrimPrefix(overlay, ","))
	}
	if plan.Audio.LoudnessLUFS != 0 {
		af = append(af, fmt.Sprintf("loudnorm=I=%0.1f:TP=-1.5:LRA=11", plan.Audio.LoudnessLUFS))
	}
	if len(vf) > 0 {
		args = append(args, "-vf", strings.Join(vf, ","))
	}
	if len(af) > 0 {
		args = append(args, "-af", strings.Join(af, ","))
	}
	args = append(args, encodeArgs(plan)...)
	args = append(args, outPath)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg post: %w: %s", err, string(out))
	}
	return nil
}

func splitAndUploadMaster(
	ctx context.Context,
	store *s3store.Client,
	bucket, workDir, videoPath string,
	plan renderplan.Plan,
) ([]string, error) {
	spec, err := mastersplit.Parse(plan.MasterSplit)
	if err != nil {
		return nil, err
	}
	if spec.IsZero() {
		return nil, nil
	}
	totalMs := probeDurationMs(ctx, workDir, videoPath)
	if totalMs <= 0 {
		totalMs = estimateDurationMs(plan)
	}
	windows := spec.PartWindows(totalMs)
	if len(windows) == 0 {
		return nil, nil
	}
	base := strings.TrimSuffix(plan.OutputKey, filepath.Ext(plan.OutputKey))
	if base == "" {
		base = plan.OutputKey
	}
	keys := make([]string, 0, len(windows))
	for i, w := range windows {
		startSec := float64(w[0]) / 1000.0
		durSec := float64(w[1]-w[0]) / 1000.0
		local := filepath.Join(workDir, fmt.Sprintf("part-%02d.mp4", i+1))
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-y", "-hide_banner", "-loglevel", "error",
			"-ss", fmt.Sprintf("%g", startSec),
			"-i", videoPath,
			"-t", fmt.Sprintf("%g", durSec),
			"-c", "copy", "-avoid_negative_ts", "make_zero",
			"-movflags", "+faststart",
			local,
		)
		cmd.Dir = workDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("ffmpeg split part %d: %w: %s", i+1, err, string(out))
		}
		key := fmt.Sprintf("%s-part-%02d.mp4", base, i+1)
		if err := store.UploadFile(ctx, bucket, key, local, "video/mp4"); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func probeDurationMs(ctx context.Context, workDir, path string) int64 {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", path,
	)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || sec <= 0 {
		return 0
	}
	return int64(sec * 1000)
}

func encodeArgs(plan renderplan.Plan) []string {
	fps := plan.Output.FPS
	if fps <= 0 {
		fps = 30
	}
	crf := plan.Output.CRF
	if crf <= 0 {
		crf = 20
	}
	preset := plan.Output.Preset
	if preset == "" {
		preset = "veryfast"
	}
	return []string{
		"-c:v", "libx264", "-preset", preset, "-crf", strconv.Itoa(crf),
		"-r", fmt.Sprintf("%g", fps),
		"-c:a", "aac", "-b:a", "192k",
		"-movflags", "+faststart",
	}
}

func escapeFilterPath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.ReplaceAll(p, `\`, `/`)
	p = strings.ReplaceAll(p, `'`, `\'`)
	p = strings.ReplaceAll(p, `:`, `\:`)
	return "'" + p + "'"
}

// videoGeometry scales each segment to the output frame.
// center/left/right fill+crop; none letterboxes with pad.
func videoGeometry(cropMode string, w, h int) string {
	if w <= 0 || h <= 0 {
		w, h = 1920, 1080
	}
	switch strings.ToLower(cropMode) {
	case "center", "left", "right", "face_track", "speaker_track":
		// face_track without a path falls back to center crop.
		xExpr := "(iw-ow)/2"
		switch strings.ToLower(cropMode) {
		case "left":
			xExpr = "0"
		case "right":
			xExpr = "iw-ow"
		}
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d:%s:(ih-oh)/2,setsar=1",
			w, h, w, h, xExpr,
		)
	default:
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1",
			w, h, w, h,
		)
	}
}

func writeHeadlineFile(workDir string, plan renderplan.Plan) (string, error) {
	text := ""
	for _, o := range plan.Overlays {
		if o.Kind == "headline" && strings.TrimSpace(o.Text) != "" {
			text = o.Text
			break
		}
	}
	if text == "" {
		return "", nil
	}
	path := filepath.Join(workDir, "headline.txt")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func headlineDrawtext(plan renderplan.Plan, fontsDir, textFile string) string {
	if strings.TrimSpace(textFile) == "" {
		return ""
	}
	var startMs, endMs int64
	for _, o := range plan.Overlays {
		if o.Kind == "headline" && strings.TrimSpace(o.Text) != "" {
			startMs = o.StartMs
			endMs = o.EndMs
			break
		}
	}
	if endMs <= startMs {
		endMs = startMs + 3000
	}
	start := float64(startMs) / 1000.0
	end := float64(endMs) / 1000.0
	fs := plan.Output.Height / 28
	if fs < 36 {
		fs = 36
	}
	if fs > 96 {
		fs = 96
	}
	fontFile := resolveDrawtextFont(fontsDir)
	if fontFile == "" {
		// Without fontfile, ffmpeg falls back to family "Sans" and fails in Alpine.
		return ""
	}
	// textfile avoids apostrophe/comma breakage in FR hooks (text='J'ai…' → filter "0").
	// enable commas escaped; caller chains ,setsar=1[vhead].
	return fmt.Sprintf(
		"drawtext=textfile=%s:fontsize=%d:fontcolor=white:borderw=3:bordercolor=black@0.7:fontfile=%s:x=(w-text_w)/2:y=h*0.08:enable=between(t\\,%g\\,%g)",
		escapeDrawtextFontPath(textFile), fs, escapeDrawtextFontPath(fontFile), start, end,
	)
}

// escapeDrawtextFontPath escapes a TTF path for drawtext fontfile= (no wrapping quotes).
func escapeDrawtextFontPath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.ReplaceAll(p, `\`, `\\`)
	p = strings.ReplaceAll(p, `:`, `\:`)
	p = strings.ReplaceAll(p, `'`, `\'`)
	p = strings.ReplaceAll(p, `[`, `\[`)
	p = strings.ReplaceAll(p, `]`, `\]`)
	p = strings.ReplaceAll(p, `,`, `\,`)
	return p
}

func resolveDrawtextFont(fontsDir string) string {
	// Alpine 3.22+ ships font-dejavu under /usr/share/fonts/dejavu;
	// older images and Debian use ttf-dejavu / truetype/dejavu.
	candidates := []string{
		filepath.Join(fontsDir, "DejaVuSans.ttf"),
		filepath.Join(fontsDir, "dejavu", "DejaVuSans.ttf"),
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/ttf-dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func escapeDrawtext(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, `:`, `\:`)
	s = strings.ReplaceAll(s, "%", "%%")
	return s
}

func estimateDurationMs(plan renderplan.Plan) int64 {
	var total int64
	for _, s := range plan.Segments {
		if s.SrcEndMs > s.SrcStartMs {
			total += s.SrcEndMs - s.SrcStartMs
		}
	}
	return total
}

func finish(ctx context.Context, bus *topology.Bus, d amqp.Delivery, msg *envelope.Message, ok bool, output any, code, errMsg string, retryable bool) error {
	corr := msg.CorrelationID
	m, body, err := envelope.NewResult("render", corr, msg.MessageID, msg.Attempt, ok, output, code, errMsg, retryable)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	if err := bus.PublishResult(ctx, envelope.ResultRoutingKey("render", ok), body, m.MessageID, corr); err != nil {
		_ = bus.Nack(d)
		return err
	}
	return bus.Ack(d)
}

func publishProgress(ctx context.Context, bus *topology.Bus, msg *envelope.Message, pct float64, stage, message string) error {
	m, body, err := envelope.NewProgress("render", msg.CorrelationID, msg.MessageID, msg.Attempt, pct, stage, message)
	if err != nil {
		return err
	}
	return bus.PublishProgress(ctx, body, m.MessageID, msg.CorrelationID)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
