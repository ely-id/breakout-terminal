package game

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func advanceFor(g *Game, d time.Duration) {
	for d > 0 {
		step := 10 * time.Millisecond
		if d < step {
			step = d
		}
		g.Advance(step.Seconds())
		d -= step
	}
}

func TestFixedStepAccumulationAndClamp(t *testing.T) {
	g := New()
	g.Advance(time.Millisecond.Seconds())
	if g.time != 0 {
		t.Fatalf("substep advanced: %v", g.time)
	}
	g.Advance((4 * time.Millisecond).Seconds())
	if math.Abs(g.time-fixedStepSeconds) > 1e-12 {
		t.Fatalf("want one tick, got %.12f", g.time)
	}
	h := New()
	h.Advance(time.Second.Seconds())
	if math.Abs(h.time-.05) > 1e-10 {
		t.Fatalf("50ms clamp failed: %.12f", h.time)
	}
}

func TestPauseFreezesAllEngineTime(t *testing.T) {
	g := New()
	g.StartLevel(1)
	g.TogglePause()
	g.Advance(time.Second.Seconds())
	if g.time != 0 {
		t.Fatal("paused game advanced")
	}
	if g.messageLeft != 2.4 {
		t.Fatal("paused message timer advanced")
	}
	g.TogglePause()
	g.Advance((10 * time.Millisecond).Seconds())
	if g.time == 0 {
		t.Fatal("resume did not advance")
	}
}

func TestPaddleIntentClampAndOpposites(t *testing.T) {
	g := New()
	g.StartLevel(1)
	start := g.paddle.X
	g.KeyDown(MoveRight)
	advanceFor(g, 100*time.Millisecond)
	if math.Abs(g.paddle.X-(start+66)) > .01 {
		t.Fatalf("paddle x %.3f", g.paddle.X)
	}
	g.KeyDown(MoveLeft)
	advanceFor(g, 50*time.Millisecond)
	if math.Abs(g.paddle.X-(start+66)) > .01 {
		t.Fatal("opposite keys should cancel")
	}
	g.KeyUp(MoveRight)
	advanceFor(g, time.Second)
	if g.paddle.X != g.paddle.W/2+20 {
		t.Fatalf("left clamp %.2f", g.paddle.X)
	}
}

func TestWallPaddleAndBrickCollisions(t *testing.T) {
	g := New()
	g.StartLevel(1)
	g.mode = ModePlaying
	g.balls = []Ball{makeBall(21, 500, -600, 0)}
	g.updateBalls(fixedStepSeconds)
	if g.balls[0].VX <= 0 || g.balls[0].X < 29 {
		t.Fatal("left wall did not reflect")
	}
	g.combo = 5
	g.balls = []Ball{makeBall(g.paddle.X, g.paddle.Y-g.paddle.H/2-BallRadius-1, 0, 600)}
	g.updateBalls(fixedStepSeconds)
	if g.balls[0].VY >= 0 || g.combo != 1 {
		t.Fatal("paddle did not return/reset combo")
	}
	b := &g.level.Bricks[0]
	g.balls = []Ball{makeBall(b.X+b.W/2, b.Y-BallRadius-.1, 0, 600)}
	before := g.bricksLeft
	g.updateBalls(fixedStepSeconds)
	if b.HP != 0 || g.bricksLeft != before-1 {
		t.Fatalf("brick collision failed hp=%d left=%d", b.HP, g.bricksLeft)
	}
}

func TestReinforcedSteelAndScoring(t *testing.T) {
	g := New()
	g.StartLevel(6)
	g.mode = ModePlaying
	var tough, steel *Brick
	for i := range g.level.Bricks {
		b := &g.level.Bricks[i]
		if b.MaxHP == 2 && tough == nil {
			tough = b
		}
	}
	if tough == nil {
		t.Fatal("no reinforced brick")
	}
	g.hitBrick(tough)
	if tough.HP != 1 || g.bricksLeft != g.level.Breakable || g.score != 25 {
		t.Fatalf("reinforced damage mismatch hp=%d score=%d", tough.HP, g.score)
	}
	g.hitBrick(tough)
	if tough.HP != 0 || g.combo != 2 {
		t.Fatal("reinforced brick did not break")
	}
	g.loadLevel(14, false)
	g.mode = ModePlaying
	for i := range g.level.Bricks {
		if g.level.Bricks[i].Steel {
			steel = &g.level.Bricks[i]
			break
		}
	}
	if steel == nil {
		t.Fatal("no steel brick")
	}
	score, left := g.score, g.bricksLeft
	g.hitBrick(steel)
	if steel.HP != -1 || g.score != score || g.bricksLeft != left {
		t.Fatal("steel changed state")
	}
	h := New()
	h.StartLevel(1)
	h.mode = ModePlaying
	h.hitBrick(&h.level.Bricks[0])
	if h.score != 200 || h.combo != 2 {
		t.Fatalf("first break score=%d combo=%d", h.score, h.combo)
	}
}

func TestMultiballOnlyLosesFinalBall(t *testing.T) {
	g := New()
	g.StartLevel(1)
	g.mode = ModePlaying
	g.balls = []Ball{makeBall(200, 500, 0, 0), makeBall(200, 1010, 0, 0)}
	g.updateBalls(fixedStepSeconds)
	if len(g.balls) != 1 || g.lives != 3 || g.mode != ModePlaying {
		t.Fatal("draining one multiball cost a life")
	}
	g.balls[0] = makeBall(200, 1010, 0, 0)
	g.updateBalls(fixedStepSeconds)
	if g.lives != 2 || g.mode != ModeLifeLost {
		t.Fatal("final drain did not enter lifeLost")
	}
}

func TestPowerUpsAndStickyRelease(t *testing.T) {
	g := New()
	g.StartLevel(1)
	g.mode = ModePlaying
	g.balls = []Ball{makeBall(400, 500, 100, -500)}
	g.activatePower("wide")
	if g.paddle.W < 190 || g.wideLeft != 15 {
		t.Fatal("wide")
	}
	g.activatePower("slow")
	if g.balls[0].VX != 72 || g.slowLeft != 12 {
		t.Fatal("slow")
	}
	g.lives = 7
	g.activatePower("life")
	if g.lives != 7 {
		t.Fatal("life cap")
	}
	g.activatePower("multi")
	if len(g.balls) != 3 {
		t.Fatal("multi")
	}
	g.activatePower("shield")
	g.activatePower("shield")
	if g.shieldCharges != 1 {
		t.Fatal("shield stacked")
	}
	g.balls = []Ball{makeBall(g.paddle.X, g.paddle.Y-g.paddle.H/2-BallRadius-1, 0, 400)}
	g.activatePower("sticky")
	g.updateBalls(fixedStepSeconds)
	if !g.balls[0].Stuck {
		t.Fatal("sticky did not catch")
	}
	if !g.Launch() || g.balls[0].Stuck || g.balls[0].VY >= 0 {
		t.Fatal("sticky did not release")
	}
}

func TestBossStagesAttacksAndProjectileLifeLoss(t *testing.T) {
	g := New()
	g.StartLevel(6)
	g.mode = ModePlaying
	boss := g.activeBoss()
	if boss == nil || boss.HP != 6 {
		t.Fatal("boss missing")
	}
	g.hitBrick(boss)
	g.hitBrick(boss)
	if boss.BossStage != 2 || len(g.bossShots) != 0 || g.bossShotCooldown != .55 {
		t.Fatal("stage two transition")
	}
	g.hitBrick(boss)
	g.hitBrick(boss)
	if boss.BossStage != 3 {
		t.Fatal("stage three transition")
	}
	h := New()
	h.StartLevel(12)
	h.Launch()
	h.balls = []Ball{makeBall(100, 700, 0, 0)}
	boss = h.activeBoss()
	h.hitBrick(boss)
	h.hitBrick(boss)
	h.hitBrick(boss)
	advanceFor(h, 650*time.Millisecond)
	if len(h.bossShots) != 2 {
		t.Fatalf("stage two volley=%d", len(h.bossShots))
	}
	h.bossShots = []BossShot{{h.paddle.X, h.paddle.Y, 0, 0, 6, 2}}
	h.updateBossAttack(fixedStepSeconds)
	if h.lives != 2 || h.mode != ModeLifeLost {
		t.Fatal("boss projectile did not cost one life")
	}
}

func TestLifeLevelGameOverAndVictoryTransitions(t *testing.T) {
	g := New()
	g.StartLevel(1)
	g.mode = ModePlaying
	g.loseBall()
	if g.mode != ModeLifeLost || g.lives != 2 {
		t.Fatal("life loss")
	}
	advanceFor(g, 1100*time.Millisecond)
	if g.mode != ModeServe || len(g.balls) != 1 {
		t.Fatal("serve delay")
	}
	g.mode = ModePlaying
	g.completeLevel()
	if g.mode != ModeLevelClear {
		t.Fatal("level clear")
	}
	advanceFor(g, 1500*time.Millisecond)
	if g.mode != ModeServe || g.levelIndex != 1 {
		t.Fatal("next level delay")
	}
	h := New()
	h.StartLevel(1)
	for i := 0; i < 3; i++ {
		h.mode = ModePlaying
		h.balls = []Ball{makeBall(400, 500, 0, 0)}
		h.loseBall()
	}
	if h.mode != ModeGameOver || len(h.highScores) != 1 {
		t.Fatal("game over")
	}
	v := New()
	v.StartLevel(25)
	v.mode = ModePlaying
	boss := v.activeBoss()
	for boss.HP > 0 {
		v.hitBrick(boss)
	}
	if v.mode != ModeVictory || !v.highScores[0].Completed {
		t.Fatal("victory")
	}
}

func TestHighScoreNormalization(t *testing.T) {
	g := New()
	g.LoadHighScoreState(HighScoreState{LastScore: 9, HighScores: []HighScore{{3, 99, false}, {20, 2, true}, {-1, 1, false}, {20, 5, false}}})
	state := g.HighScoreState()
	if len(state.HighScores) != 3 || state.HighScores[0].Score != 20 || state.HighScores[0].Level != 2 || state.HighScores[2].Level != 25 || state.LastScore != 9 {
		t.Fatalf("normalized %#v", state)
	}
}

func TestDeterministicCampaignAndSilhouettes(t *testing.T) {
	wantCounts := []int{77, 78, 91, 64, 54, 69, 54, 67, 57, 65, 55, 41, 73, 45, 110, 53, 44, 73, 38, 77, 96, 83, 98, 72, 1}
	if LevelCount() != 25 {
		t.Fatal(LevelCount())
	}
	powers := map[string]bool{}
	for i := 0; i < LevelCount(); i++ {
		a, b := CreateLevel(i), CreateLevel(i)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("level %d nondeterministic", i+1)
		}
		if a.Breakable != wantCounts[i] {
			t.Fatalf("level %d breakable=%d", i+1, a.Breakable)
		}
		for _, br := range a.Bricks {
			if br.Power != "" {
				powers[br.Power] = true
			}
		}
		if i < 24 {
			art := levelArt[levelDefs[i].pattern]
			occ := map[[2]int]bool{}
			for _, br := range a.Bricks {
				if !br.Boss {
					occ[[2]int{br.Row, br.Col}] = true
				}
			}
			for row, line := range art {
				for col, ch := range line {
					if (ch == '#') != occ[[2]int{row, col}] {
						t.Fatalf("silhouette mismatch level %d %d:%d", i+1, row, col)
					}
				}
			}
		}
	}
	if len(powers) != 8 {
		t.Fatalf("power kinds=%v", powers)
	}
}

func TestReplayIndependentOfAdapterRate(t *testing.T) {
	run := func(rate int) Snapshot {
		g := New()
		g.StartLevel(1)
		g.Launch()
		g.KeyDown(MoveRight)
		for i := 0; i < rate*2; i++ {
			g.Advance((time.Second / time.Duration(rate)).Seconds())
		}
		g.KeyUp(MoveRight)
		return g.Snapshot()
	}
	a, b := run(30), run(60)
	if a.Mode != b.Mode || a.Score != b.Score || a.BricksLeft != b.BricksLeft || a.Lives != b.Lives || a.BallCount != b.BallCount {
		t.Fatalf("snapshots diverged %#v %#v", a, b)
	}
	if a.BallCount > 0 && (math.Abs(a.Balls[0].X-b.Balls[0].X) > .01 || math.Abs(a.Balls[0].Y-b.Balls[0].Y) > .01) {
		t.Fatalf("ball diverged %.3f,%.3f / %.3f,%.3f", a.Balls[0].X, a.Balls[0].Y, b.Balls[0].X, b.Balls[0].Y)
	}
}

func benchmarkAdvance(b *testing.B, balls int) {
	var g *Game
	for i := 0; i < b.N; i++ {
		if i%30 == 0 {
			b.StopTimer()
			g = New()
			if balls == 1 {
				g.StartLevel(1)
				g.Launch()
			} else {
				g.spawnBallStorm(balls)
			}
			b.StartTimer()
		}
		g.Advance((time.Second / 60).Seconds())
	}
}
func BenchmarkAdvanceOneBall(b *testing.B)      { benchmarkAdvance(b, 1) }
func BenchmarkAdvanceNineBalls(b *testing.B)    { benchmarkAdvance(b, 9) }
func BenchmarkAdvanceHundredBalls(b *testing.B) { benchmarkAdvance(b, 100) }
func BenchmarkSnapshot(b *testing.B) {
	g := New()
	g.spawnBallStorm(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Snapshot()
	}
}
