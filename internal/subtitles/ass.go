package subtitles

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var emojiRe = regexp.MustCompile(`[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}]`)

// GenerateASS builds a karaoke ASS document in OutputTime from remapped words + style.
func GenerateASS(words []TimedWord, style Style) string {
	style = style.normalized()
	cleaned := prepareWords(words, style)
	if len(cleaned) == 0 {
		return header(style)
	}
	lines := groupLines(cleaned, style)

	var b strings.Builder
	b.WriteString(header(style))
	b.WriteString("\n[Events]\n")
	b.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	for li, line := range lines {
		var nextLineStart int64 = -1
		if li+1 < len(lines) {
			nextLineStart = lines[li+1].words[0].OutputStartMs
		}
		for i := range line.words {
			start := line.words[i].OutputStartMs
			end := start
			if i+1 < len(line.words) {
				end = line.words[i+1].OutputStartMs
			} else {
				end = line.words[i].OutputEndMs + style.LastWordPadMs
				if nextLineStart >= 0 && end > nextLineStart {
					end = nextLineStart
				}
			}
			if end <= start {
				end = start + 40
			}
			text := renderLineText(line.words, i, style)
			fmt.Fprintf(&b, "Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n",
				formatASSTime(start), formatASSTime(end), text)
		}
	}
	return b.String()
}

// GenerateSRT builds a plain SRT from remapped words (no karaoke styling).
func GenerateSRT(words []TimedWord, style Style) string {
	style = style.normalized()
	cleaned := prepareWords(words, style)
	if len(cleaned) == 0 {
		return ""
	}
	lines := groupLines(cleaned, style)
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		start := line.words[0].OutputStartMs
		end := line.words[len(line.words)-1].OutputEndMs + style.LastWordPadMs
		if i+1 < len(lines) {
			nextStart := lines[i+1].words[0].OutputStartMs
			if end > nextStart {
				end = nextStart
			}
		}
		parts := make([]string, 0, len(line.words))
		for _, w := range line.words {
			parts = append(parts, w.Text)
		}
		fmt.Fprintf(&b, "%d\n", i+1)
		fmt.Fprintf(&b, "%s --> %s\n", formatSRTTime(start), formatSRTTime(end))
		b.WriteString(strings.Join(parts, " "))
		b.WriteByte('\n')
	}
	return b.String()
}

type lineGroup struct {
	words []TimedWord
}

func prepareWords(words []TimedWord, style Style) []TimedWord {
	out := make([]TimedWord, 0, len(words))
	for _, w := range words {
		text := sanitizeText(w.Text)
		if text == "" || w.OutputEndMs <= w.OutputStartMs {
			continue
		}
		if style.Uppercase {
			text = strings.ToUpper(text)
		}
		out = append(out, TimedWord{Text: text, OutputStartMs: w.OutputStartMs, OutputEndMs: w.OutputEndMs})
	}
	return out
}

func groupLines(words []TimedWord, style Style) []lineGroup {
	out := make([]lineGroup, 0)
	var cur []TimedWord
	flush := func() {
		if len(cur) == 0 {
			return
		}
		cp := append([]TimedWord(nil), cur...)
		out = append(out, lineGroup{words: cp})
		cur = nil
	}
	for _, w := range words {
		if len(cur) == 0 {
			cur = []TimedWord{w}
			continue
		}
		gap := w.OutputStartMs - cur[len(cur)-1].OutputEndMs
		nextLen := lineCharLen(cur) + 1 + utf8.RuneCountInString(w.Text)
		preferBreak := gap >= style.PauseBreakMs || endsWithPunct(cur[len(cur)-1].Text)
		if len(cur) >= style.WordsPerLine || nextLen > style.MaxChars || (preferBreak && len(cur) >= 2) {
			flush()
			cur = []TimedWord{w}
			continue
		}
		cur = append(cur, w)
	}
	flush()
	return out
}

func lineCharLen(words []TimedWord) int {
	n := 0
	for i, w := range words {
		if i > 0 {
			n++
		}
		n += utf8.RuneCountInString(w.Text)
	}
	return n
}

func endsWithPunct(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)
	last := r[len(r)-1]
	return unicode.IsPunct(last)
}

func renderLineText(words []TimedWord, active int, style Style) string {
	base := toASSColor(style.BaseColor)
	hi := toASSColor(style.HighlightColor)
	parts := make([]string, 0, len(words)*2)
	for i, w := range words {
		if i > 0 {
			parts = append(parts, " ")
		}
		if i == active {
			chunk := fmt.Sprintf(`{\c%s}`, hi)
			if style.Animation == "pop" {
				chunk += `{\t(0,80,\fscx110\fscy110)\t(80,160,\fscx100\fscy100)}`
			}
			chunk += escapeASS(w.Text) + `{\fscx100\fscy100\c` + base + `}`
			parts = append(parts, chunk)
			continue
		}
		parts = append(parts, escapeASS(w.Text))
	}
	return strings.Join(parts, "")
}

func header(style Style) string {
	align := 2 // bottom center
	switch style.Position {
	case "top":
		align = 8
	case "middle":
		align = 5
	}
	font := sanitizeFontFamily(style.FontFamily)
	return strings.Join([]string{
		"[Script Info]",
		"ScriptType: v4.00+",
		fmt.Sprintf("PlayResX: %d", style.PlayResX),
		fmt.Sprintf("PlayResY: %d", style.PlayResY),
		"WrapStyle: 0",
		"ScaledBorderAndShadow: yes",
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		fmt.Sprintf(
			"Style: Default,%s,%d,%s,&H000000FF,%s,&H80000000,0,0,0,0,100,100,0,0,1,%d,%d,%d,40,40,%d,1",
			font,
			style.FontSize,
			toASSColor(style.BaseColor),
			toASSColor(style.OutlineColor),
			style.OutlineSize,
			style.Shadow,
			align,
			style.MarginV,
		),
		"",
	}, "\n")
}

func sanitizeFontFamily(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Montserrat"
	}
	if strings.ContainsAny(name, ",\r\n{}") {
		return "Montserrat"
	}
	return name
}

func sanitizeText(s string) string {
	s = strings.TrimSpace(s)
	s = emojiRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func escapeASS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "{", "(")
	s = strings.ReplaceAll(s, "}", ")")
	s = strings.ReplaceAll(s, "\n", `\N`)
	return s
}

// toASSColor converts #RRGGBB / #AARRGGBB into ASS &HAABBGGRR.
func toASSColor(hex string) string {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	var a, r, g, b uint64
	a = 0x00 // opaque in ASS (00 = opaque)
	switch len(hex) {
	case 6:
		v, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return "&H00FFFFFF"
		}
		r = (v >> 16) & 0xFF
		g = (v >> 8) & 0xFF
		b = v & 0xFF
	case 8:
		v, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return "&H00FFFFFF"
		}
		a = (v >> 24) & 0xFF
		r = (v >> 16) & 0xFF
		g = (v >> 8) & 0xFF
		b = v & 0xFF
		// CSS alpha 0=transparent, ASS 00=opaque — invert.
		a = 0xFF - a
	default:
		return "&H00FFFFFF"
	}
	return fmt.Sprintf("&H%02X%02X%02X%02X", a, b, g, r)
}

func formatASSTime(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	cs := ms / 10 // centiseconds
	h := cs / 360000
	cs %= 360000
	m := cs / 6000
	cs %= 6000
	s := cs / 100
	cs %= 100
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

func formatSRTTime(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	h := ms / 3_600_000
	ms %= 3_600_000
	m := ms / 60_000
	ms %= 60_000
	s := ms / 1000
	milli := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, milli)
}
