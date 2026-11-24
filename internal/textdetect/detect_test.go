package textdetect

import "testing"

func TestDetect_Fillers(t *testing.T) {
	words := []Word{
		{Index: 0, Text: "euh", SourceStartMs: 0, SourceEndMs: 200},
		{Index: 1, Text: "bonjour", SourceStartMs: 300, SourceEndMs: 700},
		{Index: 2, Text: "genre", SourceStartMs: 800, SourceEndMs: 1000},
	}
	out := Detect(words, Options{Level: "aggressive", FillersEnabled: true})
	if len(out) != 2 {
		t.Fatalf("got %#v", out)
	}
	if out[0].Kind != "filler" || out[0].FirstWord != 0 {
		t.Fatalf("%+v", out[0])
	}
}

func TestDetect_Repetitions(t *testing.T) {
	words := []Word{
		{Index: 0, Text: "je", SourceStartMs: 0, SourceEndMs: 100},
		{Index: 1, Text: "je", SourceStartMs: 120, SourceEndMs: 220},
		{Index: 2, Text: "pense", SourceStartMs: 300, SourceEndMs: 600},
	}
	out := Detect(words, Options{RepetitionsEnabled: true})
	if len(out) != 1 || out[0].Kind != "repetition" {
		t.Fatalf("%#v", out)
	}
}
