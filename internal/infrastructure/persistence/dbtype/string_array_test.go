package dbtype

import "testing"

func TestStringArrayRoundTrip(t *testing.T) {
	a := StringArray{"foo", "bar baz"}
	v, err := a.Value()
	if err != nil {
		t.Fatal(err)
	}
	var got StringArray
	if err := got.Scan(v); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar baz" {
		t.Fatalf("got %#v", got)
	}
}
