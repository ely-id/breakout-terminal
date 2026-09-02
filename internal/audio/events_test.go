package audio

import (
	"breakout-terminal/internal/game"
	"testing"
)

func TestCuePriorityAndPitch(t *testing.T) {
	events := []game.Event{{Type: "wall"}, {Type: "brickBroken"}, {Type: "lifeLost"}}
	if got := Select(events); got != "life" {
		t.Fatal(got)
	}
	want := []int{1, 1, 2, 3, 3, 4, 4}
	for i, m := range []int{1, 2, 3, 4, 5, 6, 9} {
		if got := BreakPitchTier(m); got != want[i] {
			t.Fatalf("multiplier %d tier %d", m, got)
		}
	}
	if !IsImpact("break") || IsImpact("level") {
		t.Fatal("impact classes")
	}
}
