// Package render projects immutable game snapshots into bounded terminal cells.
package render

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"math"
	"strings"

	"breakout-terminal/internal/game"
	"charm.land/lipgloss/v2"
)

type ColorProfile int

const (
	TrueColor ColorProfile = iota
	ANSI256
	ANSI16
	Monochrome
)

type Options struct {
	ASCII   bool
	Profile ColorProfile
}

type halfCell struct {
	top, bottom         string
	topKind, bottomKind string
	r                   rune
	fg                  string
}

type cacheKey struct {
	w, h, level int
	hash        uint64
	ascii       bool
}

// Renderer caches the static level raster and overlays dynamic entities on a copy.
type Renderer struct {
	opts       Options
	muted      bool
	styles     map[string]lipgloss.Style
	cacheKey   cacheKey
	static     [][]halfCell
	cacheValid bool
}

func New(opts Options) *Renderer {
	return &Renderer{opts: opts, styles: make(map[string]lipgloss.Style)}
}

func (r *Renderer) SetMuted(muted bool) { r.muted = muted }

var palette = map[string]string{
	"background": "#1a1b26", "foreground": "#c0caf5", "accent": "#7aa2f7",
	"blue": "#7aa2f7", "cyan": "#7dcfff", "green": "#9ece6a", "yellow": "#e0af68",
	"red": "#f7768e", "magenta": "#bb9af7", "bright": "#c0caf5", "muted": "#414868",
	"mutedText": "#7982a9", "surface": "#24283b", "surface2": "#16161e",
	"playfield": "#101119", "border": "#3b4261", "steel": "#565f89", "overlay": "#151515",
}

var ansi256 = map[string]string{
	"background": "234", "foreground": "189", "accent": "111", "blue": "111", "cyan": "117", "green": "149",
	"yellow": "179", "red": "210", "magenta": "183", "bright": "189", "muted": "60", "mutedText": "103",
	"surface": "235", "surface2": "233", "playfield": "233", "border": "60", "steel": "60", "overlay": "234",
}

var ansi16 = map[string]string{
	"background": "0", "foreground": "15", "accent": "12", "blue": "12", "cyan": "14", "green": "10",
	"yellow": "11", "red": "9", "magenta": "13", "bright": "15", "muted": "8", "mutedText": "7",
	"surface": "8", "surface2": "0", "playfield": "0", "border": "8", "steel": "8", "overlay": "0",
}

type dropStyle struct{ unicode, ascii, role string }

var dropStyles = map[string]dropStyle{
	"wide": {"↔", "W", "accent"}, "slow": {"S", "S", "blue"}, "life": {"+", "+", "green"}, "multi": {"×3", "3", "magenta"},
	"fire": {"F", "F", "red"}, "sticky": {"G", "G", "magenta"}, "shield": {"▰", "#", "cyan"}, "magnet": {"U", "U", "yellow"},
}

func (r *Renderer) color(role string) color.Color {
	if r.opts.Profile == Monochrome {
		return lipgloss.NoColor{}
	}
	value := palette[role]
	if r.opts.Profile == ANSI256 {
		value = ansi256[role]
	}
	if r.opts.Profile == ANSI16 {
		value = ansi16[role]
	}
	return lipgloss.Color(value)
}

func (r *Renderer) style(fg, bg string) lipgloss.Style {
	key := fg + "/" + bg
	if s, ok := r.styles[key]; ok {
		return s
	}
	s := lipgloss.NewStyle()
	if r.opts.Profile != Monochrome {
		if fg != "" {
			s = s.Foreground(r.color(fg))
		}
		if bg != "" {
			s = s.Background(r.color(bg))
		}
	}
	r.styles[key] = s
	return s
}

func brickHash(s game.Snapshot) uint64 {
	h := fnv.New64a()
	for _, b := range s.Bricks {
		_, _ = h.Write([]byte(b.ID))
		_, _ = h.Write([]byte{byte(b.HP + 2)})
	}
	return h.Sum64()
}

func dimensions(width, height int) (int, int) {
	if width > 120 {
		width = 120
	}
	if height > 75 {
		height = 75
	}
	return width, height
}

func blank(w, h int) [][]halfCell {
	cells := make([][]halfCell, h)
	for y := range cells {
		cells[y] = make([]halfCell, w)
		for x := range cells[y] {
			cells[y][x].top, cells[y][x].bottom = "background", "background"
			cells[y][x].topKind, cells[y][x].bottomKind = "background", "background"
		}
	}
	return cells
}

func clone(src [][]halfCell) [][]halfCell {
	out := make([][]halfCell, len(src))
	for y := range src {
		out[y] = append([]halfCell(nil), src[y]...)
	}
	return out
}

func setHalf(c [][]halfCell, x, hy int, role string) {
	setHalfSemantic(c, x, hy, role, role)
}

func setHalfSemantic(c [][]halfCell, x, hy int, role, kind string) {
	if hy < 0 || hy >= len(c)*2 || x < 0 || x >= len(c[0]) {
		return
	}
	if hy%2 == 0 {
		c[hy/2][x].top = role
		c[hy/2][x].topKind = kind
	} else {
		c[hy/2][x].bottom = role
		c[hy/2][x].bottomKind = kind
	}
}

func fillHalf(c [][]halfCell, x0, y0, x1, y1 int, role string) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			setHalf(c, x, y, role)
		}
	}
}

func fillSemantic(c [][]halfCell, x0, y0, x1, y1 int, role, kind string) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			setHalfSemantic(c, x, y, role, kind)
		}
	}
}

type transform struct {
	w, h                     int
	scale, x0, y0            float64
	xMin, xMax, hyMin, hyMax int
}

func newTransform(w, h int) transform {
	xMin, xMax, hyMin := 1, w-2, 6
	bottomRow := h - 3
	if h >= 30 {
		bottomRow = h - 4
	}
	hyMax := bottomRow*2 + 1
	scale := math.Min(float64(xMax-xMin)/game.WorldWidth, float64(hyMax-hyMin)/game.WorldHeight)
	return transform{w: w, h: h, scale: scale, x0: (float64(xMin+xMax) - game.WorldWidth*scale) / 2, y0: (float64(hyMin+hyMax) - game.WorldHeight*scale) / 2, xMin: xMin, xMax: xMax, hyMin: hyMin, hyMax: hyMax}
}
func (t transform) x(v float64) int  { return int(math.Round(t.x0 + v*t.scale)) }
func (t transform) hy(v float64) int { return int(math.Round(t.y0 + v*t.scale)) }

func (r *Renderer) buildStatic(s game.Snapshot, w, h int) [][]halfCell {
	c := blank(w, h)
	t := newTransform(w, h)
	borderRole := "border"
	if r.opts.Profile == Monochrome {
		borderRole = "foreground"
	}
	for x := 0; x < w; x++ {
		setHalf(c, x, 0, borderRole)
		setHalf(c, x, 2*h-1, borderRole)
	}
	for y := 0; y < 2*h; y++ {
		setHalf(c, 0, y, borderRole)
		setHalf(c, w-1, y, borderRole)
	}
	r.drawFrame(c)
	fillSemantic(c, t.x(20), t.hy(92), t.x(780), t.hy(946), "playfield", "playfield")
	for _, b := range s.Bricks {
		if b.HP == 0 || b.Boss {
			continue
		}
		role, kind := b.Palette, "normal"
		if b.Steel {
			role = "steel"
			kind = "steel"
		}
		if !b.Steel && b.MaxHP > 1 && b.HP == b.MaxHP {
			kind = "reinforced"
		}
		if b.HP > 0 && b.MaxHP > 1 && b.HP < b.MaxHP {
			role = "muted"
			kind = "damaged"
		}
		x0, x1 := t.x(b.X), t.x(b.X+b.W)
		y0, y1 := t.hy(b.Y), t.hy(b.Y+b.H)
		if x1 < x0 {
			x1 = x0
		}
		if y1 < y0 {
			y1 = y0
		}
		fillSemantic(c, x0, y0, x1, y1, role, kind)
		if r.opts.ASCII {
			ch := '='
			if b.Steel {
				ch = '#'
			} else if b.MaxHP > 1 && b.HP == b.MaxHP {
				ch = '%'
			} else if b.HP > 0 && b.MaxHP > 1 && b.HP < b.MaxHP {
				ch = '+'
			}
			for y := (y0 + 1) / 2; y <= y1/2; y++ {
				for x := x0; x <= x1; x++ {
					if y >= 0 && y < h && x >= 0 && x < w {
						c[y][x].r = ch
						c[y][x].fg = role
					}
				}
			}
		}
		if b.Special != "" {
			glyph := map[string]rune{"DHH": 'D', "RYAN": 'R', "OMARCHY": 'O', "37": '7'}[b.Special]
			x, y := t.x(b.X+b.W/2), t.hy(b.Y+b.H/2)/2
			if y >= 0 && y < h && x > 0 && x < w-1 {
				c[y][x].r = glyph
				c[y][x].fg = "foreground"
			}
		}
	}
	return c
}

func (r *Renderer) drawFrame(c [][]halfCell) {
	h, w := len(c), len(c[0])
	horizontal, vertical, tl, tr, bl, br := '─', '│', '┌', '┐', '└', '┘'
	if r.opts.ASCII {
		horizontal, vertical, tl, tr, bl, br = '-', '|', '+', '+', '+', '+'
	}
	for x := 0; x < w; x++ {
		c[0][x].r = horizontal
		c[0][x].fg = "border"
		c[h-1][x].r = horizontal
		c[h-1][x].fg = "border"
	}
	for y := 0; y < h; y++ {
		c[y][0].r = vertical
		c[y][0].fg = "border"
		c[y][w-1].r = vertical
		c[y][w-1].fg = "border"
	}
	c[0][0].r, c[0][w-1].r, c[h-1][0].r, c[h-1][w-1].r = tl, tr, bl, br
}

func writeText(c [][]halfCell, x, y int, text, role string) {
	if y < 0 || y >= len(c) {
		return
	}
	column := x
	for _, ch := range text {
		glyph := string(ch)
		width := lipgloss.Width(glyph)
		if width != 1 {
			glyph = "?"
			width = 1
		}
		xx := column
		if xx <= 0 || xx >= len(c[y])-1 {
			column += width
			continue
		}
		c[y][xx].r = []rune(glyph)[0]
		c[y][xx].fg = role
		column += width
	}
}
func centered(c [][]halfCell, y int, text, role string) {
	writeText(c, (len(c[0])-lipgloss.Width(text))/2, y, text, role)
}

func (r *Renderer) overlayDynamic(c [][]halfCell, s game.Snapshot) {
	h, w := len(c), len(c[0])
	t := newTransform(w, h)
	if len(s.Balls) <= 5 {
		for _, b := range s.Balls {
			for _, point := range b.Trail {
				x, y := t.x(point.X), t.hy(point.Y)/2
				if x > 0 && x < w-1 && y > 2 && y < h-2 {
					c[y][x].r = map[bool]rune{true: '.', false: '·'}[r.opts.ASCII]
					c[y][x].fg = "mutedText"
				}
			}
		}
	}
	for _, b := range s.Balls {
		x, y := t.x(b.X), t.hy(b.Y)/2
		if x > 0 && x < w-1 && y > 1 && y < h-1 {
			c[y][x].r = map[bool]rune{true: 'o', false: '●'}[r.opts.ASCII]
			c[y][x].fg = map[bool]string{true: "yellow", false: "foreground"}[s.FireLeft > 0]
		}
	}
	px0, px1 := t.x(s.Paddle.X-s.Paddle.W/2), t.x(s.Paddle.X+s.Paddle.W/2)
	py := t.hy(s.Paddle.Y) / 2
	paddleRole := "accent"
	if s.StickyLeft > 0 {
		paddleRole = "magenta"
	} else if s.MagnetLeft > 0 {
		paddleRole = "cyan"
	} else if s.DHHLeft > 0 {
		paddleRole = "yellow"
	}
	for x := px0; x <= px1; x++ {
		if py >= 0 && py < h && x > 0 && x < w-1 {
			c[py][x].r = map[bool]rune{true: '=', false: '▀'}[r.opts.ASCII]
			c[py][x].fg = paddleRole
		}
	}
	for _, d := range s.Drops {
		x, y := t.x(d.X), t.hy(d.Y)/2
		if x > 0 && x < w-1 && y > 1 && y < h-1 {
			style, ok := dropStyles[d.Kind]
			if !ok {
				continue
			}
			glyph := style.unicode
			if r.opts.ASCII {
				glyph = style.ascii
			}
			writeText(c, x-lipgloss.Width(glyph)/2, y, glyph, style.role)
		}
	}
	for _, shot := range s.BossShots {
		x, y := t.x(shot.X), t.hy(shot.Y)/2
		if x > 0 && x < w-1 && y > 1 && y < h-1 {
			c[y][x].r = 'o'
			c[y][x].fg = []string{"magenta", "yellow", "red"}[clampIndex(shot.Stage-1, 0, 2)]
		}
	}
	if s.ShieldCharges > 0 {
		y := t.hy(game.ShieldY) / 2
		for x := t.x(24); x <= t.x(776); x++ {
			c[y][x].r = map[bool]rune{true: '-', false: '─'}[r.opts.ASCII]
			c[y][x].fg = "cyan"
		}
	}
	if s.StickyHeld > 0 {
		originX, originY := t.x(s.Paddle.X), t.hy(s.Paddle.Y)/2-1
		for i := 1; i <= 5; i++ {
			x := originX + int(math.Round(math.Sin(s.StickyAim)*float64(i)))
			y := originY - i
			if x > 0 && x < w-1 && y > 2 && y < h-2 {
				c[y][x].r = map[bool]rune{true: '.', false: '·'}[r.opts.ASCII]
				c[y][x].fg = "magenta"
			}
		}
	}
	if s.Boss != nil {
		r.drawBoss(c, t, *s.Boss)
	}
	r.drawImpacts(c, t, s)
	if w >= 80 {
		for _, b := range s.Bricks {
			if b.HP > 0 && b.Special != "" {
				x := t.x(b.X+b.W/2) - lipgloss.Width(b.Special)/2
				y := t.hy(b.Y+b.H/2) / 2
				writeText(c, x, y, r.clean(b.Special), "foreground")
			}
		}
	}
	centered(c, 0, "BREAKOUT", "accent")
	stats := fmt.Sprintf("%08d  LV %02d/%02d  %s  LIVES %d", s.Score, s.Level, s.LevelCount, r.truncate(r.clean(s.LevelName), w/3), s.Lives)
	if w < 60 {
		stats = fmt.Sprintf("%08d L%02d/%02d LIFE:%d", s.Score, s.Level, s.LevelCount, s.Lives)
	}
	centered(c, 1, stats, "foreground")
	if s.Message != "" {
		centered(c, 2, r.truncate(r.clean(s.Message), w-4), "yellow")
	}
	status := fmt.Sprintf("x%d  BRICKS %d  BALLS %d", s.Multiplier, s.BricksLeft, s.BallCount)
	if w < 60 {
		status = fmt.Sprintf("x%d B:%d BALL:%d", s.Multiplier, s.BricksLeft, s.BallCount)
	}
	centered(c, h-2, r.truncate(status, w-4), "mutedText")
	if h >= 30 {
		powers := make([]string, 0, 7)
		add := func(name string, left float64) {
			if left > 0 {
				powers = append(powers, fmt.Sprintf("%s %.0fs", name, left))
			}
		}
		add("WIDE", s.WideLeft)
		add("SLOW", s.SlowLeft)
		add("FIRE", s.FireLeft)
		add("STICKY", s.StickyLeft)
		add("MAGNET", s.MagnetLeft)
		add("DHH", s.DHHLeft)
		if s.ShieldCharges > 0 {
			powers = append(powers, "SHIELD 1")
		}
		if len(powers) > 0 {
			centered(c, h-3, r.truncate(strings.Join(powers, "  "), w-4), "cyan")
		}
	}
	sound := "ON"
	if r.muted {
		sound = "OFF"
	}
	footer := "<- -> / A D MOVE   SPACE LAUNCH   P PAUSE   F10 SOUND " + sound + "   ESC QUIT"
	if !r.opts.ASCII {
		footer = "← → / A D MOVE   SPACE LAUNCH   P PAUSE   F10 SOUND " + sound + "   ESC QUIT"
	}
	if w < 60 {
		footer = "A/D MOVE  SPACE GO  P PAUSE  ESC QUIT"
	}
	centered(c, h-1, r.truncate(footer, w-2), "mutedText")
	switch s.Mode {
	case game.ModeAttract:
		lines := []string{"A WINTERMUTE ARCADE", "BREAKOUT", r.clean("25 LEVELS · 3 LIVES"), "PRESS ENTER TO SHIP"}
		if h >= 30 && len(s.HighScores) > 0 {
			lines = append(lines, "HIGH SCORES")
			for i, score := range s.HighScores {
				lines = append(lines, fmt.Sprintf("%d  %08d  LV %02d", i+1, score.Score, score.Level))
			}
		}
		r.overlay(c, lines)
	case game.ModeGameOver:
		r.overlay(c, []string{"GAME OVER", fmt.Sprintf("SCORE %08d", s.Score), "PRESS ENTER TO REWORK"})
	case game.ModeVictory:
		r.overlay(c, []string{"BREAKOUT SHIPPED", "RUNNING IN PRODUCTION", fmt.Sprintf("SCORE %08d", s.Score), "PRESS ENTER TO REWORK"})
	}
	if s.Paused {
		r.overlay(c, []string{"PAUSED"})
	}
}

func (r *Renderer) drawImpacts(c [][]halfCell, t transform, s game.Snapshot) {
	shown := 0
	for _, event := range s.Events {
		if shown >= 8 {
			break
		}
		if event.Type != "brickBroken" && event.Type != "brickDamaged" && event.Type != "steel" {
			continue
		}
		id, ok := event.Data["id"].(string)
		if !ok {
			continue
		}
		for _, b := range s.Bricks {
			if b.ID == id {
				x, y := t.x(b.X+b.W/2), t.hy(b.Y+b.H/2)/2
				if y >= 0 && y < len(c) && x > 0 && x < len(c[0])-1 {
					c[y][x].r = '*'
					c[y][x].fg = "yellow"
					shown++
				}
				break
			}
		}
	}
}

func (r *Renderer) drawBoss(c [][]halfCell, t transform, b game.BrickView) {
	x0, x1, y0, y1 := t.x(b.X), t.x(b.X+b.W), t.hy(b.Y), t.hy(b.Y+b.H)
	role := b.Palette
	kind := "boss-" + b.BossKind
	switch b.BossKind {
	case "creep":
		fillSemantic(c, x0, y0, x1, y1, role, kind)
		for hy := y0; hy <= y1; hy++ {
			for x := x0; x <= x1; x++ {
				corner := (hy-y0) < maxInt(1, (y1-y0)/4) && (x-x0 < 1 || x1-x < 1)
				tooth := hy == y1 && (x-x0)%3 == 1
				if corner || tooth {
					setHalfSemantic(c, x, hy, "playfield", "playfield")
				}
			}
		}
	case "reload":
		for hy := y0; hy <= y1; hy++ {
			for x := x0; x <= x1; x++ {
				edge := x == x0 || x == x1 || hy == y0 || hy == y1
				bar := hy == (y0+y1)/2 || x == (x0+x1)/2
				if edge || bar {
					setHalfSemantic(c, x, hy, role, kind)
				}
			}
		}
	case "maze":
		for hy := y0; hy <= y1; hy++ {
			for x := x0; x <= x1; x++ {
				if (x-x0)%3 != 1 || (hy-y0)%4 != 2 {
					setHalfSemantic(c, x, hy, role, kind)
				}
			}
		}
	case "monolith":
		inset := maxInt(1, (x1-x0)/5)
		for hy := y0; hy <= y1; hy++ {
			cut := 0
			if hy-y0 < 2 {
				cut = 1
			}
			for x := x0 + inset + cut; x <= x1-inset-cut; x++ {
				setHalfSemantic(c, x, hy, role, kind)
			}
		}
	case "apple":
		roles := []string{"red", "yellow", "green", "cyan", "blue", "magenta"}
		cx, cy := float64(x0+x1)/2, float64(y0+y1)/2
		rx, ry := math.Max(1, float64(x1-x0)/2), math.Max(1, float64(y1-y0)/2)
		for hy := y0; hy <= y1; hy++ {
			for x := x0; x <= x1; x++ {
				nx, ny := (float64(x)-cx)/rx, (float64(hy)-cy)/ry
				inside := nx*nx+ny*ny <= 1.05 && !(nx > .55 && ny > -.55 && ny < .15)
				if inside {
					stripe := clampIndex(int(float64(hy-y0)/math.Max(1, float64(y1-y0+1))*6), 0, 5)
					setHalfSemantic(c, x, hy, roles[stripe], kind)
				}
			}
		}
		leafY := y0
		for x := x0 + (x1-x0)/2; x <= x0+(x1-x0)*3/4; x++ {
			setHalfSemantic(c, x, leafY, "green", kind)
		}
	default:
		fillSemantic(c, x0, y0, x1, y1, role, kind)
	}
	if r.opts.ASCII {
		glyph := map[string]rune{"creep": 'C', "reload": 'R', "maze": 'M', "monolith": 'I', "apple": '@'}[b.BossKind]
		for y := maxInt(0, y0/2); y <= minInt(len(c)-1, y1/2); y++ {
			for x := maxInt(1, x0); x <= minInt(len(c[y])-2, x1); x++ {
				if c[y][x].topKind == kind || c[y][x].bottomKind == kind {
					c[y][x].r = glyph
					c[y][x].fg = role
				}
			}
		}
	}
	y := (y0 + y1) / 4
	face := "[><]"
	switch b.BossFace {
	case "ouch":
		face = "[XX]"
	case "shoot":
		face = "[OO]"
	case "grin":
		face = "[^v]"
	case "blink":
		face = "[--]"
	case "lookLeft":
		face = "[<o]"
	case "lookRight":
		face = "[o>]"
	}
	writeText(c, (x0+x1-lipgloss.Width(face))/2, y, face, "foreground")
	health := fmt.Sprintf("%s %d/%d", r.clean(b.BossName), b.HP, b.MaxHP)
	centered(c, minInt(y+1, len(c)-3), r.truncate(health, len(c[0])-4), "foreground")
}

func (r *Renderer) clean(s string) string {
	if !r.opts.ASCII {
		return s
	}
	replacer := strings.NewReplacer("—", "-", "←", "<", "→", ">", "↔", "<->", "×", "x", "▰", "#", "●", "o", "·", "-", "…", "...")
	return replacer.Replace(s)
}

func (r *Renderer) truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	suffix := "…"
	if r.opts.ASCII {
		suffix = "..."
	}
	suffixWidth := lipgloss.Width(suffix)
	if n <= suffixWidth {
		suffix = ""
		suffixWidth = 0
	}
	var out strings.Builder
	width := 0
	for _, ch := range s {
		glyph := string(ch)
		gw := lipgloss.Width(glyph)
		if gw != 1 {
			glyph = "?"
			gw = 1
		}
		if width+gw > n-suffixWidth {
			break
		}
		out.WriteString(glyph)
		width += gw
	}
	return out.String() + suffix
}

func (r *Renderer) overlay(c [][]halfCell, lines []string) {
	h, w := len(c), len(c[0])
	y0 := h/2 - len(lines)/2 - 1
	y1 := y0 + len(lines) + 1
	maxW := 0
	for _, line := range lines {
		if lipgloss.Width(line) > maxW {
			maxW = lipgloss.Width(line)
		}
	}
	maxW = minInt(maxW+4, w-4)
	x0 := (w - maxW) / 2
	for y := y0; y <= y1; y++ {
		for x := x0; x < x0+maxW; x++ {
			if y >= 0 && y < h && x >= 0 && x < w {
				c[y][x].top = "overlay"
				c[y][x].bottom = "overlay"
				c[y][x].topKind = "overlay"
				c[y][x].bottomKind = "overlay"
				c[y][x].r = 0
			}
		}
	}
	for i, line := range lines {
		centered(c, y0+1+i, r.truncate(r.clean(line), maxW-2), map[bool]string{true: "accent", false: "foreground"}[i == 1])
	}
}

func (r *Renderer) compose(c [][]halfCell) string {
	var out strings.Builder
	for y, row := range c {
		lastKey := ""
		var run strings.Builder
		flush := func() {
			if run.Len() > 0 {
				parts := strings.SplitN(lastKey, "/", 2)
				out.WriteString(r.style(parts[0], parts[1]).Render(run.String()))
				run.Reset()
			}
		}
		for _, cell := range row {
			ch := cell.r
			fg, bg := cell.fg, ""
			if ch == 0 {
				if r.opts.ASCII {
					if cell.top == "playfield" || cell.bottom == "playfield" {
						ch = '.'
						fg = "muted"
					} else {
						ch = ' '
						fg = "foreground"
					}
				} else if r.opts.Profile == Monochrome {
					ch = monoGlyph(cell)
					fg = "foreground"
				} else if cell.top == cell.bottom {
					ch = ' '
					bg = cell.top
				} else {
					ch = '▀'
					fg, bg = cell.top, cell.bottom
				}
			} else {
				if fg == "" {
					fg = "foreground"
				}
				bg = cell.bottom
			}
			key := fg + "/" + bg
			if key != lastKey {
				flush()
				lastKey = key
			}
			run.WriteRune(ch)
		}
		flush()
		if y < len(c)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func monoGlyph(cell halfCell) rune {
	semantic := func(kind string) rune {
		switch kind {
		case "normal":
			return '█'
		case "reinforced":
			return '▓'
		case "damaged":
			return '░'
		case "steel":
			return '#'
		case "boss-creep":
			return 'C'
		case "boss-reload":
			return 'R'
		case "boss-maze":
			return 'M'
		case "boss-monolith":
			return 'I'
		case "boss-apple":
			return '@'
		case "overlay", "background", "playfield":
			return ' '
		}
		return ' '
	}
	if cell.topKind == cell.bottomKind {
		return semantic(cell.topKind)
	}
	top, bottom := semantic(cell.topKind), semantic(cell.bottomKind)
	if top != ' ' && bottom == ' ' {
		if cell.topKind == "normal" {
			return '▀'
		}
		return top
	}
	if bottom != ' ' && top == ' ' {
		if cell.bottomKind == "normal" {
			return '▄'
		}
		return bottom
	}
	if top != ' ' {
		return top
	}
	return bottom
}

// Render composes one centered, bounded frame. Sizes below 40x20 return a clear requirement.
func (r *Renderer) Render(s game.Snapshot, width, height int) string {
	if width < 40 || height < 20 {
		w, h := maxInt(1, width), maxInt(1, height)
		lines := []string{r.truncate("need at least 40x20", w), r.truncate(fmt.Sprintf("current %dx%d", width, height), w)}
		if h == 1 {
			lines = lines[:1]
		}
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, strings.Join(lines, "\n"))
	}
	w, h := dimensions(width, height)
	key := cacheKey{w, h, s.Level, brickHash(s), r.opts.ASCII}
	if !r.cacheValid || key != r.cacheKey {
		r.static = r.buildStatic(s, w, h)
		r.cacheKey = key
		r.cacheValid = true
	}
	c := clone(r.static)
	r.overlayDynamic(c, s)
	frame := r.compose(c)
	if width > w || height > h {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, frame)
	}
	return frame
}

func clampIndex(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
