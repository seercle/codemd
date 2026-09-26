package lineutil

import "testing"

func TestSplitJoinRoundTrip(t *testing.T) {
	cases := []string{
		"a\nb\nc\n",
		"a\nb\nc",
		"a\r\nb\r\n",
		"",
		"\n",
		"single",
	}
	for _, in := range cases {
		if got := Split(in).Join(); got != in {
			t.Errorf("round trip %q -> %q", in, got)
		}
	}
}

func TestSplitFields(t *testing.T) {
	l := Split("a\r\nb\r\n")
	if len(l.Content) != 2 || l.Content[0] != "a" || l.EOL != "\r\n" || !l.TrailingNewline {
		t.Fatalf("unexpected: %+v", l)
	}
	l2 := Split("x\ny")
	if l2.EOL != "\n" || l2.TrailingNewline {
		t.Fatalf("unexpected: %+v", l2)
	}
}
