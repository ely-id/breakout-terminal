package render

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"breakout-terminal/internal/game"
	"charm.land/lipgloss/v2"
)

func activeSnapshot() game.Snapshot {
	g := game.New()
	g.StartLevel(6)
	g.Launch()
	for range 5 {
		g.Advance(.05)
	}
	return g.Snapshot()
}

func TestRenderSizesProfilesAndGoldenFrames(t *testing.T) {
	s := activeSnapshot()
	sizes := [][2]int{{40, 20}, {60, 30}, {80, 50}, {120, 75}}
	profiles := []ColorProfile{TrueColor, ANSI256, ANSI16, Monochrome}
	expected := map[string]string{
		"40x20/0": "e7ecc76543ab3c44cc50012955134efb0e0f832011b6fbf2e3ba1013490050e5", "40x20/1": "f5f64e5e04c9dc222fb0433da8687987e0871d1706bb8dae1fe402d4e782d42b", "40x20/2": "b7b21896ddc3815a6d20e6c9e730229dc36c404380f81e25bfa07ee2653fbca8", "40x20/3": "b26f7aef3a8c8fb181f7964705ea705685b36db8d98e1d114cf8977ba3dabbcf",
		"60x30/0": "b77c71ee32a0a6d8449062da21bfe48109a9018a26024424859f19723e147e95", "60x30/1": "58e620db73f31406a0cf1d45a12ac3455e5c5cf1fbda9deecba2a38e4862a68c", "60x30/2": "39357097fa6e1ce79cb23b2fb9bd9c222734e51a4bbb3f6e644ab295ac891cb1", "60x30/3": "4d9e153d6bb6a8eab9c2c729c5ed7ac2b98f212df217e31ea258f10d9fb861eb",
		"80x50/0": "f890d60431e4bbee690ba1ef0d37f9bd3a1c3fce85a9535dfd2e9c565c5cacd7", "80x50/1": "59cfc69aa85a65b6ca9dfbf5621e703afe254fe0b283edc01e3ff1ea287c6f86", "80x50/2": "e89f1788ececfb74509ad4ab653f3c6eef894cdb882f94da908a3e977cedbb49", "80x50/3": "97132cd36827164be6956ea3c89c37d2f42875bc9606f2d7efb88114764678bf",
		"120x75/0": "f55c025a0f319ebe631da4c8147a5bd25feaca56912357d351dbf9ae646c8677", "120x75/1": "66144866b09360596c8fdda947ef1135ae916502999cba157089da7b1f42a672", "120x75/2": "587470093c1c734357ca975a4c8f955af06cc77ebb092007e3800c0afc3826c0", "120x75/3": "11f33a443eb098ad76a7eaaac9f73447a7c786c7199a65a00791af1cd7b89ab1",
	}
	for _, size := range sizes {
		for _, profile := range profiles {
			key := fmt.Sprintf("%dx%d/%d", size[0], size[1], profile)
			frame := New(Options{Profile: profile}).Render(s, size[0], size[1])
			if lipgloss.Width(frame) != size[0] || lipgloss.Height(frame) != size[1] {
				t.Fatalf("%s dimensions %dx%d", key, lipgloss.Width(frame), lipgloss.Height(frame))
			}
			if !strings.Contains(frame, "BREAKOUT") || (!strings.Contains(frame, "BALLS 1") && !strings.Contains(frame, "BALL:1")) {
				t.Fatalf("%s missing status", key)
			}
			sum := fmt.Sprintf("%x", sha256.Sum256([]byte(frame)))
			if want := expected[key]; want != "" && sum != want {
				t.Errorf("%s golden mismatch got %s want %s", key, sum, want)
			} else if want == "" {
				t.Fatalf("%s has no golden", key)
			}
		}
	}
}

func TestMonochromeAndASCII(t *testing.T) {
	s := activeSnapshot()
	mono := New(Options{Profile: Monochrome}).Render(s, 60, 30)
	if strings.Contains(mono, "\x1b[") {
		t.Fatal("monochrome emitted ANSI color")
	}
	ascii := New(Options{ASCII: true, Profile: Monochrome}).Render(s, 60, 30)
	for _, r := range ascii {
		if r > 127 {
			t.Fatalf("ASCII frame contains %q", r)
		}
	}
	if !strings.Contains(ascii, "<- ->") || !strings.Contains(ascii, "o") {
		t.Fatal("ASCII controls/ball missing")
	}
}

func TestUniformLetterboxedTransform(t *testing.T) {
	for _, size := range [][2]int{{80, 30}, {120, 20}, {40, 75}, {97, 43}} {
		tr := newTransform(size[0], size[1])
		sx := float64(tr.x(game.WorldWidth)-tr.x(0)) / game.WorldWidth
		sy := float64(tr.hy(game.WorldHeight)-tr.hy(0)) / game.WorldHeight
		if math.Abs(sx-sy) > .003 {
			t.Errorf("%dx%d nonuniform projection: x=%f half-y=%f", size[0], size[1], sx, sy)
		}
		left, right := tr.x(0)-tr.xMin, tr.xMax-tr.x(game.WorldWidth)
		top, bottom := tr.hy(0)-tr.hyMin, tr.hyMax-tr.hy(game.WorldHeight)
		if absInt(left-right) > 1 || absInt(top-bottom) > 1 {
			t.Errorf("%dx%d world not centered: horizontal %d/%d vertical %d/%d", size[0], size[1], left, right, top, bottom)
		}
	}
}

func TestEveryDropKindIsDistinguishable(t *testing.T) {
	s := activeSnapshot()
	kinds := []string{"wide", "slow", "life", "multi", "fire", "sticky", "shield", "magnet"}
	s.Drops = nil
	for i, kind := range kinds {
		s.Drops = append(s.Drops, game.Drop{X: 90 + float64(i)*88, Y: 560, Kind: kind})
	}
	for _, profile := range []ColorProfile{TrueColor, Monochrome} {
		frame := New(Options{Profile: profile}).Render(s, 120, 75)
		for _, glyph := range []string{"↔", "S", "+", "×3", "F", "G", "▰", "U"} {
			if !strings.Contains(frame, glyph) {
				t.Errorf("profile %d missing drop glyph %q", profile, glyph)
			}
		}
	}
	ascii := New(Options{ASCII: true, Profile: Monochrome}).Render(s, 120, 75)
	for _, glyph := range []string{"W", "S", "+", "3", "F", "G", "#", "U"} {
		if !strings.Contains(ascii, glyph) {
			t.Errorf("ASCII missing drop glyph %q", glyph)
		}
	}
	seenUnicode, seenASCII := map[string]bool{}, map[string]bool{}
	for _, kind := range kinds {
		style := dropStyles[kind]
		if seenUnicode[style.unicode] || seenASCII[style.ascii] {
			t.Errorf("drop %q does not have unique encodings", kind)
		}
		seenUnicode[style.unicode], seenASCII[style.ascii] = true, true
	}
}

func semanticSnapshot() game.Snapshot {
	s := activeSnapshot()
	s.Bricks = []game.BrickView{
		{ID: "normal", X: 100, Y: 240, W: 100, H: 50, HP: 1, MaxHP: 1, Palette: "red"},
		{ID: "reinforced", X: 250, Y: 240, W: 100, H: 50, HP: 2, MaxHP: 2, Palette: "blue"},
		{ID: "damaged", X: 400, Y: 240, W: 100, H: 50, HP: 1, MaxHP: 2, Palette: "green"},
		{ID: "steel", X: 550, Y: 240, W: 100, H: 50, HP: -1, MaxHP: -1, Steel: true, Palette: "steel"},
		{ID: "special", X: 325, Y: 340, W: 150, H: 50, HP: 1, MaxHP: 1, Palette: "yellow", Special: "DHH"},
	}
	return s
}

func TestMonochromeAndASCIIPreserveBoardSemantics(t *testing.T) {
	s := semanticSnapshot()
	mono := New(Options{Profile: Monochrome}).Render(s, 120, 75)
	for _, glyph := range []string{"┌", "─", "│", "█", "▓", "░", "#", "DHH"} {
		if !strings.Contains(mono, glyph) {
			t.Errorf("monochrome frame missing semantic glyph %q", glyph)
		}
	}
	ascii := New(Options{ASCII: true, Profile: Monochrome}).Render(s, 120, 75)
	for _, glyph := range []string{"+", "-", "|", "=", "%", "#", "DHH"} {
		if !strings.Contains(ascii, glyph) {
			t.Errorf("ASCII frame missing semantic glyph %q", glyph)
		}
	}
}

func TestCompactAndTooSmallLayoutsFit(t *testing.T) {
	s := activeSnapshot()
	for _, width := range []int{40, 41} {
		frame := New(Options{ASCII: true, Profile: Monochrome}).Render(s, width, 20)
		if lipgloss.Width(frame) != width || lipgloss.Height(frame) != 20 {
			t.Fatalf("compact %d dimensions are %dx%d", width, lipgloss.Width(frame), lipgloss.Height(frame))
		}
		for _, field := range []string{"00000000", "L06/25", "LIFE:3", "SPACE GO", "P PAUSE", "ESC QUIT"} {
			if !strings.Contains(frame, field) {
				t.Errorf("compact %d missing %q", width, field)
			}
		}
	}
	for _, size := range [][2]int{{39, 19}, {10, 3}, {1, 1}} {
		frame := New(Options{Profile: Monochrome}).Render(s, size[0], size[1])
		if lipgloss.Width(frame) != size[0] || lipgloss.Height(frame) != size[1] {
			t.Errorf("small %dx%d rendered %dx%d", size[0], size[1], lipgloss.Width(frame), lipgloss.Height(frame))
		}
	}
}

func TestSelectedUnicodeGlyphsAreOneCell(t *testing.T) {
	for _, glyph := range []string{"↔", "×", "●", "·", "▰", "←", "→", "▀", "▄", "█", "▓", "░", "─", "│"} {
		if width := lipgloss.Width(glyph); width != 1 {
			t.Errorf("glyph %q occupies %d cells", glyph, width)
		}
	}
}

func TestBossKindsAndFacesAreDistinct(t *testing.T) {
	s := activeSnapshot()
	s.Bricks = nil
	kinds := []struct {
		kind, face, glyph string
	}{
		{"creep", "lookLeft", "C"},
		{"reload", "lookRight", "R"},
		{"maze", "ouch", "M"},
		{"monolith", "blink", "I"},
		{"apple", "shoot", "@"},
	}
	hashes := map[[32]byte]string{}
	for _, tc := range kinds {
		boss := game.BrickView{ID: tc.kind, X: 280, Y: 320, W: 240, H: 160, HP: 5, MaxHP: 10, Palette: "red", Boss: true, BossName: "BOSS", BossKind: tc.kind, BossFace: tc.face}
		s.Boss = &boss
		frame := New(Options{ASCII: true, Profile: Monochrome}).Render(s, 120, 75)
		if strings.Count(frame, tc.glyph) < 4 {
			t.Errorf("boss %s lacks recognizable %q chassis", tc.kind, tc.glyph)
		}
		face := map[string]string{"lookLeft": "[<o]", "lookRight": "[o>]", "ouch": "[XX]", "blink": "[--]", "shoot": "[OO]"}[tc.face]
		if !strings.Contains(frame, face) {
			t.Errorf("boss %s missing face %s", tc.kind, face)
		}
		hash := sha256.Sum256([]byte(frame))
		if other, duplicate := hashes[hash]; duplicate {
			t.Errorf("boss %s frame duplicates %s", tc.kind, other)
		}
		hashes[hash] = tc.kind
	}
	faces := map[string]string{
		"angry": "[><]", "ouch": "[XX]", "shoot": "[OO]", "grin": "[^v]",
		"blink": "[--]", "lookLeft": "[<o]", "lookRight": "[o>]",
	}
	for face, glyph := range faces {
		boss := game.BrickView{ID: face, X: 280, Y: 320, W: 240, H: 160, HP: 5, MaxHP: 10, Palette: "red", Boss: true, BossName: "BOSS", BossKind: "creep", BossFace: face}
		s.Boss = &boss
		frame := New(Options{Profile: Monochrome}).Render(s, 120, 75)
		if !strings.Contains(frame, glyph) {
			t.Errorf("boss face %s missing %s", face, glyph)
		}
	}
}

func TestTrailsAndImpactEventsArePresented(t *testing.T) {
	s := semanticSnapshot()
	s.Balls = []game.BallView{{X: 600, Y: 700, Trail: []game.Point{{X: 100, Y: 700}, {X: 220, Y: 650}, {X: 340, Y: 600}}}}
	s.BallCount = 1
	s.Events = []game.Event{{Type: "brickDamaged", Data: map[string]any{"id": "reinforced"}}}
	frame := New(Options{Profile: Monochrome}).Render(s, 120, 75)
	if !strings.Contains(frame, "·") || !strings.Contains(frame, "*") {
		t.Fatal("trail or impact marker missing")
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func TestTooSmallAndResizeDoNotAffectSnapshot(t *testing.T) {
	g := game.New()
	g.StartLevel(6)
	g.Launch()
	before := g.Snapshot()
	before.Drops = append(before.Drops, game.Drop{X: 400, Y: 600, VY: 155, Kind: "wide"})
	r := New(Options{Profile: Monochrome})
	small := r.Render(before, 39, 19)
	if !strings.Contains(small, "at least 40x20") {
		t.Fatal(small)
	}
	for _, size := range [][2]int{{40, 20}, {120, 75}, {60, 30}, {80, 50}} {
		_ = r.Render(before, size[0], size[1])
	}
	after := g.Snapshot()
	if before.Level != after.Level || before.Score != after.Score || before.BricksLeft != after.BricksLeft || before.Paddle != after.Paddle || before.BallCount != after.BallCount {
		t.Fatal("render/resize mutated core")
	}
}

func TestDenseFrameShowsActualBallCount(t *testing.T) {
	s := activeSnapshot()
	source := s.Balls[0]
	s.Balls = make([]game.BallView, 100)
	for i := range s.Balls {
		s.Balls[i] = source
		s.Balls[i].X += float64(i % 9)
		s.Balls[i].Y -= float64(i % 5)
	}
	s.BallCount = 100
	frame := New(Options{Profile: Monochrome}).Render(s, 80, 50)
	if !strings.Contains(frame, "BALLS 100") {
		t.Fatal("dense count missing")
	}
}

func BenchmarkComposeOneBall(b *testing.B) {
	s := activeSnapshot()
	r := New(Options{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Render(s, 120, 75)
	}
}
func BenchmarkComposeHundredBalls(b *testing.B) {
	s := activeSnapshot()
	source := s.Balls[0]
	s.Balls = make([]game.BallView, 100)
	for i := range s.Balls {
		s.Balls[i] = source
	}
	s.BallCount = 100
	r := New(Options{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Render(s, 120, 75)
	}
}
func BenchmarkStaticCacheRebuild(b *testing.B) {
	s := activeSnapshot()
	r := New(Options{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.cacheValid = false
		_ = r.Render(s, 120, 75)
	}
}

func TestTenMinuteSimulatedSoak(t *testing.T) {
	if os.Getenv("BREAKOUT_SOAK") != "1" {
		t.Skip("set BREAKOUT_SOAK=1 for the 36,000-frame soak")
	}
	g := game.New()
	g.StartLevel(1)
	g.Launch()
	r := New(Options{Profile: Monochrome})
	snap := g.Snapshot()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	goroutines := runtime.NumGoroutine()
	for frame := 0; frame < 60*60*10; frame++ {
		g.Advance((time.Second / 60).Seconds())
		snap = g.Snapshot()
		if snap.Mode == game.ModeGameOver || snap.Mode == game.ModeVictory {
			g.KeyDown(game.NewGame)
			g.KeyDown(game.Launch)
			snap = g.Snapshot()
		}
		_ = r.Render(snap, 120, 75)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if got := runtime.NumGoroutine(); got > goroutines+2 {
		t.Fatalf("goroutines grew from %d to %d", goroutines, got)
	}
	if after.HeapAlloc > before.HeapAlloc+16<<20 {
		t.Fatalf("retained heap grew from %d to %d", before.HeapAlloc, after.HeapAlloc)
	}
}
