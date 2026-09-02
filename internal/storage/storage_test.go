package storage

import (
	"breakout-terminal/internal/game"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRoundTripAndCorruptState(t *testing.T) {
	s := &Store{dir: filepath.Join(t.TempDir(), "breakout")}
	scores := game.HighScoreState{Version: 1, LastScore: 7, HighScores: []game.HighScore{{Score: 99, Level: 3}}}
	if err := s.SaveHighScores(scores); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadHighScores()
	if err != nil || !reflect.DeepEqual(got, scores) {
		t.Fatalf("%#v %v", got, err)
	}
	if err := s.SaveSettings(Settings{Muted: true}); err != nil {
		t.Fatal(err)
	}
	settings, err := s.LoadSettings()
	if err != nil || !settings.Muted || settings.Version != 1 {
		t.Fatalf("%#v %v", settings, err)
	}
}

func TestReferenceCompatibleHighScoreNormalization(t *testing.T) {
	s := &Store{dir: filepath.Join(t.TempDir(), "breakout")}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		t.Fatal(err)
	}
	fixture := `{"version":1,"highScores":[{"score":12.9,"level":2.8,"completed":true},{"score":"40.7","level":"99","completed":"true"},{"score":-1,"level":1},{"score":"bad","level":4},{"score":7,"level":null},{"level":3},null],"lastScore":"-8.2"}`
	if err := os.WriteFile(filepath.Join(s.dir, "highscores.json"), []byte(fixture), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadHighScores()
	if err != nil {
		t.Fatal(err)
	}
	want := game.HighScoreState{Version: 1, HighScores: []game.HighScore{{Score: 40, Level: 25}, {Score: 12, Level: 2, Completed: true}, {Score: 7, Level: 1}}, LastScore: 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
