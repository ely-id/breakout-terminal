package game

import (
	"math"
	"sort"
	"strings"
)

const (
	fixedStepSeconds = 1.0 / 240.0
	startPaddleW     = 150.0
	minPaddleW       = 98.0
	maxMultiball     = 9
)

var konamiCode = []string{"up", "up", "down", "down", "left", "right", "left", "right", "b", "a", "enter"}

type inputState struct{ left, right, aimLeft, aimRight bool }

// Game owns the complete deterministic campaign simulation. It contains no
// terminal or process lifecycle state; callers supply elapsed time and intent.
type Game struct {
	time, accumulator float64
	mode              Mode
	paused            bool
	levelIndex        int
	level             Level
	bricksLeft        int
	score, lives      int
	combo             int
	comboLeft         float64
	paddle            Paddle
	balls             []Ball
	input             inputState
	drops             []Drop
	bossShots         []BossShot
	bossShotCooldown  float64
	powerRainCount    int
	powerRainCooldown float64
	message           string
	messageLeft       float64
	levelDelay        float64
	lifeDelay         float64
	runFinished       bool
	wideLeft          float64
	slowLeft          float64
	fireLeft          float64
	stickyLeft        float64
	magnetLeft        float64
	shieldCharges     int
	stickyAim         float64
	dhhLeft           float64
	secretBuffer      string
	codeBuffer        []string
	secrets           Secrets
	highScores        []HighScore
	lastScore         int
	events            []Event
}

// New returns a game on its level-one attract screen.
func New() *Game {
	g := &Game{mode: ModeAttract, level: CreateLevel(0), lives: 3, combo: 1,
		paddle: Paddle{X: 400, Y: PaddleY, W: startPaddleW, H: 18}, message: "PRESS ENTER"}
	g.bricksLeft = g.level.Breakable
	return g
}

func (g *Game) emit(kind string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	g.events = append(g.events, Event{kind, data, math.Round(g.time*1000) / 1000})
}

func (g *Game) setMessage(text string, seconds float64) {
	g.message, g.messageLeft = text, seconds
	g.emit("message", map[string]any{"text": text})
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func (g *Game) multiplier() int {
	extra := 0
	if g.dhhLeft > 0 {
		extra = 2
	}
	return minInt(9, g.combo+extra)
}

func (g *Game) addScore(base int, source string) int {
	points := base * g.multiplier()
	g.score += points
	g.emit("scored", map[string]any{"points": points, "base": base, "multiplier": g.multiplier(), "source": source})
	return points
}

func (g *Game) levelPaddleWidth() float64 {
	return math.Max(minPaddleW, startPaddleW-float64(g.levelIndex)*2.25)
}

func makeBall(x, y, vx, vy float64) Ball {
	return Ball{X: x, Y: y, VX: vx, VY: vy, R: BallRadius, pierce: make(map[string]bool)}
}

func (g *Game) attachBall() {
	g.balls = []Ball{makeBall(g.paddle.X, g.paddle.Y-g.paddle.H/2-BallRadius-2, 0, 0)}
}

func (g *Game) loadLevel(index int, announce bool) {
	g.levelIndex = clampInt(index, 0, LevelCount()-1)
	g.level = CreateLevel(g.levelIndex)
	g.bricksLeft = g.level.Breakable
	g.drops, g.bossShots = nil, nil
	if g.level.BossIndex >= 0 {
		g.bossShotCooldown = 1.25
	} else {
		g.bossShotCooldown = 0
	}
	g.powerRainCount = 0
	if g.level.PowerRain != nil {
		g.powerRainCooldown = g.level.PowerRain.Delay
	} else {
		g.powerRainCooldown = 0
	}
	g.paddle.X, g.paddle.VX = WorldWidth/2, 0
	g.wideLeft, g.slowLeft, g.fireLeft, g.stickyLeft, g.magnetLeft = 0, 0, 0, 0, 0
	g.shieldCharges, g.stickyAim = 0, 0
	g.paddle.W = g.levelPaddleWidth()
	g.attachBall()
	g.mode = ModeServe
	if announce {
		prefix := "LEVEL "
		if g.level.BossIndex >= 0 {
			prefix = "BOSS LEVEL "
		}
		g.setMessage(prefix+itoa(g.levelIndex+1)+" — "+g.level.Name, 2.4)
		g.emit("levelStarted", map[string]any{"level": g.levelIndex + 1, "name": g.level.Name})
		if boss := g.activeBoss(); boss != nil {
			g.emit("bossStarted", map[string]any{"level": g.levelIndex + 1, "id": boss.ID, "name": boss.BossName, "hp": boss.MaxHP, "stages": boss.BossStages})
		}
	}
}

func (g *Game) resetRun() {
	g.score, g.lives, g.combo = 0, 3, 1
	g.comboLeft, g.dhhLeft = 0, 0
	g.paused, g.runFinished = false, false
	g.secretBuffer, g.codeBuffer = "", nil
	g.secrets, g.input = Secrets{}, inputState{}
	g.levelDelay, g.lifeDelay = 0, 0
	g.bossShots, g.drops = nil, nil
	g.bossShotCooldown = 0
}

// StartLevel starts a fresh run at a 1-based campaign level.
func (g *Game) StartLevel(levelNumber int) bool {
	if levelNumber < 1 || levelNumber > LevelCount() {
		return false
	}
	g.resetRun()
	g.loadLevel(levelNumber-1, true)
	g.emit("newGame", map[string]any{"startingLevel": levelNumber})
	return true
}

func (g *Game) newGame() { g.StartLevel(1) }

// Launch releases a serve or sticky-held balls.
func (g *Game) Launch() bool {
	if g.releaseStickyBalls() {
		return true
	}
	if g.mode != ModeServe || len(g.balls) == 0 {
		return false
	}
	direction := 1.0
	if (g.levelIndex+g.lives)%2 == 0 {
		direction = -1
	}
	g.balls[0].VX, g.balls[0].VY = direction*g.level.Speed*.34, -g.level.Speed*.94
	g.mode = ModePlaying
	if g.level.BossIndex >= 0 && g.bossShotCooldown <= 0 {
		g.bossShotCooldown = 1
	}
	g.setMessage("BREAK OUT", 1)
	g.emit("launch", map[string]any{"level": g.levelIndex + 1, "speed": g.level.Speed})
	return true
}

func (g *Game) heldBallCount() int {
	n := 0
	for i := range g.balls {
		if g.balls[i].Stuck {
			n++
		}
	}
	return n
}

func (g *Game) releaseStickyBalls() bool {
	held := g.heldBallCount()
	if held == 0 {
		return false
	}
	speed, released := g.targetBallSpeed(), 0
	for i := range g.balls {
		b := &g.balls[i]
		if !b.Stuck {
			continue
		}
		spread := 0.0
		if held > 1 {
			spread = (float64(released) - float64(held-1)/2) * .11
		}
		angle := clamp(g.stickyAim+spread, -1.12, 1.12)
		b.Stuck, b.VX, b.VY = false, math.Sin(angle)*speed, -math.Abs(math.Cos(angle)*speed)
		b.pierce = make(map[string]bool)
		released++
	}
	g.emit("stickyLaunch", map[string]any{"balls": released, "aim": g.stickyAim})
	return true
}

// KeyDown applies a gameplay action. One-shot filtering belongs to the adapter.
func (g *Game) KeyDown(action Action) bool {
	switch action {
	case MoveLeft:
		g.input.left = true
	case MoveRight:
		g.input.right = true
	case AimLeft:
		g.input.aimLeft = true
	case AimRight:
		g.input.aimRight = true
	case Launch:
		if g.mode == ModeAttract || g.mode == ModeGameOver || g.mode == ModeVictory {
			g.newGame()
		} else {
			g.Launch()
		}
	case NewGame:
		if g.mode == ModeAttract || g.mode == ModeGameOver || g.mode == ModeVictory {
			g.newGame()
		}
	case Pause:
		return g.TogglePause()
	default:
		return false
	}
	return true
}

func (g *Game) KeyUp(action Action) bool {
	switch action {
	case MoveLeft:
		g.input.left = false
	case MoveRight:
		g.input.right = false
	case AimLeft:
		g.input.aimLeft = false
	case AimRight:
		g.input.aimRight = false
	default:
		return false
	}
	return true
}

func (g *Game) TogglePause() bool {
	if g.mode == ModeAttract || g.mode == ModeGameOver || g.mode == ModeVictory {
		return false
	}
	g.paused = !g.paused
	g.emit("pause", map[string]any{"paused": g.paused})
	return true
}

// InputCharacter updates the rolling typed-secret buffer.
func (g *Game) InputCharacter(ch string) bool {
	var clean strings.Builder
	for _, r := range strings.ToUpper(ch) {
		if r >= 'A' && r <= 'Z' {
			clean.WriteRune(r)
		}
	}
	if clean.Len() == 0 {
		return false
	}
	g.secretBuffer += clean.String()
	if len(g.secretBuffer) > 12 {
		g.secretBuffer = g.secretBuffer[len(g.secretBuffer)-12:]
	}
	if strings.HasSuffix(g.secretBuffer, "BASECAMP") {
		return g.triggerSecret("BASECAMP")
	}
	if strings.HasSuffix(g.secretBuffer, "DHH") {
		return g.triggerSecret("DHH")
	}
	return false
}

func (g *Game) triggerSecret(name string) bool {
	switch name {
	case "DHH":
		g.dhhLeft, g.secrets.DHH = 37, true
		g.setMessage("DHH MODE — RUNNING IN PRODUCTION", 3.7)
		g.emit("secret", map[string]any{"name": "DHH", "seconds": 37})
	case "BASECAMP":
		g.secrets.Basecamp, g.wideLeft = true, 37
		g.paddle.W = math.Max(222, g.levelPaddleWidth()+72)
		g.setMessage("BASECAMP MODE — CLARITY WINS", 3.7)
		g.emit("secret", map[string]any{"name": "BASECAMP", "seconds": 37})
	default:
		return false
	}
	return true
}

// InputCodeToken updates the exact Konami token buffer.
func (g *Game) InputCodeToken(token string) bool {
	token = strings.ToLower(token)
	valid := false
	for _, want := range konamiCode {
		if token == want {
			valid = true
			break
		}
	}
	if !valid {
		return false
	}
	g.codeBuffer = append(g.codeBuffer, token)
	if len(g.codeBuffer) > len(konamiCode) {
		g.codeBuffer = g.codeBuffer[1:]
	}
	if len(g.codeBuffer) != len(konamiCode) {
		return false
	}
	for i := range konamiCode {
		if g.codeBuffer[i] != konamiCode[i] {
			return false
		}
	}
	g.codeBuffer = nil
	return g.spawnBallStorm(100)
}

func (g *Game) spawnBallStorm(count int) bool {
	count = clampInt(count, 1, 100)
	if g.mode == ModeAttract || g.mode == ModeGameOver || g.mode == ModeVictory {
		g.newGame()
	}
	if g.mode == ModeLifeLost {
		g.startNextLife()
	}
	if g.mode == ModeLevelClear {
		return false
	}
	ox, oy := g.paddle.X, g.paddle.Y-30
	if len(g.balls) > 0 {
		ox, oy = g.balls[0].X, math.Min(g.balls[0].Y, g.paddle.Y-30)
	}
	speed := g.level.Speed * 1.06
	g.balls = make([]Ball, 0, count)
	for i := 0; i < count; i++ {
		ratio := .5
		if count > 1 {
			ratio = float64(i) / float64(count-1)
		}
		angle := -math.Pi*.86 + ratio*math.Pi*.72
		stagger := float64(i%9-4) * 1.8
		g.balls = append(g.balls, makeBall(clamp(ox+stagger, 30, WorldWidth-30), oy-float64(i%5)*2, math.Cos(angle)*speed, math.Sin(angle)*speed))
	}
	g.paused, g.mode, g.secrets.Konami = false, ModePlaying, true
	g.setMessage("KONAMI MODE — 100 BALLS", 3.7)
	g.emit("konami", map[string]any{"balls": count})
	return true
}

// Advance advances the game by elapsed seconds with the reference 50 ms
// submission clamp and exact 240 Hz logical integration. Paused time is discarded.
func (g *Game) Advance(seconds float64) {
	if seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) || g.paused {
		return
	}
	g.accumulator += math.Min(seconds, .05)
	for g.accumulator+1e-15 >= fixedStepSeconds {
		g.tick(fixedStepSeconds)
		g.accumulator -= fixedStepSeconds
	}
}

func (g *Game) tick(dt float64) {
	g.time += dt
	if g.messageLeft > 0 && g.messageLeft < 9000 {
		g.messageLeft -= dt
		if g.messageLeft <= 0 {
			g.message = ""
		}
	}
	if g.comboLeft > 0 {
		g.comboLeft -= dt
		if g.comboLeft <= 0 {
			g.combo = 1
		}
	}
	if g.wideLeft > 0 {
		g.wideLeft -= dt
		if g.wideLeft <= 0 {
			g.paddle.W = g.levelPaddleWidth()
		}
	}
	if g.slowLeft > 0 {
		g.slowLeft -= dt
	}
	if g.fireLeft > 0 {
		g.fireLeft -= dt
	}
	if g.stickyLeft > 0 {
		g.stickyLeft -= dt
		if g.stickyLeft <= 0 {
			g.releaseStickyBalls()
		}
	}
	if g.magnetLeft > 0 {
		g.magnetLeft -= dt
	}
	if g.dhhLeft > 0 {
		g.dhhLeft -= dt
	}
	if boss := g.activeBoss(); boss != nil {
		if boss.BossOuchLeft > 0 {
			boss.BossOuchLeft = math.Max(0, boss.BossOuchLeft-dt)
		}
		if boss.BossShootLeft > 0 {
			boss.BossShootLeft = math.Max(0, boss.BossShootLeft-dt)
		}
	}
	g.updatePaddle(dt)
	switch g.mode {
	case ModePlaying:
		g.updateBossMovement(dt)
		g.updateBalls(dt)
		g.updateDrops(dt)
		g.updatePowerRain(dt)
		g.updateBossAttack(dt)
	case ModeLevelClear:
		g.levelDelay -= dt
		if g.levelDelay <= 1e-12 {
			g.loadLevel(g.levelIndex+1, true)
		}
	case ModeLifeLost:
		g.lifeDelay -= dt
		if g.lifeDelay <= 1e-12 {
			g.startNextLife()
		}
	}
}

func (g *Game) updatePaddle(dt float64) {
	direction := 0
	if g.input.right {
		direction++
	}
	if g.input.left {
		direction--
	}
	g.paddle.VX = float64(direction) * 660
	g.paddle.X = clamp(g.paddle.X+g.paddle.VX*dt, g.paddle.W/2+20, WorldWidth-g.paddle.W/2-20)
	if g.heldBallCount() > 0 {
		aim := 0
		if g.input.aimRight {
			aim++
		}
		if g.input.aimLeft {
			aim--
		}
		g.stickyAim = clamp(g.stickyAim+float64(aim)*dt*1.9, -1.05, 1.05)
		for i := range g.balls {
			b := &g.balls[i]
			if !b.Stuck {
				continue
			}
			b.StickOffset = clamp(b.StickOffset, -g.paddle.W/2+BallRadius, g.paddle.W/2-BallRadius)
			b.X, b.Y = g.paddle.X+b.StickOffset, g.paddle.Y-g.paddle.H/2-b.R-2
		}
	}
	if g.mode == ModeServe && len(g.balls) > 0 {
		g.balls[0].X, g.balls[0].Y = g.paddle.X, g.paddle.Y-g.paddle.H/2-BallRadius-2
	}
}

func circleHitsRect(x, y, r float64, rect *Brick) bool {
	cx, cy := clamp(x, rect.X, rect.X+rect.W), clamp(y, rect.Y, rect.Y+rect.H)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

func circleHitsPaddle(x, y, r float64, p Paddle) bool {
	cx, cy := clamp(x, p.X-p.W/2, p.X+p.W/2), clamp(y, p.Y-p.H/2, p.Y+p.H/2)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

func bounceFromRect(b *Ball, rect *Brick, px, py float64) {
	if py+b.R <= rect.Y && b.VY > 0 {
		b.Y, b.VY = rect.Y-b.R-.1, -math.Abs(b.VY)
	} else if py-b.R >= rect.Y+rect.H && b.VY < 0 {
		b.Y, b.VY = rect.Y+rect.H+b.R+.1, math.Abs(b.VY)
	} else if px+b.R <= rect.X && b.VX > 0 {
		b.X, b.VX = rect.X-b.R-.1, -math.Abs(b.VX)
	} else if px-b.R >= rect.X+rect.W && b.VX < 0 {
		b.X, b.VX = rect.X+rect.W+b.R+.1, math.Abs(b.VX)
	} else {
		left, right := math.Abs((b.X+b.R)-rect.X), math.Abs((rect.X+rect.W)-(b.X-b.R))
		top, bottom := math.Abs((b.Y+b.R)-rect.Y), math.Abs((rect.Y+rect.H)-(b.Y-b.R))
		minPen := math.Min(math.Min(left, right), math.Min(top, bottom))
		if minPen == left || minPen == right {
			b.VX = -b.VX
		} else {
			b.VY = -b.VY
		}
	}
}

func (g *Game) targetBallSpeed() float64 {
	cleared := 0.0
	if g.level.Breakable > 0 {
		cleared = 1 - float64(g.bricksLeft)/float64(g.level.Breakable)
	}
	target := g.level.Speed * (1 + clamp(cleared, 0, 1)*.22)
	if g.slowLeft > 0 {
		target *= .72
	}
	return target
}

func (g *Game) normalizeBallSpeed(b *Ball) {
	speed := math.Hypot(b.VX, b.VY)
	if speed < 1 {
		return
	}
	target := g.targetBallSpeed()
	desired := clamp(speed, target*.88, target*1.22)
	b.VX, b.VY = b.VX*desired/speed, b.VY*desired/speed
	if math.Abs(b.VY) < desired*.28 {
		signY, signX := 1.0, 1.0
		if b.VY < 0 {
			signY = -1
		}
		if b.VX < 0 {
			signX = -1
		}
		b.VY = signY * desired * .28
		b.VX = signX * math.Sqrt(math.Max(0, desired*desired-b.VY*b.VY))
	}
}

func (g *Game) updateBall(b *Ball, dt float64) bool {
	if b.Stuck {
		return true
	}
	px, py := b.X, b.Y
	b.X, b.Y = b.X+b.VX*dt, b.Y+b.VY*dt
	if b.X-b.R < 20 {
		b.X, b.VX = 20+b.R, math.Abs(b.VX)
		g.emit("wall", map[string]any{"side": "left"})
	}
	if b.X+b.R > WorldWidth-20 {
		b.X, b.VX = WorldWidth-20-b.R, -math.Abs(b.VX)
		g.emit("wall", map[string]any{"side": "right"})
	}
	if b.Y-b.R < 92 {
		b.Y, b.VY = 92+b.R, math.Abs(b.VY)
		g.emit("wall", map[string]any{"side": "top"})
	}
	if b.VY > 0 && circleHitsPaddle(b.X, b.Y, b.R, g.paddle) {
		b.Y = g.paddle.Y - g.paddle.H/2 - b.R - .1
		offset := clamp((b.X-g.paddle.X)/(g.paddle.W/2), -1, 1)
		if g.stickyLeft > 0 {
			b.Stuck, b.StickOffset, b.VX, b.VY = true, clamp(b.X-g.paddle.X, -g.paddle.W/2+b.R, g.paddle.W/2-b.R), 0, 0
			b.pierce = make(map[string]bool)
			g.emit("stickyCatch", map[string]any{"offset": offset})
			return true
		}
		speed := math.Max(g.targetBallSpeed(), math.Hypot(b.VX, b.VY))
		angle := offset * 1.05
		b.VX, b.VY = math.Sin(angle)*speed+g.paddle.VX*.13, -math.Abs(math.Cos(angle)*speed)
		g.combo, g.comboLeft = 1, 0
		g.emit("paddle", map[string]any{"offset": offset})
	}
	piercing := g.fireLeft > 0
	touching := make(map[string]bool)
	for i := range g.level.Bricks {
		brick := &g.level.Bricks[i]
		if brick.HP == 0 || !circleHitsRect(b.X, b.Y, b.R, brick) {
			continue
		}
		if piercing && !brick.Steel && brick.HP > 0 {
			touching[brick.ID] = true
			if !b.pierce[brick.ID] {
				g.hitBrick(brick)
			}
			continue
		}
		bounceFromRect(b, brick, px, py)
		g.hitBrick(brick)
		break
	}
	if piercing {
		b.pierce = touching
	} else {
		b.pierce = make(map[string]bool)
	}
	g.normalizeBallSpeed(b)
	b.Trail = append(b.Trail, Point{b.X, b.Y})
	if len(b.Trail) > 6 {
		b.Trail = b.Trail[len(b.Trail)-6:]
	}
	if g.shieldCharges > 0 && b.VY > 0 && py+b.R < ShieldY && b.Y+b.R >= ShieldY {
		g.shieldCharges--
		b.Y, b.VY = ShieldY-b.R-.1, -math.Abs(b.VY)
		g.normalizeBallSpeed(b)
		g.emit("shieldSaved", map[string]any{"x": b.X, "y": ShieldY, "charges": g.shieldCharges})
		g.setMessage("SHIELD SAVED THE RUN", 1.4)
	}
	return b.Y-b.R <= WorldHeight
}

func (g *Game) updateBalls(dt float64) {
	for i := len(g.balls) - 1; i >= 0; i-- {
		if g.mode != ModePlaying {
			return
		}
		if !g.updateBall(&g.balls[i], dt) {
			g.balls = append(g.balls[:i], g.balls[i+1:]...)
		}
	}
	if g.mode == ModePlaying && len(g.balls) == 0 {
		g.loseBall()
	}
}

func (g *Game) hitBrick(brick *Brick) {
	if brick == nil || brick.HP == 0 {
		return
	}
	if brick.Steel || brick.HP < 0 {
		g.emit("steel", map[string]any{"id": brick.ID})
		return
	}
	previousStage := brick.BossStage
	brick.HP--
	if brick.HP > 0 {
		base, source := 25, "reinforced"
		if brick.Boss {
			base, source = 90+(g.levelIndex+1)*8, "bossDamage"
		}
		g.addScore(base, source)
		if brick.Boss {
			brick.BossOuchLeft = .72
			if stage := bossStageFor(brick); stage != previousStage {
				g.enterBossStage(brick, stage)
			}
		}
		g.emit("brickDamaged", map[string]any{"id": brick.ID, "hp": brick.HP, "maxHp": brick.MaxHP, "boss": brick.Boss, "bossStage": brick.BossStage})
		return
	}
	g.bricksLeft--
	g.combo, g.comboLeft = minInt(7, g.combo+1), 2.5
	base, source := 100+g.levelIndex*12, "brick"
	if brick.Special == "37" {
		base += 270
	}
	if brick.Boss {
		base, source = 900+(g.levelIndex+1)*45, "boss"
	}
	g.addScore(base, source)
	g.emit("brickBroken", map[string]any{"id": brick.ID, "special": brick.Special, "power": brick.Power, "boss": brick.Boss, "bossName": brick.BossName})
	if brick.Boss {
		g.bossShots, g.bossShotCooldown = nil, 0
		g.emit("bossDefeated", map[string]any{"id": brick.ID, "name": brick.BossName, "level": g.levelIndex + 1})
		g.setMessage("BOSS DOWN — "+brick.BossName, 2)
	}
	g.specialBrickMessage(brick)
	if brick.Power != "" {
		g.drops = append(g.drops, Drop{brick.X + brick.W/2, brick.Y + brick.H/2, 0, 155, brick.Power})
	}
	if g.bricksLeft <= 0 {
		g.completeLevel()
	}
}

func (g *Game) specialBrickMessage(b *Brick) {
	switch b.Special {
	case "DHH":
		g.setMessage("DHH: SHIP IT", 2.2)
	case "RYAN":
		g.setMessage("RYAN: KEEP THE SHELL FAST", 2.2)
	case "OMARCHY":
		g.setMessage("BEAUTIFUL MOTIVATIONS", 2.2)
	case "37":
		g.secrets.Signal37 = true
		g.setMessage("SIGNAL RECEIVED: 37", 2.2)
	}
}

func (g *Game) completeLevel() {
	if g.mode != ModePlaying {
		return
	}
	g.addScore(1000+(g.levelIndex+1)*250, "levelClear")
	g.emit("levelCleared", map[string]any{"level": g.levelIndex + 1, "name": g.level.Name})
	if g.levelIndex >= LevelCount()-1 {
		g.finishRun(true)
		return
	}
	g.mode, g.levelDelay = ModeLevelClear, 1.5
	g.balls, g.drops = nil, nil
	g.setMessage("LEVEL CLEAR — "+g.level.Name, 1.5)
}

func (g *Game) loseBall() {
	if g.mode != ModePlaying {
		return
	}
	g.lives--
	g.combo, g.comboLeft = 1, 0
	g.drops, g.bossShots = nil, nil
	g.emit("lifeLost", map[string]any{"lives": g.lives})
	if g.lives <= 0 {
		g.finishRun(false)
		return
	}
	g.mode, g.lifeDelay, g.balls = ModeLifeLost, 1.1, nil
	g.setMessage("STAY IN THE FIGHT — "+itoa(g.lives)+" LEFT", 1.1)
}

func (g *Game) startNextLife() {
	g.attachBall()
	if g.level.BossIndex >= 0 {
		g.bossShotCooldown = 1
	} else {
		g.bossShotCooldown = 0
	}
	g.mode = ModeServe
	g.setMessage("SPACE TO SERVE", 1.5)
}

func (g *Game) finishRun(victory bool) {
	if g.runFinished {
		return
	}
	g.runFinished, g.lastScore = true, g.score
	g.highScores = append(g.highScores, HighScore{g.score, g.levelIndex + 1, victory})
	sort.SliceStable(g.highScores, func(i, j int) bool { return g.highScores[i].Score > g.highScores[j].Score })
	if len(g.highScores) > 10 {
		g.highScores = g.highScores[:10]
	}
	g.mode, g.balls = ModeGameOver, nil
	message, event := "GAME OVER", "gameOver"
	if victory {
		g.mode, message, event = ModeVictory, "OMARCHY SHIPPED", "victory"
	}
	g.setMessage(message, 9999)
	g.emit(event, map[string]any{"score": g.score, "level": g.levelIndex + 1})
	g.emit("highScoreUpdated", map[string]any{"score": g.score, "rank": g.highScoreRank(g.score)})
}

func (g *Game) highScoreRank(score int) int {
	for i, h := range g.highScores {
		if h.Score == score {
			return i + 1
		}
	}
	return 0
}

func (g *Game) activatePower(kind string) {
	switch kind {
	case "wide":
		g.wideLeft, g.paddle.W = 15, math.Max(190, g.levelPaddleWidth()+62)
		g.setMessage("WIDE MODE — LESS, BUT BIGGER", 2)
	case "slow":
		g.slowLeft = 12
		for i := range g.balls {
			g.balls[i].VX *= .72
			g.balls[i].VY *= .72
		}
		g.setMessage("TAKE IT SLOW", 2)
	case "life":
		g.lives = minInt(7, g.lives+1)
		g.setMessage("ONE MORE PERSON", 2)
	case "multi":
		g.splitMultiball()
		g.setMessage("THREE-BALL SHIPPING", 2)
	case "fire":
		g.fireLeft = 11
		g.setMessage("FIREBALL — PIERCE THE LAYERS", 2)
	case "sticky":
		g.stickyLeft, g.stickyAim = 16, 0
		g.setMessage("STICKY — CATCH, AIM, RELEASE", 2)
	case "shield":
		g.shieldCharges = 1
		g.setMessage("BOTTOM SHIELD — ONE SAVE", 2)
	case "magnet":
		g.magnetLeft = 14
		g.setMessage("MAGNET — CAPSULES COME TO YOU", 2)
	default:
		return
	}
	g.emit("powerUp", map[string]any{"kind": kind})
}

func (g *Game) splitMultiball() {
	if len(g.balls) == 0 || len(g.balls) >= maxMultiball {
		return
	}
	source := &g.balls[0]
	for i := range g.balls {
		if !g.balls[i].Stuck {
			source = &g.balls[i]
			break
		}
	}
	additions := minInt(2, maxMultiball-len(g.balls))
	sourceSpeed := math.Hypot(source.VX, source.VY)
	speed := math.Max(g.targetBallSpeed(), sourceSpeed)
	heading := -math.Pi / 2
	if sourceSpeed > 1 {
		heading = math.Atan2(source.VY, source.VX)
	}
	clones := make([]Ball, 0, additions)
	for n := 0; n < additions; n++ {
		sign := -1.0
		if n == 1 {
			sign = 1
		}
		angle := heading + sign*.34
		clone := makeBall(source.X+sign*4, source.Y-2, math.Cos(angle)*speed, math.Sin(angle)*speed)
		if source.Stuck {
			clone.Stuck, clone.StickOffset, clone.VX, clone.VY = true, clamp(source.StickOffset+sign*26, -g.paddle.W/2+BallRadius, g.paddle.W/2-BallRadius), 0, 0
		}
		clones = append(clones, clone)
	}
	g.balls = append(g.balls, clones...)
}

func (g *Game) updateDrops(dt float64) {
	for i := len(g.drops) - 1; i >= 0; i-- {
		d := &g.drops[i]
		if g.magnetLeft > 0 {
			desired := clamp((g.paddle.X-d.X)*3.2, -400, 400)
			d.VX += (desired - d.VX) * math.Min(1, dt*7)
		} else {
			d.VX *= math.Max(0, 1-dt*4)
		}
		d.X, d.Y = clamp(d.X+d.VX*dt, 34, WorldWidth-34), d.Y+d.VY*dt
		if d.Y >= g.paddle.Y-20 && d.Y <= g.paddle.Y+20 && math.Abs(d.X-g.paddle.X) <= g.paddle.W/2+12 {
			kind := d.Kind
			g.drops = append(g.drops[:i], g.drops[i+1:]...)
			g.activatePower(kind)
		} else if d.Y > WorldHeight+20 {
			g.drops = append(g.drops[:i], g.drops[i+1:]...)
		}
	}
}

func bossStageFor(b *Brick) int {
	if b.HP > b.MaxHP*2/3 {
		return 1
	}
	if b.HP > b.MaxHP/3 {
		return 2
	}
	return 3
}

func (g *Game) enterBossStage(b *Brick, stage int) {
	b.BossStage, g.bossShots, g.bossShotCooldown = stage, nil, .55
	g.emit("bossStage", map[string]any{"id": b.ID, "name": b.BossName, "stage": stage, "stages": b.BossStages, "hp": b.HP, "maxHp": b.MaxHP})
	g.setMessage("BOSS PHASE "+itoa(stage)+"/"+itoa(b.BossStages)+" — FIRE RATE UP", 2)
}

func (g *Game) activeBoss() *Brick {
	for i := range g.level.Bricks {
		if g.level.Bricks[i].Boss && g.level.Bricks[i].HP > 0 {
			return &g.level.Bricks[i]
		}
	}
	return nil
}

func (g *Game) reflectedBallX(b *Ball, seconds float64) float64 {
	minX, maxX := 20+b.R, WorldWidth-20-b.R
	span, period := maxX-minX, (maxX-minX)*2
	wrapped := math.Mod((b.X-minX)+b.VX*seconds, period)
	if wrapped < 0 {
		wrapped += period
	}
	if wrapped <= span {
		return minX + wrapped
	}
	return maxX - (wrapped - span)
}

func (g *Game) updateBossMovement(dt float64) {
	boss := g.activeBoss()
	if boss == nil {
		return
	}
	stage, centerX := boss.BossStage, boss.X+boss.W/2
	var threat *Ball
	threatX, threatTime := 0.0, math.Inf(1)
	for i := range g.balls {
		ball := &g.balls[i]
		if ball.Stuck || ball.VY >= -50 || ball.Y <= boss.Y+boss.H {
			continue
		}
		seconds := (ball.Y - (boss.Y + boss.H + ball.R)) / -ball.VY
		if seconds < 0 || seconds > 1.35 {
			continue
		}
		predicted := g.reflectedBallX(ball, seconds)
		margin := 54 + ball.R
		if predicted < boss.X-margin || predicted > boss.X+boss.W+margin || seconds >= threatTime {
			continue
		}
		threat, threatX, threatTime = ball, predicted, seconds
	}
	direction := boss.BossDirection
	desired, acceleration := 0.0, 380.0
	boss.BossAvoiding = threat != nil
	if threat != nil {
		boss.BossThreatX = threatX
		if math.Abs(threatX-centerX) < 8 {
			if threat.VX > 0 {
				direction = -1
			} else if threat.VX < 0 {
				direction = 1
			}
		} else if threatX < centerX {
			direction = 1
		} else {
			direction = -1
		}
		desired, acceleration = float64(direction)*(158+float64(stage)*20+float64(g.levelIndex)*1.5), 920
	} else {
		desired = float64(direction) * (52 + float64(stage)*9 + float64(g.levelIndex)*.8)
	}
	boss.BossDirection = direction
	boss.BossVX += clamp(desired-boss.BossVX, -acceleration*dt, acceleration*dt)
	boss.X += boss.BossVX * dt
	minX, maxX := 44.0, WorldWidth-44-boss.W
	if boss.X <= minX {
		boss.X, boss.BossDirection, boss.BossVX = minX, 1, math.Abs(boss.BossVX)
	} else if boss.X >= maxX {
		boss.X, boss.BossDirection, boss.BossVX = maxX, -1, -math.Abs(boss.BossVX)
	}
}

func (g *Game) bossVolleyCount(stage int) int {
	if g.levelIndex == 5 {
		return 1
	}
	return max(1, stage)
}
func bossFireInterval(stage int) float64 {
	if stage >= 3 {
		return 1.12
	}
	if stage == 2 {
		return 1.62
	}
	return 2.18
}

func (g *Game) fireBossShots(boss *Brick) int {
	if boss == nil || (g.levelIndex == 5 && len(g.bossShots) > 0) {
		return 0
	}
	stage, count := boss.BossStage, g.bossVolleyCount(boss.BossStage)
	if len(g.bossShots)+count > 12 {
		return 0
	}
	originY := boss.Y + boss.H - 8
	speedY := 170 + float64(stage)*18 + float64(g.levelIndex)*1.2
	flight := math.Max(.35, (g.paddle.Y-originY)/speedY)
	for i := 0; i < count; i++ {
		centered := float64(i) - float64(count-1)/2
		originX := boss.X + boss.W/2 + centered*46
		spacing := 64.0
		if stage == 2 {
			spacing = 76
		}
		targetX := g.paddle.X + centered*spacing
		g.bossShots = append(g.bossShots, BossShot{originX, originY, (targetX - originX) / flight, speedY, 6, stage})
	}
	boss.BossShootLeft = .34
	g.emit("bossShotFired", map[string]any{"id": boss.ID, "stage": stage, "count": count, "x": boss.X + boss.W/2, "y": originY})
	return count
}

func (g *Game) updateBossAttack(dt float64) {
	if g.mode != ModePlaying {
		return
	}
	boss := g.activeBoss()
	if boss == nil {
		g.bossShots, g.bossShotCooldown = nil, 0
		return
	}
	g.bossShotCooldown -= dt
	if g.bossShotCooldown <= 0 {
		if g.levelIndex == 5 && len(g.bossShots) > 0 {
			g.bossShotCooldown = 0
		} else if g.fireBossShots(boss) > 0 {
			g.bossShotCooldown += bossFireInterval(boss.BossStage)
		} else {
			g.bossShotCooldown = 0
		}
	}
	for i := len(g.bossShots) - 1; i >= 0; i-- {
		s := &g.bossShots[i]
		s.X, s.Y = s.X+s.VX*dt, s.Y+s.VY*dt
		if s.X-s.R < 20 {
			s.X, s.VX = 20+s.R, math.Abs(s.VX)
		}
		if s.X+s.R > WorldWidth-20 {
			s.X, s.VX = WorldWidth-20-s.R, -math.Abs(s.VX)
		}
		if circleHitsPaddle(s.X, s.Y, s.R, g.paddle) {
			g.emit("bossShotHit", map[string]any{"x": s.X, "y": s.Y, "stage": s.Stage})
			g.setMessage("BOSS HIT — LIFE LOST", 1.1)
			g.bossShots = nil
			g.loseBall()
			return
		}
		if s.Y-s.R > WorldHeight {
			g.bossShots = append(g.bossShots[:i], g.bossShots[i+1:]...)
		}
	}
}

func (g *Game) updatePowerRain(dt float64) {
	rain := g.level.PowerRain
	if rain == nil || g.mode != ModePlaying || g.powerRainCount >= rain.Max {
		return
	}
	g.powerRainCooldown -= dt
	if g.powerRainCooldown > 0 || len(g.drops) > 0 {
		return
	}
	slots := []float64{86, 168, 632, 714}
	kind := rain.Kinds[g.powerRainCount%len(rain.Kinds)]
	g.drops = append(g.drops, Drop{slots[g.powerRainCount%len(slots)], 118, 0, 140, kind})
	g.powerRainCount++
	g.powerRainCooldown = rain.Interval
	g.emit("powerRain", map[string]any{"kind": kind, "count": g.powerRainCount, "max": rain.Max})
}

func (g *Game) bossFace(b *Brick) string {
	if b.BossOuchLeft > 0 {
		return "ouch"
	}
	if b.BossShootLeft > 0 {
		return "shoot"
	}
	if b.BossAvoiding {
		if b.BossThreatX < b.X+b.W/2 {
			return "lookLeft"
		}
		return "lookRight"
	}
	if math.Mod(g.time, 2.6) > 2.48 {
		return "blink"
	}
	if b.BossStage >= 3 {
		return "grin"
	}
	return "angry"
}

// Snapshot returns a detached view and drains semantic events since the prior snapshot.
func (g *Game) Snapshot() Snapshot {
	events := g.events
	g.events = nil
	balls := make([]BallView, len(g.balls))
	for i, b := range g.balls {
		trail := append([]Point(nil), b.Trail...)
		if i >= 12 {
			trail = nil
		}
		balls[i] = BallView{b.X, b.Y, b.VX, b.VY, b.R, b.Stuck, trail}
	}
	bricks := make([]BrickView, len(g.level.Bricks))
	var boss *BrickView
	for i := range g.level.Bricks {
		b := &g.level.Bricks[i]
		face := ""
		if b.Boss {
			face = g.bossFace(b)
		}
		bricks[i] = BrickView{b.ID, b.X, b.Y, b.W, b.H, b.HP, b.MaxHP, b.Steel, b.Palette, b.Special, b.Boss, b.BossName, b.BossKind, b.BossStage, b.BossStages, b.BossVX, b.BossAvoiding, b.BossThreatX, face}
		if b.Boss && b.HP > 0 {
			copy := bricks[i]
			boss = &copy
		}
	}
	high := append([]HighScore(nil), g.highScores...)
	if len(high) > 5 {
		high = high[:5]
	}
	s := Snapshot{Mode: g.mode, Paused: g.paused, Time: g.time, Level: g.levelIndex + 1, LevelCount: LevelCount(), LevelName: g.level.Name, LevelSubtitle: g.level.Subtitle, Score: g.score, Lives: g.lives, Combo: g.combo, Multiplier: g.multiplier(), BricksLeft: g.bricksLeft, Balls: balls, BallCount: len(balls), Paddle: g.paddle, Bricks: bricks, Boss: boss, Drops: append([]Drop(nil), g.drops...), BossShots: append([]BossShot(nil), g.bossShots...), Message: g.message, DHHLeft: g.dhhLeft, WideLeft: g.wideLeft, SlowLeft: g.slowLeft, FireLeft: g.fireLeft, StickyLeft: g.stickyLeft, MagnetLeft: g.magnetLeft, ShieldCharges: g.shieldCharges, StickyHeld: g.heldBallCount(), StickyAim: g.stickyAim, LaunchSpeed: g.level.Speed, TargetSpeed: g.targetBallSpeed(), BasePaddleWidth: g.levelPaddleWidth(), Secrets: g.secrets, HighScores: high, Events: events, PowerRainCount: g.powerRainCount, PowerRainCooldown: g.powerRainCooldown}
	if len(g.highScores) > 0 {
		s.HighScore = g.highScores[0].Score
	}
	if g.level.PowerRain != nil {
		s.PowerRainMax = g.level.PowerRain.Max
	}
	return s
}

// LoadHighScoreState normalizes persisted data without affecting play.
func (g *Game) LoadHighScoreState(state HighScoreState) {
	clean := make([]HighScore, 0, 10)
	for _, h := range state.HighScores {
		if h.Score < 0 {
			continue
		}
		h.Level = clampInt(h.Level, 1, LevelCount())
		clean = append(clean, h)
	}
	sort.SliceStable(clean, func(i, j int) bool { return clean[i].Score > clean[j].Score })
	if len(clean) > 10 {
		clean = clean[:10]
	}
	g.highScores = clean
	g.lastScore = max(0, state.LastScore)
}

func (g *Game) HighScoreState() HighScoreState {
	return HighScoreState{1, append([]HighScore(nil), g.highScores...), g.lastScore}
}
