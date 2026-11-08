package main

import (
	"strings"
	"testing"
)

func TestTimeAgoMinutes(t *testing.T) {
	got := timeAgo("2026-11-08T04:52:40.000Z")
	if got == "" || got == "2026-11-08T04:52:40.000Z" {
		t.Fatalf("expected relative time, got %q", got)
	}
}

func TestTimeAgoBadInput(t *testing.T) {
	if got := timeAgo("not-a-time"); got != "not-a-time" {
		t.Fatalf("bad input should pass through, got %q", got)
	}
}

func TestValidatePost(t *testing.T) {
	if err := validatePost("hello"); err != nil {
		t.Fatal(err)
	}
	if err := validatePost("   "); err == nil {
		t.Fatal("empty cast should fail")
	}
	if err := validatePost(strings.Repeat("x", 321)); err == nil {
		t.Fatal("321-rune cast should fail")
	}
	if err := validatePost(strings.Repeat("x", 320)); err != nil {
		t.Fatal("320-rune cast should pass")
	}
}

func TestOneLine(t *testing.T) {
	if got := oneLine("a\nb\nc", 80); got != "a b c" {
		t.Fatalf("got %q", got)
	}
	if got := oneLine("", 80); got != "(no text)" {
		t.Fatalf("got %q", got)
	}
	if got := oneLine("hello", 3); got != "he…" {
		t.Fatalf("got %q", got)
	}
}

func TestWrapText(t *testing.T) {
	got := wrapText("aa bb cc dd", 10)
	for _, line := range strings.Split(got, "\n") {
		if len([]rune(line)) > 10 {
			t.Fatalf("line too long: %q", line)
		}
	}
	if got != "aa bb cc\ndd" {
		t.Fatalf("unexpected wrap: %q", got)
	}
	// Paragraph breaks preserved.
	got = wrapText("one\n\ntwo", 80)
	if !strings.Contains(got, "one\n\ntwo") {
		t.Fatalf("paragraph break lost: %q", got)
	}
}

func TestDetailContent(t *testing.T) {
	c := Cast{Hash: "0xabc", Text: "hello\nworld", Timestamp: "2026-11-08T04:52:40.000Z"}
	c.Author.Username = "jdl"
	c.Author.DisplayName = "jd"
	c.Author.Fid = 10695
	got := detailContent(c, 40)
	for _, want := range []string{"@jdl", "0xabc", "hello", "world", "10695"} {
		if !strings.Contains(got, want) {
			t.Fatalf("detail missing %q:\n%s", want, got)
		}
	}
}
