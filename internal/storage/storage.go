// Package storage provides best-effort XDG state persistence.
package storage

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"breakout-terminal/internal/game"
)

type Settings struct {
	Version int  `json:"version"`
	Muted   bool `json:"muted"`
}

type Store struct{ dir string }

func New() (*Store, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return nil, errors.New("cannot determine state directory")
		}
		base = filepath.Join(home, ".local", "state")
	}
	return &Store{dir: filepath.Join(base, "breakout")}, nil
}

func (s *Store) load(name string, target any) error {
	data, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (s *Store) save(name string, value any) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(s.dir, ".breakout-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpName, filepath.Join(s.dir, name))
	}
	if err != nil {
		return err
	}
	ok = true
	return nil
}

func (s *Store) LoadHighScores() (game.HighScoreState, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, "highscores.json"))
	if err != nil {
		return game.HighScoreState{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	var root map[string]any
	if err = decoder.Decode(&root); err != nil {
		return game.HighScoreState{}, err
	}
	state := game.HighScoreState{Version: 1}
	if rows, ok := root["highScores"].([]any); ok {
		for _, raw := range rows {
			row, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			scoreRaw, hasScore := row["score"]
			levelRaw, hasLevel := row["level"]
			if !hasScore || !hasLevel {
				continue
			}
			score, scoreOK := coerceNumber(scoreRaw)
			level, levelOK := coerceNumber(levelRaw)
			if !scoreOK || score < 0 || !levelOK {
				continue
			}
			score = math.Floor(score)
			level = math.Floor(level)
			if score >= float64(maxIntValue()) {
				continue
			}
			completed, valueOK := row["completed"].(bool)
			if level < 1 {
				level = 1
			} else if level > float64(game.LevelCount()) {
				level = float64(game.LevelCount())
			}
			state.HighScores = append(state.HighScores, game.HighScore{Score: int(score), Level: int(level), Completed: valueOK && completed})
		}
	}
	sort.SliceStable(state.HighScores, func(i, j int) bool { return state.HighScores[i].Score > state.HighScores[j].Score })
	if len(state.HighScores) > 10 {
		state.HighScores = state.HighScores[:10]
	}
	if raw, ok := root["lastScore"]; ok {
		if value, valid := coerceNumber(raw); valid {
			value = math.Max(0, math.Floor(value))
			if value < float64(maxIntValue()) {
				state.LastScore = int(value)
			}
		}
	}
	return state, nil
}
func (s *Store) SaveHighScores(state game.HighScoreState) error {
	return s.save("highscores.json", state)
}
func (s *Store) LoadSettings() (Settings, error) {
	var v Settings
	err := s.load("settings.json", &v)
	if err == nil && v.Version != 1 {
		return Settings{}, errors.New("unsupported settings version")
	}
	return v, err
}
func (s *Store) SaveSettings(v Settings) error { v.Version = 1; return s.save("settings.json", v) }

func coerceNumber(value any) (float64, bool) {
	var number float64
	var err error
	switch value := value.(type) {
	case json.Number:
		number, err = value.Float64()
	case float64:
		number = value
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return 0, true
		}
		number, err = strconv.ParseFloat(text, 64)
	case bool:
		if value {
			return 1, true
		}
		return 0, true
	case nil:
		return 0, true
	default:
		return 0, false
	}
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	return number, true
}
func maxIntValue() int { return int(^uint(0) >> 1) }
