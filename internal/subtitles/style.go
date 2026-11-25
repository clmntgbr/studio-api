package subtitles

// Style drives ASS Look & Line layout (mirrors subtitle_style + settings).
type Style struct {
	FontFamily     string
	FontSize       int
	BaseColor      string // CSS #RRGGBB or #AARRGGBB
	HighlightColor string
	OutlineColor   string
	OutlineSize    int
	WordsPerLine   int
	Position       string // bottom | middle | top
	Uppercase      bool
	MaxChars       int
	PauseBreakMs   int64  // preferred line break on pause (default 350)
	LastWordPadMs  int64  // pad after last word of a line (default 120)
	Animation      string // none | pop
	MarginV        int
	Shadow         int
	PlayResX       int // ASS PlayResX (0 → default)
	PlayResY       int // ASS PlayResY (0 → default)
}

// DefaultStyle returns a karaoke-friendly white/yellow bottom style.
// Font family must exist under FONTS_DIR (DejaVu on Alpine worker images).
func DefaultStyle() Style {
	return Style{
		FontFamily:     "DejaVu Sans",
		FontSize:       64,
		BaseColor:      "#FFFFFF",
		HighlightColor: "#FFE600",
		OutlineColor:   "#000000",
		OutlineSize:    3,
		WordsPerLine:   3,
		Position:       "bottom",
		Uppercase:      true,
		MaxChars:       28,
		PauseBreakMs:   350,
		LastWordPadMs:  120,
		Animation:      "pop",
		MarginV:        80,
		Shadow:         1,
	}
}

func (s Style) normalized() Style {
	out := s
	if out.FontFamily == "" {
		out.FontFamily = "DejaVu Sans"
	}
	if out.FontSize <= 0 {
		out.FontSize = 64
	}
	if out.BaseColor == "" {
		out.BaseColor = "#FFFFFF"
	}
	if out.HighlightColor == "" {
		out.HighlightColor = "#FFE600"
	}
	if out.OutlineColor == "" {
		out.OutlineColor = "#000000"
	}
	if out.OutlineSize < 0 {
		out.OutlineSize = 0
	}
	if out.WordsPerLine < 2 {
		out.WordsPerLine = 2
	}
	if out.WordsPerLine > 4 {
		out.WordsPerLine = 4
	}
	switch out.Position {
	case "top", "middle", "bottom":
	default:
		out.Position = "bottom"
	}
	if out.MaxChars <= 0 {
		out.MaxChars = 28
	}
	if out.PauseBreakMs <= 0 {
		out.PauseBreakMs = 350
	}
	if out.LastWordPadMs <= 0 {
		out.LastWordPadMs = 120
	}
	if out.PlayResX <= 0 {
		out.PlayResX = 1920
	}
	if out.PlayResY <= 0 {
		out.PlayResY = 1080
	}
	if out.Animation == "" {
		out.Animation = "none"
	}
	if out.MarginV <= 0 {
		out.MarginV = 80
	}
	return out
}
