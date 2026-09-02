package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"breakout-terminal/internal/app"
	"breakout-terminal/internal/game"
	brender "breakout-terminal/internal/render"
	"breakout-terminal/internal/storage"
	tea "charm.land/bubbletea/v2"
)

func terminalFile(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func utf8Locale() bool {
	locale := os.Getenv("LC_ALL")
	if locale == "" {
		locale = os.Getenv("LC_CTYPE")
	}
	if locale == "" {
		locale = os.Getenv("LANG")
	}
	locale = strings.ToUpper(locale)
	return strings.Contains(locale, "UTF-8") || strings.Contains(locale, "UTF8")
}

func bubbleTeaEnvironment(environment []string) []string {
	// Bubble Tea's transitive color detector executes `tmux info` whenever
	// TMUX is present. The standalone port must not invoke helper processes;
	// TERM/COLORTERM/terminfo detection remains available without this entry.
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		if strings.HasPrefix(entry, "TMUX=") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func main() {
	ascii := flag.Bool("ascii", false, "use ASCII-only rendering")
	colorMode := flag.String("color", "auto", "color mode: auto, truecolor, 256, 16, mono")
	level := flag.Int("level", 0, "start a fresh run at level 1-25")
	flag.Parse()
	if !terminalFile(os.Stdin) || !terminalFile(os.Stdout) {
		fmt.Fprintln(os.Stderr, "breakout: standard input and output must be terminals")
		os.Exit(2)
	}
	profile := brender.TrueColor
	mode := strings.ToLower(*colorMode)
	if os.Getenv("NO_COLOR") != "" && mode == "auto" {
		mode = "mono"
	}
	switch mode {
	case "auto", "truecolor":
		profile = brender.TrueColor
	case "256":
		profile = brender.ANSI256
	case "16":
		profile = brender.ANSI16
	case "mono":
		profile = brender.Monochrome
	default:
		fmt.Fprintln(os.Stderr, "breakout: invalid -color value")
		os.Exit(2)
	}
	g := game.New()
	store, _ := storage.New()
	muted := false
	if store != nil {
		if state, err := store.LoadHighScores(); err == nil {
			g.LoadHighScoreState(state)
		}
		if settings, err := store.LoadSettings(); err == nil {
			muted = settings.Muted
		}
	}
	if *level != 0 && !g.StartLevel(*level) {
		fmt.Fprintln(os.Stderr, "breakout: -level must be between 1 and 25")
		os.Exit(2)
	}
	r := brender.New(brender.Options{ASCII: *ascii || !utf8Locale(), Profile: profile})
	model := app.New(app.Config{Game: g, Renderer: r, Store: store, Muted: muted})
	program := tea.NewProgram(model, tea.WithEnvironment(bubbleTeaEnvironment(os.Environ())))
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "breakout:", err)
		os.Exit(1)
	}
}
