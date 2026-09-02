// Package audio preserves the original semantic cue selection boundary. The
// initial terminal port intentionally has no playback backend and is silent.
package audio

import "breakout-terminal/internal/game"

var priorities = []struct {
	cue    string
	events []string
}{
	{"victory", []string{"victory"}}, {"gameover", []string{"gameOver"}}, {"level", []string{"levelCleared"}},
	{"life", []string{"lifeLost", "bossShotHit"}}, {"secret", []string{"konami", "secret", "bossDefeated"}},
	{"power", []string{"powerUp", "shieldSaved", "bossStage"}}, {"launch", []string{"launch", "stickyLaunch", "bossShotFired"}},
	{"ready", []string{"levelStarted"}}, {"break", []string{"brickBroken"}}, {"damage", []string{"brickDamaged"}},
	{"steel", []string{"steel"}}, {"bounce", []string{"paddle", "wall", "stickyCatch"}},
}

func Select(events []game.Event) string {
	present := map[string]bool{}
	for _, e := range events {
		present[e.Type] = true
	}
	for _, p := range priorities {
		for _, e := range p.events {
			if present[e] {
				return p.cue
			}
		}
	}
	return ""
}
func IsImpact(cue string) bool {
	return cue == "break" || cue == "damage" || cue == "steel" || cue == "bounce"
}
func BreakPitchTier(multiplier int) int {
	if multiplier >= 6 {
		return 4
	}
	if multiplier >= 4 {
		return 3
	}
	if multiplier >= 3 {
		return 2
	}
	return 1
}
