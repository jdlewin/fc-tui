package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testCasts() []Cast {
	c := Cast{Hash: "0xabc123", Text: "hello world", Timestamp: "2026-11-08T04:52:40.000Z"}
	c.Author.Username = "jdl"
	c.Author.DisplayName = "jd"
	c.Author.Fid = 10695
	c2 := Cast{Hash: "0xdef456", Text: "second cast", Timestamp: "2026-11-08T04:50:40.000Z"}
	c2.Author.Username = "linda"
	return []Cast{c, c2}
}

func sizedModel(casts []Cast) Model {
	m := NewModel("key", "10695", "feed.json", casts)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return um.(Model)
}

func TestListViewRenders(t *testing.T) {
	m := sizedModel(testCasts())
	v := m.View()
	for _, want := range []string{"@jdl", "@linda", "hello world", "j/k move"} {
		if !strings.Contains(v, want) {
			t.Fatalf("list view missing %q:\n%s", want, v)
		}
	}
}

func TestEnterOpensDetail(t *testing.T) {
	m := sizedModel(testCasts())
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = um.(Model)
	if m.mode != modeDetail {
		t.Fatal("enter should open detail")
	}
	if v := m.View(); !strings.Contains(v, "0xabc123") {
		t.Fatalf("detail missing hash:\n%s", v)
	}
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = um.(Model)
	if m.mode != modeList {
		t.Fatal("esc should return to list")
	}
}

func TestNavigation(t *testing.T) {
	m := sizedModel(testCasts())
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = um.(Model)
	if m.cursor != 1 {
		t.Fatalf("j should move to 1, got %d", m.cursor)
	}
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = um.(Model)
	if m.cursor != 0 {
		t.Fatalf("k should move to 0, got %d", m.cursor)
	}
}

func TestComposerFocusCycle(t *testing.T) {
	m := sizedModel(testCasts())
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = um.(Model)
	if !m.composerFocused {
		t.Fatal("i should focus composer")
	}
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = um.(Model)
	if m.composerFocused {
		t.Fatal("esc should blur composer")
	}
}

func TestEmptyFeed(t *testing.T) {
	m := sizedModel(nil)
	if v := m.View(); !strings.Contains(v, "No casts") {
		t.Fatalf("empty state missing:\n%s", v)
	}
}

func TestComposerEnterValidatesEmpty(t *testing.T) {
	m := sizedModel(testCasts())
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = um.(Model)
	m.composer.SetValue("   ")
	um, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = um.(Model)
	if cmd != nil {
		t.Fatal("empty cast should not produce a post command")
	}
	if !m.postIsErr || m.postStatus == "" {
		t.Fatalf("empty cast should set error status, got %q", m.postStatus)
	}
}

func TestComposerEnterPostsValid(t *testing.T) {
	m := sizedModel(testCasts())
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = um.(Model)
	m.composer.SetValue("hello")
	um, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = um.(Model)
	if cmd == nil {
		t.Fatal("valid cast should produce a post command")
	}
	if m.postStatus != "Posting…" {
		t.Fatalf("expected Posting status, got %q", m.postStatus)
	}
}

func TestPostCastNeedsSigner(t *testing.T) {
	if err := postCast("key", "", "hello"); err == nil {
		t.Fatal("missing signer should error before network")
	}
}

func TestReplyKeyTargetsCursorCast(t *testing.T) {
	m := sizedModel(testCasts())
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	m = um.(Model)
	if !m.composerFocused {
		t.Fatal("R should focus composer")
	}
	if m.target.Reply == nil || m.target.Reply.Hash != "0xabc123" || m.target.Reply.Fid != 10695 {
		t.Fatalf("R should target cursor cast, got %+v", m.target)
	}
}

func TestQuoteKeyTargetsCursorCast(t *testing.T) {
	m := sizedModel(testCasts())
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
	m = um.(Model)
	if m.target.Quote == nil || m.target.Quote.Hash != "0xabc123" {
		t.Fatalf("Q should target cursor cast, got %+v", m.target)
	}
}

func TestEscClearsReplyTarget(t *testing.T) {
	m := sizedModel(testCasts())
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	m = um.(Model)
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = um.(Model)
	if m.target.Reply != nil || m.composerFocused {
		t.Fatalf("esc should clear target and blur, got %+v focused=%v", m.target, m.composerFocused)
	}
}

func TestNotifPrimingHushesFirstLoad(t *testing.T) {
	m := sizedModel(testCasts())
	items := []notif{{Type: "mention", Hash: "0x1", Author: "a", Text: "hi"}}
	um, cmd := m.Update(notifsLoadedMsg{items})
	m = um.(Model)
	if cmd != nil {
		t.Fatal("first notif load should stay silent")
	}
	if !m.seenNotifs["mention:0x1"] {
		t.Fatal("first load should mark seen")
	}
	um2, cmd2 := m.Update(notifsLoadedMsg{[]notif{{Type: "mention", Hash: "0x2", Author: "b", Text: "yo"}}})
	_ = um2
	if cmd2 == nil {
		t.Fatal("second load with fresh mention should ring")
	}
}

func TestDebugKeyFiresNotifCmd(t *testing.T) {
	m := sizedModel(testCasts())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Fatal("T should produce a notification command")
	}
}
