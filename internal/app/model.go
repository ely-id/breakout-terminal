// Package app adapts Bubble Tea events and monotonic frame delivery to the
// framework-independent game core.
package app

import (
	"strings"
	"time"

	"breakout-terminal/internal/audio"
	"breakout-terminal/internal/game"
	brender "breakout-terminal/internal/render"
	"breakout-terminal/internal/storage"
	tea "charm.land/bubbletea/v2"
)

const (
	frameInterval             = time.Second / 60
	legacyDirectionLease      = 140 * time.Millisecond
	legacyOneShotInitialGuard = 800 * time.Millisecond
	legacyOneShotRepeatGuard  = 180 * time.Millisecond
)

type frameMsg time.Time

type StateStore interface {
	SaveHighScores(game.HighScoreState) error
	SaveSettings(storage.Settings) error
}

type heldKey struct {
	action    game.Action
	deadline  time.Time
	direction bool
	legacy    bool
}

type Config struct {
	Game     *game.Game
	Renderer *brender.Renderer
	Store    StateStore
	Muted    bool
}

type Model struct {
	game           *game.Game
	renderer       *brender.Renderer
	store          StateStore
	snap           game.Snapshot
	width, height  int
	lastFrame      time.Time
	enhancedEvents bool
	held           map[string]heldKey
	pausedByResize bool
	muted          bool
	lastCue        string
}

func New(cfg Config) *Model {
	if cfg.Game == nil {
		cfg.Game = game.New()
	}
	if cfg.Renderer == nil {
		cfg.Renderer = brender.New(brender.Options{})
	}
	m := &Model{game: cfg.Game, renderer: cfg.Renderer, store: cfg.Store, held: make(map[string]heldKey), muted: cfg.Muted, width: 80, height: 30}
	m.renderer.SetMuted(m.muted)
	m.snap = m.game.Snapshot()
	return m
}

func nextFrame() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg { return frameMsg(t) })
}
func (m *Model) Init() tea.Cmd { return nextFrame() }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case frameMsg:
		now := time.Time(msg)
		m.expireLegacy(now)
		if m.lastFrame.IsZero() {
			m.lastFrame = now
		} else {
			m.game.Advance(now.Sub(m.lastFrame).Seconds())
			m.lastFrame = now
		}
		m.refresh()
		return m, nextFrame()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applyResizePause()
		m.refresh()
	case tea.KeyboardEnhancementsMsg:
		m.setEnhancedEvents(msg.SupportsEventTypes())
	case tea.KeyPressMsg:
		if m.press(msg.String(), msg.Key().IsRepeat, time.Now()) {
			m.persist()
			return m, tea.Quit
		}
		m.refresh()
	case tea.KeyReleaseMsg:
		m.release(msg.String())
		m.refresh()
	}
	return m, nil
}

func (m *Model) refresh() {
	m.snap = m.game.Snapshot()
	for _, e := range m.snap.Events {
		if e.Type == "highScoreUpdated" && m.store != nil {
			_ = m.store.SaveHighScores(m.game.HighScoreState())
		}
	}
	if !m.muted {
		m.lastCue = audio.Select(m.snap.Events)
	}
}

func (m *Model) applyResizePause() {
	tooSmall := m.width < 40 || m.height < 20
	if tooSmall && !m.pausedByResize && !m.snap.Paused {
		if m.game.TogglePause() {
			m.pausedByResize = true
		}
	}
	if !tooSmall && m.pausedByResize {
		m.game.TogglePause()
		m.pausedByResize = false
	}
}

func (m *Model) expireLegacy(now time.Time) {
	for key, h := range m.held {
		if h.legacy && !h.deadline.IsZero() && !now.Before(h.deadline) {
			if h.direction {
				m.game.KeyUp(h.action)
			}
			delete(m.held, key)
		}
	}
}

func (m *Model) setEnhancedEvents(supported bool) {
	if supported == m.enhancedEvents {
		return
	}
	if supported {
		// A key pressed before capability negotiation is not guaranteed to emit
		// an enhanced release. Release its leased intent before changing modes.
		for key, h := range m.held {
			if h.legacy && h.direction {
				m.game.KeyUp(h.action)
			}
			delete(m.held, key)
		}
	}
	m.enhancedEvents = supported
}

func normalizeKey(key string) string {
	key = strings.ToLower(key)
	if key == "esc" {
		return "escape"
	}
	if key == "return" {
		return "enter"
	}
	return key
}

// press returns true when the application should quit.
func (m *Model) press(raw string, reportedRepeat bool, now time.Time) bool {
	key := normalizeKey(raw)
	if key == "escape" || key == "ctrl+c" {
		return true
	}
	h, already := m.held[key]
	if already && h.legacy && !h.deadline.IsZero() && !now.Before(h.deadline) {
		if h.direction {
			m.game.KeyUp(h.action)
		}
		delete(m.held, key)
		already = false
	}
	if already {
		// The initial direction choice is authoritative. Repeats only renew its
		// lease; they never re-evaluate sticky state or issue another KeyDown.
		if h.legacy {
			if h.direction {
				h.deadline = now.Add(legacyDirectionLease)
			} else {
				// A duplicate before auto-repeat starts must not shorten the
				// longer initial guard. Once repeating, each event extends the
				// quiescence deadline without moving it backwards.
				deadline := now.Add(legacyOneShotRepeatGuard)
				if deadline.After(h.deadline) {
					h.deadline = deadline
				}
			}
			m.held[key] = h
		}
		return false
	}
	// A repeat without a matching initial entry can occur after capability
	// migration. Ignoring it avoids reviving an intent that was safely cleared.
	if reportedRepeat {
		return false
	}
	rememberDirection := func(action game.Action) {
		h := heldKey{action: action, direction: true, legacy: !m.enhancedEvents}
		if h.legacy {
			h.deadline = now.Add(legacyDirectionLease)
		}
		m.held[key] = h
	}
	rememberOneShot := func() {
		h := heldKey{legacy: !m.enhancedEvents}
		if h.legacy {
			h.deadline = now.Add(legacyOneShotInitialGuard)
		}
		m.held[key] = h
	}
	if key == "f10" {
		m.muted = !m.muted
		m.renderer.SetMuted(m.muted)
		if m.store != nil {
			_ = m.store.SaveSettings(storage.Settings{Version: 1, Muted: m.muted})
		}
		rememberOneShot()
		return false
	}
	if m.width < 40 || m.height < 20 {
		return false
	}
	if len([]rune(key)) == 1 {
		r := []rune(strings.ToUpper(key))[0]
		if r >= 'A' && r <= 'Z' {
			code := false
			if key == "a" || key == "b" {
				code = m.game.InputCodeToken(key)
			}
			secret := m.game.InputCharacter(key)
			if code || secret {
				rememberOneShot()
				return false
			}
		}
	}
	switch key {
	case "up", "down":
		m.game.InputCodeToken(key)
		rememberOneShot()
		return false
	case "left", "right":
		m.game.InputCodeToken(key)
	case "enter":
		if m.game.InputCodeToken("enter") {
			rememberOneShot()
			return false
		}
		m.game.KeyDown(game.NewGame)
		rememberOneShot()
		return false
	case "space":
		m.game.KeyDown(game.Launch)
		rememberOneShot()
		return false
	case "p":
		m.game.KeyDown(game.Pause)
		rememberOneShot()
		return false
	case "b":
		rememberOneShot()
		return false
	}
	action, ok := m.directionAction(key)
	if !ok {
		rememberOneShot()
		return false
	}
	m.game.KeyDown(action)
	rememberDirection(action)
	return false
}

func (m *Model) directionAction(key string) (game.Action, bool) {
	switch key {
	case "a":
		return game.MoveLeft, true
	case "d":
		return game.MoveRight, true
	case "left":
		if m.snap.StickyHeld > 0 {
			return game.AimLeft, true
		}
		return game.MoveLeft, true
	case "right":
		if m.snap.StickyHeld > 0 {
			return game.AimRight, true
		}
		return game.MoveRight, true
	}
	return "", false
}

func (m *Model) release(raw string) {
	key := normalizeKey(raw)
	if h, ok := m.held[key]; ok {
		if h.direction {
			m.game.KeyUp(h.action)
		}
		delete(m.held, key)
	}
	if key == "left" {
		m.game.KeyUp(game.MoveLeft)
		m.game.KeyUp(game.AimLeft)
	}
	if key == "right" {
		m.game.KeyUp(game.MoveRight)
		m.game.KeyUp(game.AimRight)
	}
}

func (m *Model) persist() {
	if m.store != nil {
		_ = m.store.SaveHighScores(m.game.HighScoreState())
		_ = m.store.SaveSettings(storage.Settings{Version: 1, Muted: m.muted})
	}
}

func (m *Model) View() tea.View {
	v := tea.NewView(m.renderer.Render(m.snap, m.width, m.height))
	v.AltScreen = true
	v.WindowTitle = "Breakout"
	v.KeyboardEnhancements.ReportEventTypes = true
	return v
}
