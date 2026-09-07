package render

import (
	"fmt"
	"strings"
	"testing"

	"breakout-terminal/internal/game"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Derive occupied terminal cells from the real collision rectangles, not from
// the renderer's glyph buffer: a missing glyph must not erase the test oracle.
func TestASCIICampaignOccupiedRegions(t *testing.T) {
	sizes := [][2]int{{40, 20}, {41, 20}, {60, 24}, {80, 24}, {60, 30}, {80, 50}, {120, 75}}
	for _, size := range sizes {
		for level := 1; level <= game.LevelCount(); level++ {
			t.Run(fmt.Sprintf("%dx%d/level%d", size[0], size[1], level), func(t *testing.T) {
				g := game.New()
				g.StartLevel(level)
				s := g.Snapshot()
				tr := newTransform(size[0], size[1])
				occupied := make(map[[2]int]string)
				for _, b := range s.Bricks {
					if b.HP == 0 || b.Boss {
						continue // Boss silhouettes are drawn separately.
					}
					for hy := tr.hy(b.Y); hy <= tr.hy(b.Y+b.H); hy++ {
						for x := tr.x(b.X); x <= tr.x(b.X+b.W); x++ {
							occupied[[2]int{x, hy / 2}] = b.ID
						}
					}
				}
				for _, profile := range []ColorProfile{TrueColor, ANSI256, ANSI16, Monochrome} {
					r := New(Options{ASCII: true, Profile: profile})
					frame := ansi.Strip(r.Render(s, size[0], size[1]))
					if lipgloss.Width(frame) != size[0] || lipgloss.Height(frame) != size[1] {
						t.Fatalf("profile %d: wrong frame dimensions", profile)
					}
					for _, ch := range frame {
						if ch > 127 {
							t.Fatalf("profile %d: non-ASCII glyph %q", profile, ch)
						}
					}
					rows := strings.Split(frame, "\n")
					for pos, id := range occupied {
						if ch := rows[pos[1]][pos[0]]; ch == '.' || ch == ' ' {
							t.Fatalf("profile %d: live brick %s at cell %v rendered as %q", profile, id, pos, ch)
						}
					}
				}
			})
		}
	}
}

func TestASCIICollapsedCampaignBrickIdentity(t *testing.T) {
	// L1-3-2 is one of the final review's invisible bricks. Keep its actual
	// rectangle, testing damage/material states without enlarging its geometry.
	g := game.New()
	g.StartLevel(1)
	var brick game.BrickView
	for _, b := range g.Snapshot().Bricks {
		if b.ID == "L1-3-2" {
			brick = b
		}
	}
	if brick.ID == "" {
		t.Fatal("campaign regression brick missing")
	}
	tr := newTransform(40, 20)
	y0, y1 := tr.hy(brick.Y), tr.hy(brick.Y+brick.H)
	if y0 != y1 || y0%2 != 1 {
		t.Fatalf("regression must occupy one odd half-cell; got %d..%d", y0, y1)
	}
	for _, tc := range []struct {
		name      string
		hp, maxHP int
		steel     bool
		special   string
		glyph     byte
	}{
		{"normal", 1, 1, false, "", '='},
		{"reinforced", 2, 2, false, "", '%'},
		{"damaged", 1, 2, false, "", '+'},
		{"steel", -1, -1, true, "", '#'},
		{"special", 1, 1, false, "DHH", 'D'},
		{"destroyed", 0, 1, false, "", '.'},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := brick
			b.HP, b.MaxHP, b.Steel, b.Special = tc.hp, tc.maxHP, tc.steel, tc.special
			s := g.Snapshot()
			s.Bricks = []game.BrickView{b} // Isolate identity from neighboring cells.
			r := New(Options{ASCII: true, Profile: Monochrome})
			rows := strings.Split(r.Render(s, 40, 20), "\n")
			x := tr.x(b.X + b.W/2)
			if got := rows[y0/2][x]; got != tc.glyph {
				t.Fatalf("%s at original campaign coordinates rendered %q, want %q", b.ID, got, tc.glyph)
			}
		})
	}
}

func TestASCIIHalfCellMaterialCollapse(t *testing.T) {
	for kind, want := range map[string]string{"normal": "=", "reinforced": "%", "damaged": "+", "steel": "#"} {
		for _, half := range []string{"top", "bottom", "both"} {
			t.Run(kind+"/"+half, func(t *testing.T) {
				cell := halfCell{top: "playfield", bottom: "playfield", topKind: "playfield", bottomKind: "playfield"}
				if half != "bottom" {
					cell.topKind = kind
				}
				if half != "top" {
					cell.bottomKind = kind
				}
				// Identical color roles deliberately make color unusable as
				// an occupancy signal. Only the material identifies the brick.
				r := New(Options{ASCII: true, Profile: Monochrome})
				if got := r.compose([][]halfCell{{cell}}); got != want {
					t.Fatalf("got %q, want material glyph %q", got, want)
				}
			})
		}
	}
	// One ASCII cell cannot show two materials: the lower occupied half wins,
	// while an explicit special label takes precedence over both materials.
	cell := halfCell{topKind: "steel", bottomKind: "damaged"}
	r := New(Options{ASCII: true, Profile: Monochrome})
	if got := r.compose([][]halfCell{{cell}}); got != "+" {
		t.Fatalf("mixed materials rendered %q, want lower material +", got)
	}
	cell.r = 'D'
	if got := r.compose([][]halfCell{{cell}}); got != "D" {
		t.Fatalf("special label rendered %q, want D", got)
	}
}
