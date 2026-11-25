package mastersplit

import "testing"

func TestParse(t *testing.T) {
	s, err := Parse("parts:5")
	if err != nil || s.Parts != 5 {
		t.Fatalf("%+v %v", s, err)
	}
	s, err = Parse("durationMs:600000")
	if err != nil || s.DurationMs != 600000 {
		t.Fatalf("%+v %v", s, err)
	}
	s, err = Parse("")
	if err != nil || !s.IsZero() {
		t.Fatalf("%+v %v", s, err)
	}
	if _, err := Parse("parts:1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPartWindows_Parts(t *testing.T) {
	s := Spec{Parts: 3}
	w := s.PartWindows(90_000)
	if len(w) != 3 {
		t.Fatalf("len=%d", len(w))
	}
	if w[0][0] != 0 || w[2][1] != 90_000 {
		t.Fatalf("%v", w)
	}
}

func TestPartWindows_Duration(t *testing.T) {
	s := Spec{DurationMs: 600_000}
	w := s.PartWindows(1_500_000)
	if len(w) != 3 {
		t.Fatalf("len=%d %#v", len(w), w)
	}
	if w[0][1]-w[0][0] != 600_000 {
		t.Fatalf("first=%v", w[0])
	}
}

func TestPartWindows_ShortNoSplit(t *testing.T) {
	s := Spec{DurationMs: 600_000}
	if w := s.PartWindows(300_000); w != nil {
		t.Fatalf("expected nil, got %#v", w)
	}
}
