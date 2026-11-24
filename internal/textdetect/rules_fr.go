package textdetect

import (
	"strings"
	"unicode"
)

// RulesFR@1 — French filler dictionary (docs/docs.md).
var fillersGentle = map[string]struct{}{
	"euh": {}, "heu": {}, "hum": {}, "hm": {}, "mmh": {}, "mm": {},
}

var fillersStandard = map[string]struct{}{
	"euh": {}, "heu": {}, "hum": {}, "hm": {}, "mmh": {}, "mm": {},
	"bah": {}, "ben": {},
}

var fillersAggressive = map[string]struct{}{
	"euh": {}, "heu": {}, "hum": {}, "hm": {}, "mmh": {}, "mm": {},
	"bah": {}, "ben": {}, "genre": {}, "du coup": {}, "en fait": {},
	"voila": {}, "voilà": {}, "quoi": {}, "tu vois": {}, "en gros": {},
}

func fillerSet(level string) map[string]struct{} {
	switch level {
	case "gentle":
		return fillersGentle
	case "standard":
		return fillersStandard
	default:
		return fillersAggressive
	}
}

func normalizeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '\'' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
