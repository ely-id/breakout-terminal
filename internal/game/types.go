// Package game implements Breakout's deterministic rules without terminal,
// filesystem, audio, environment, or wall-clock dependencies.
package game

const (
	WorldWidth  = 800.0
	WorldHeight = 1000.0
	FixedStep   = 1.0 / 240.0
	BallRadius  = 9.0
	PaddleY     = 925.0
	ShieldY     = 944.0
)

type Mode string

const (
	ModeAttract    Mode = "attract"
	ModeServe      Mode = "serve"
	ModePlaying    Mode = "playing"
	ModeLevelClear Mode = "levelClear"
	ModeLifeLost   Mode = "lifeLost"
	ModeGameOver   Mode = "gameOver"
	ModeVictory    Mode = "victory"
)

type Action string

const (
	MoveLeft  Action = "left"
	MoveRight Action = "right"
	AimLeft   Action = "aimLeft"
	AimRight  Action = "aimRight"
	Launch    Action = "launch"
	NewGame   Action = "newGame"
	Pause     Action = "pause"
)

type Brick struct {
	ID                          string
	X, Y, W, H                  float64
	HP, MaxHP                   int
	Steel                       bool
	Row, Col                    int
	Palette, Special, Power     string
	Boss                        bool
	BossName, BossKind          string
	BossStage, BossStages       int
	BossVX                      float64
	BossDirection               int
	BossAvoiding                bool
	BossThreatX                 float64
	BossOuchLeft, BossShootLeft float64
}

type PowerRain struct {
	Delay, Interval float64
	Max             int
	Kinds           []string
}

type Level struct {
	Index     int
	Number    int
	Name      string
	Subtitle  string
	Speed     float64
	Colors    []string
	Bricks    []Brick
	Breakable int
	BossIndex int
	PowerRain *PowerRain
}

type Ball struct {
	X, Y, VX, VY float64
	R            float64
	Trail        []Point
	Stuck        bool
	StickOffset  float64
	pierce       map[string]bool
}

type Point struct{ X, Y float64 }

type Paddle struct{ X, Y, W, H, VX float64 }

type Drop struct {
	X, Y, VX, VY float64
	Kind         string
}

type BossShot struct {
	X, Y, VX, VY, R float64
	Stage           int
}

type Secrets struct{ DHH, Basecamp, Signal37, Konami bool }

type Event struct {
	Type string
	Data map[string]any
	Time float64
}

type HighScore struct {
	Score     int  `json:"score"`
	Level     int  `json:"level"`
	Completed bool `json:"completed"`
}

type HighScoreState struct {
	Version    int         `json:"version"`
	HighScores []HighScore `json:"highScores"`
	LastScore  int         `json:"lastScore"`
}

type BallView struct {
	X, Y, VX, VY, R float64
	Stuck           bool
	Trail           []Point
}

type BrickView struct {
	ID                    string
	X, Y, W, H            float64
	HP, MaxHP             int
	Steel                 bool
	Palette, Special      string
	Boss                  bool
	BossName, BossKind    string
	BossStage, BossStages int
	BossVX                float64
	BossAvoiding          bool
	BossThreatX           float64
	BossFace              string
}

type Snapshot struct {
	Mode                         Mode
	Paused                       bool
	Time                         float64
	Level, LevelCount            int
	LevelName, LevelSubtitle     string
	Score, HighScore             int
	HighScores                   []HighScore
	Lives, Combo, Multiplier     int
	BricksLeft                   int
	Balls                        []BallView
	BallCount                    int
	Paddle                       Paddle
	Bricks                       []BrickView
	Boss                         *BrickView
	Drops                        []Drop
	BossShots                    []BossShot
	Message                      string
	DHHLeft, WideLeft, SlowLeft  float64
	FireLeft, StickyLeft         float64
	MagnetLeft                   float64
	ShieldCharges, StickyHeld    int
	StickyAim                    float64
	LaunchSpeed, TargetSpeed     float64
	BasePaddleWidth              float64
	Secrets                      Secrets
	Events                       []Event
	PowerRainCount, PowerRainMax int
	PowerRainCooldown            float64
}
