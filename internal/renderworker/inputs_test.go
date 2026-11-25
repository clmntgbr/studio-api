package renderworker

import "testing"

func TestUniqueMediaKeys(t *testing.T) {
	keys := []string{
		"media/a/source/original.mp4",
		"media/a/source/original.mp4",
		"media/b/source/original.mp4",
		"media/a/source/original.mp4",
	}
	unique, idx := uniqueMediaKeys(keys)
	if len(unique) != 2 {
		t.Fatalf("unique=%d want 2: %v", len(unique), unique)
	}
	if unique[0] != keys[0] || unique[1] != keys[2] {
		t.Fatalf("unique order wrong: %v", unique)
	}
	want := []int{0, 0, 1, 0}
	for i := range want {
		if idx[i] != want[i] {
			t.Fatalf("idx[%d]=%d want %d", i, idx[i], want[i])
		}
	}
}
