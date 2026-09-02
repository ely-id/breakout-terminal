package app

import (
	"testing"
	"time"

	"breakout-terminal/internal/game"
	"breakout-terminal/internal/storage"
)

type fakeStore struct {
	scores, settings int
	muted            bool
}

func (f *fakeStore) SaveHighScores(game.HighScoreState) error { f.scores++; return nil }
func (f *fakeStore) SaveSettings(s storage.Settings) error {
	f.settings++
	f.muted = s.Muted
	return nil
}

func TestEnhancedPressReleaseAndOpposites(t *testing.T) {
	m := New(Config{})
	m.enhancedEvents = true
	m.game.StartLevel(1)
	m.refresh()
	now := time.Now()
	start := m.snap.Paddle.X
	m.press("a", false, now)
	m.game.Advance((50 * time.Millisecond).Seconds())
	m.press("d", false, now)
	m.game.Advance((50 * time.Millisecond).Seconds())
	m.release("a")
	m.game.Advance((50 * time.Millisecond).Seconds())
	m.release("d")
	m.refresh()
	if m.snap.Paddle.X <= start-1 {
		t.Fatalf("opposite/release handling x=%v", m.snap.Paddle.X)
	}
	stopped := m.snap.Paddle.X
	m.game.Advance((50 * time.Millisecond).Seconds())
	m.refresh()
	if m.snap.Paddle.X != stopped {
		t.Fatal("enhanced release did not stop paddle")
	}
}

func TestLegacyLeaseRefreshAndExpiry(t *testing.T) {
	m := New(Config{})
	m.game.StartLevel(1)
	m.refresh()
	now := time.Now()
	m.press("d", false, now)
	first := m.held["d"].deadline
	m.press("d", false, now.Add(100*time.Millisecond))
	if !m.held["d"].deadline.After(first) {
		t.Fatal("repeat did not refresh lease")
	}
	m.expireLegacy(now.Add(200 * time.Millisecond))
	if _, ok := m.held["d"]; !ok {
		t.Fatal("refreshed lease expired too early")
	}
	m.expireLegacy(now.Add(250 * time.Millisecond))
	if _, ok := m.held["d"]; ok {
		t.Fatal("lease did not expire")
	}
}

func TestDirectionRepeatKeepsInitialActionAndCapabilityMigrationClearsLease(t *testing.T) {
	m := New(Config{})
	m.game.StartLevel(1)
	m.refresh()
	now := time.Now()
	m.press("left", false, now)
	first := m.held["left"].deadline
	m.snap.StickyHeld = 1
	m.press("left", false, now.Add(50*time.Millisecond))
	held := m.held["left"]
	if held.action != game.MoveLeft {
		t.Fatalf("repeat changed action to %s", held.action)
	}
	if !held.deadline.After(first) {
		t.Fatal("direction repeat did not refresh lease")
	}
	m.expireLegacy(now.Add(200 * time.Millisecond))
	if _, ok := m.held["left"]; ok {
		t.Fatal("direction lease did not expire")
	}
	n := New(Config{})
	n.game.StartLevel(1)
	n.refresh()
	start := n.snap.Paddle.X
	n.press("d", false, now)
	n.setEnhancedEvents(true)
	if len(n.held) != 0 {
		t.Fatal("legacy entries survived enhancement migration")
	}
	n.game.Advance(.05)
	n.refresh()
	if n.snap.Paddle.X != start {
		t.Fatalf("migrated direction remained active: %.2f", n.snap.Paddle.X)
	}
}

func TestLegacyOneShotGuardCoversInitialAutoRepeat(t *testing.T) {
	now := time.Now()
	m := New(Config{})
	m.press("space", false, now)
	m.refresh()
	if m.snap.Mode != game.ModeServe {
		t.Fatal("initial space did not start")
	}
	for _, d := range []time.Duration{200 * time.Millisecond, 660 * time.Millisecond, 750 * time.Millisecond} {
		m.press("space", false, now.Add(d))
	}
	m.refresh()
	if m.snap.Mode != game.ModeServe {
		t.Fatal("held Space auto-launched")
	}
	m.expireLegacy(now.Add(950 * time.Millisecond))
	m.press("space", false, now.Add(960*time.Millisecond))
	m.refresh()
	if m.snap.Mode != game.ModePlaying {
		t.Fatal("new Space after quiescence did not launch")
	}
	p := New(Config{})
	p.game.StartLevel(1)
	p.refresh()
	p.press("p", false, now)
	for _, d := range []time.Duration{200 * time.Millisecond, 660 * time.Millisecond, 750 * time.Millisecond} {
		p.press("p", false, now.Add(d))
	}
	p.refresh()
	if !p.snap.Paused {
		t.Fatal("held P toggled pause again")
	}
	store := &fakeStore{}
	f := New(Config{Store: store})
	f.press("f10", false, now)
	for _, d := range []time.Duration{200 * time.Millisecond, 660 * time.Millisecond, 750 * time.Millisecond} {
		f.press("f10", false, now.Add(d))
	}
	if !f.muted || store.settings != 1 {
		t.Fatalf("held F10 toggled %d times", store.settings)
	}
}

func TestOneShotRepeatSuppressionAndMute(t *testing.T) {
	store := &fakeStore{}
	m := New(Config{Store: store})
	now := time.Now()
	m.game.StartLevel(1)
	m.refresh()
	m.press("p", false, now)
	m.refresh()
	if !m.snap.Paused {
		t.Fatal("pause not toggled")
	}
	m.press("p", false, now.Add(10*time.Millisecond))
	m.refresh()
	if !m.snap.Paused {
		t.Fatal("repeat toggled pause")
	}
	m.press("f10", false, now)
	m.press("f10", false, now.Add(10*time.Millisecond))
	if !m.muted || store.settings != 1 {
		t.Fatal("F10 repeat was not suppressed")
	}
}

func TestStickyArrowsVersusAD(t *testing.T) {
	m := New(Config{})
	m.enhancedEvents = true
	m.snap.StickyHeld = 1
	now := time.Now()
	m.press("left", false, now)
	if m.held["left"].action != game.AimLeft {
		t.Fatal("sticky arrow did not aim")
	}
	m.press("a", false, now)
	if m.held["a"].action != game.MoveLeft {
		t.Fatal("A did not move during sticky")
	}
}

func TestTypedAndKonamiSecrets(t *testing.T) {
	m := New(Config{})
	m.enhancedEvents = true
	now := time.Now()
	for _, key := range []string{"d", "h", "h"} {
		m.press(key, false, now)
		m.release(key)
	}
	m.refresh()
	if !m.snap.Secrets.DHH {
		t.Fatal("DHH secret missing")
	}
	for _, key := range []string{"up", "up", "down", "down", "left", "right", "left", "right", "b", "a", "enter"} {
		m.press(key, false, now)
		m.release(key)
	}
	m.refresh()
	if !m.snap.Secrets.Konami || m.snap.BallCount != 100 || m.snap.Mode != game.ModePlaying {
		t.Fatalf("Konami failed balls=%d mode=%s", m.snap.BallCount, m.snap.Mode)
	}
}

func TestResizePauseAndQuit(t *testing.T) {
	m := New(Config{})
	m.game.StartLevel(1)
	m.refresh()
	m.width, m.height = 39, 19
	m.applyResizePause()
	m.refresh()
	if !m.pausedByResize || !m.snap.Paused {
		t.Fatal("small viewport did not pause")
	}
	m.width, m.height = 80, 30
	m.applyResizePause()
	m.refresh()
	if m.pausedByResize || m.snap.Paused {
		t.Fatal("resize pause did not restore")
	}
	if !m.press("escape", false, time.Now()) {
		t.Fatal("escape did not quit")
	}
	if !m.press("ctrl+c", false, time.Now()) {
		t.Fatal("ctrl+c did not quit")
	}
}
