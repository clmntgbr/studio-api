package pipeline

// Probe is ffprobe output used by media jobs.
type Probe struct {
	DurationMs    int64  `json:"durationMs"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	FPSNum        int    `json:"fpsNum"`
	FPSDen        int    `json:"fpsDen"`
	VFR           bool   `json:"vfr"`
	VideoCodec    string `json:"videoCodec"`
	AudioCodec    string `json:"audioCodec"`
	AudioChannels int    `json:"audioChannels"`
	Rotation      int    `json:"rotation"`
}

type Origin struct {
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Uploader  string `json:"uploader,omitempty"`
	Extractor string `json:"extractor,omitempty"`
}

type ProxySpec struct {
	Height    int `json:"height"`
	GOPFrames int `json:"gopFrames"`
	CRF       int `json:"crf"`
}

type AudioAnalysisSpec struct {
	SampleRate int `json:"sampleRate"`
	Channels   int `json:"channels"`
}

type AudioTranscriptionSpec struct {
	BitrateKbps int `json:"bitrateKbps"`
}

type AudioMasterSpec struct {
	SampleRate int `json:"sampleRate"`
	Channels   int `json:"channels"`
}
