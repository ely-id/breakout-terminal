package game

type levelDef struct {
	name, subtitle, pattern string
	speed                   float64
	colors                  []string
	core                    string
	coreRow, coreCol        int
	arena                   bool
	boss                    bossDef
	rain                    *PowerRain
}

type bossDef struct {
	name, kind string
	hp         int
	w, h       float64
}

var levelDefs = []levelDef{
	{"THE BOOT", "Everything starts at the prompt", "boot", 560, []string{"accent", "blue"}, "", -1, -1, false, bossDef{}, nil},
	{"ARCH BTW", "A rolling release of bricks", "arch", 578, []string{"blue", "cyan"}, "", -1, -1, false, bossDef{}, nil},
	{"TOKYO NIGHT", "The default after dark", "tokyo", 596, []string{"magenta", "blue"}, "", -1, -1, false, bossDef{}, nil},
	{"DHH'S DESK", "Ship the whole stack", "desk", 614, []string{"red", "yellow"}, "DHH", 0, 9, false, bossDef{}, nil},
	{"RYAN'S RUN", "Keep the shell moving", "runner", 632, []string{"green", "cyan"}, "RYAN", 0, 9, false, bossDef{}, nil},
	{"BASECAMP HILL", "Clarity over chaos", "hill", 650, []string{"green", "yellow"}, "", -1, -1, false, bossDef{}, nil},
	{"SIGNAL 37", "Can you read the signal?", "signal37", 678, []string{"yellow", "red"}, "", -1, -1, false, bossDef{}, nil},
	{"CAMPFIRE", "Gather around the glow", "campfire", 696, []string{"red", "yellow"}, "", -1, -1, false, bossDef{}, nil},
	{"HEY", "Email deserves better", "hey", 714, []string{"yellow", "green"}, "", -1, -1, false, bossDef{}, nil},
	{"WRITEBOOK", "A book is just a stack", "book", 732, []string{"cyan", "blue"}, "", -1, -1, false, bossDef{}, nil},
	{"RAILS", "Convention over configuration", "rails", 750, []string{"red", "bright"}, "", -1, -1, false, bossDef{}, nil},
	{"HOTWIRE", "HTML over the wire", "hotwire", 768, []string{"magenta", "yellow"}, "", -1, -1, false, bossDef{}, nil},
	{"TURBO", "Navigate without slowing down", "turbo", 796, []string{"green", "cyan"}, "", -1, -1, false, bossDef{}, nil},
	{"STIMULUS", "A little JavaScript goes far", "stimulus", 814, []string{"blue", "yellow"}, "", -1, -1, false, bossDef{}, nil},
	{"KAMAL", "Deploy anywhere", "container", 832, []string{"blue", "cyan"}, "", -1, -1, false, bossDef{}, nil},
	{"SOLID CABLE", "Realtime, simplified", "cable", 850, []string{"green", "magenta"}, "", -1, -1, false, bossDef{}, nil},
	{"ONCE", "Buy it once, own it forever", "once", 868, []string{"yellow", "bright"}, "", -1, -1, false, bossDef{}, nil},
	{"REWORK", "Underdo the competition", "rework", 886, []string{"red", "bright"}, "", -1, -1, false, bossDef{}, nil},
	{"SHAPE UP", "Appetite, pitch, bet, build", "shapeup", 914, []string{"magenta", "cyan"}, "", -1, -1, false, bossDef{}, nil},
	{"NO MEETINGS", "Protect the maker's schedule", "nomeetings", 932, []string{"cyan", "green"}, "", -1, -1, false, bossDef{}, nil},
	{"SMALL TEAMS", "Constraints make the game", "teams", 950, []string{"yellow", "blue"}, "", -1, -1, false, bossDef{}, nil},
	{"RUNNING IN PRODUCTION", "The test is the real thing", "production", 968, []string{"green", "red"}, "", -1, -1, false, bossDef{}, nil},
	{"OMARCHY", "Beautiful. Modern. Opinionated.", "omarchy", 986, []string{"accent", "magenta"}, "OMARCHY", 10, 9, false, bossDef{}, nil},
	{"ONE PERSON FRAMEWORK", "You have everything you need", "finale", 1020, []string{"red", "yellow"}, "DHH", 5, 9, false, bossDef{}, nil},
	{"ONE MORE THING", "A bite after the framework", "apple", 1040, []string{"red", "green"}, "", -1, -1, true, bossDef{"THE BITE", "apple", 16, 220, 198}, &PowerRain{1.35, 6.5, 4, []string{"shield", "multi", "fire", "slow"}}},
}

var levelArt = map[string][]string{
	"boot":       {"..###############..", "..#.............#..", "..#..##.........#..", "..#.#..#........#..", "..#..##...#####.#..", "..#.............#..", "..#.............#..", "..###############..", "........#.#........", ".....#########.....", "...#############..."},
	"arch":       {".........#.........", "........###........", ".......#####.......", "......###.###......", ".....###...###.....", "....###.....###....", "...###.......###...", "..#####.....#####..", ".######.....######.", "######.......######", "....###########...."},
	"tokyo":      {"..............###..", ".............#####.", "..###############..", "...#############...", "......#.....#......", "......#.....#......", "....###.....###....", "....#.........#....", ".##.#..##.###..##..", ".##.#######.#####..", "###################"},
	"desk":       {"....###########....", "....#.........#....", "....#..#####..#.##.", "....#.........#.##.", "....###########....", ".........#.........", ".......#####.......", "...................", ".#################.", "...#...........#...", "...#...........#..."},
	"runner":     {"........###........", "........###....###.", ".........#..###....", "......######.......", "....###..#..###....", ".........#.........", "........###........", ".......##.##.......", "......##...##......", "###..##.....##.....", "#####.........####."},
	"hill":       {".........#.........", ".........#####.....", ".........#.........", "........###........", ".......#####.......", "......###.###......", ".....###...###.....", "....###.....###....", "...###...#...###...", "..###...###...###..", "###################"},
	"signal37":   {"#.................#", ".##.....###.....##.", "...##..#####..##...", ".....##..#..##.....", "........###........", ".........#.........", ".......#####.......", "......#..#..#......", ".........#.........", "......#######......", "....###########...."},
	"campfire":   {".........#.........", "........###........", "......##.##........", ".......#####.......", ".....####.####.....", ".....#########.....", "......#######......", ".......#####.......", "...####.....####...", ".....#########.....", "...####.....####..."},
	"hey":        {".#############..#..", ".#...........#..#..", ".##.........##..#..", ".#.##.....##.#..#..", ".#...##.##...#..#..", ".#.....#.....#..#..", ".#...........#..#..", ".#############.....", "...................", "................#..", "..................."},
	"book":       {".........#.........", "..######.#.######..", ".#......###......#.", ".#.......#.......#.", ".#..###..#..###..#.", ".#.......#.......#.", ".#..####.#.####..#.", ".#.......#.......#.", ".#.......#.......#.", "..######.#.######..", ".........#........."},
	"rails":      {".......#...#.......", ".......#...#.......", "......#.....#......", ".....#########.....", ".....#.......#.....", "....#.........#....", "...#############...", "...#...........#...", "..#.............#..", ".#################.", "#.................#"},
	"hotwire":    {"###................", "..#................", "..########.........", ".........#...#.....", ".........#..##.....", ".........####......", ".......#####.......", ".........####......", "..........##..#....", "..........#...#....", "..........#####...."},
	"turbo":      {"........#####.......", "......###...###.....", ".....##.......##....", "....##...###...##...", "....#...##.##...#...", "....#...##.##...#...", "....##...###...##...", ".....##.......##..#.", "......####.####..##.", "..........##...####.", "###...###......#####"},
	"stimulus":   {"..........##.......", ".........##........", "........##.........", ".......#####.......", ".........##........", "........##.........", ".......##..........", "..##...#...##...##.", ".####....####.####.", "..##.......##...##.", "...#........#....#."},
	"container":  {"..###.###.###.###..", "..#.#.#.#.#.#.#.#..", "..###.###.###.###..", "....###.###.###....", "....#.#.#.#.#.#....", "....###.###.###....", ".#################.", "..###############..", "...#############...", "....###########....", "##...............##"},
	"cable":      {"###.............###", "#.#.............#.#", "###.............###", "..#.............#..", "..##...........##..", "...##.........##...", "....###########....", "...................", "..###.........###..", "..#.#.........#.#..", "..###.........###.."},
	"once":       {".......#####.......", ".....##.....##.....", "....#....#....#....", "...#...###.....#...", "...#.....#.....#...", "...#.....#.....#...", "...#.....#.....#...", "...#...#####...#...", "....#.........#....", ".....##.....##.....", ".......#####......."},
	"rework":     {".....########......", "...###.....#####...", "..##.........####..", ".##.............###", "##......###......##", "##.....#####.....##", ".##.....###.....##.", "..####.........##..", "...#####.....###...", "......########.....", "..................."},
	"shapeup":    {"..............#....", ".............###...", "............#####..", "...........#######.", "......#####........", "......#...#........", "......#...#........", "......#####........", ".###...............", "#...#..............", ".###..............."},
	"nomeetings": {"....##.......##....", "..###############..", "..#.............#..", "..#..##.....##..#..", "..#...##...##...#..", "..#....##.##....#..", "..#.....###.....#..", "..#....##.##....#..", "..#...##...##...#..", "..#..##.....##..#..", "..###############.."},
	"teams":      {"..##.....##.....##.", ".####...####...####", "..##.....##.....##.", ".####...####...####", "#.##.#.#.##.#.#.##.", "...#.....#.....#...", "...#############...", "...#.....#.....#...", "..###...###...###..", ".#####.#####.#####.", "..#.#...#.#...#.#.."},
	"production": {"....##.............", "...####....##......", "...####...####.....", "...####...####.....", "..###############..", "..#..###..###...#..", "..#..#.#..#.#...#..", "..#..###..###...#..", "..#.............#..", "..###############..", "###.##.##.##.##.###"},
	"omarchy":    {"##################", "#.......##.......#", "#..#######...##..#", "#..#..........#..#", "#..#..........#..#", "#..#..........#..#", "####..........#..#", "#..#..........#..#", "#..#..........#..#", "#..#..........#..#", "#..############..#", "#.......##.......#", "##########.#######"},
	"finale":     {"###################", "#.................#", "#........#........#", "#.......###.......#", "#........#........#", "#......#####......#", "#........#........#", "#.......#.#.......#", "#......#...#......#", "#.................#", "###################"},
}

var bossDefs = []bossDef{{"SCOPE CREEP", "creep", 6, 268, 64}, {"FULL RELOAD", "reload", 8, 244, 60}, {"MEETING MAZE", "maze", 10, 256, 62}, {"THE MONOLITH", "monolith", 12, 196, 74}}

func hash32(n uint32) uint32 {
	n = (n ^ (n >> 16)) * 0x45d9f3b
	n = (n ^ (n >> 16)) * 0x45d9f3b
	return n ^ (n >> 16)
}

func steelAt(pattern string, r, c, rows int) bool {
	switch pattern {
	case "container":
		return (r == 0 || r == rows-1) && c%3 == 0
	case "production":
		return (r+c)%9 == 0
	case "finale":
		return r == 4 && (c == 3 || c == 6)
	case "rails":
		return c == 4 && r%3 == 1
	}
	return false
}

// LevelCount returns the fixed campaign length.
func LevelCount() int { return len(levelDefs) }

// CreateLevel deterministically constructs a fresh campaign level.
func CreateLevel(index int) Level {
	index = clampInt(index, 0, len(levelDefs)-1)
	d := levelDefs[index]
	if d.arena {
		b := makeBoss(index, d.boss, (WorldWidth-d.boss.w)/2, 236)
		rain := *d.rain
		rain.Kinds = append([]string(nil), d.rain.Kinds...)
		return Level{index, index + 1, d.name, d.subtitle, d.speed, append([]string(nil), d.colors...), []Brick{b}, 1, 0, &rain}
	}
	art := levelArt[d.pattern]
	rows, cols := len(art), len(art[0])
	brickW := float64((734 - (cols-1)*4) / cols)
	layoutW := float64(cols)*brickW + float64(cols-1)*4
	startX := float64(int((WorldWidth - layoutW) / 2))
	bricks := make([]Brick, 0, 120)
	breakable := 0
	powerKinds := []string{"wide", "slow", "life", "multi", "fire", "sticky", "shield", "magnet"}
	for r, row := range art {
		for c := range row {
			if row[c] != '#' {
				continue
			}
			steel := steelAt(d.pattern, r, c, rows)
			tough := !steel && index >= 5 && hash32(uint32(index*2000+r*71+c*13))%100 < uint32(minInt(38, 8+index))
			hp := 1
			if steel {
				hp = -1
			} else if tough {
				hp = 2
			}
			special := ""
			if !steel && d.core != "" && r == d.coreRow && c == d.coreCol {
				special = d.core
			} else if index == 6 && (r+c)%5 == 0 {
				special = "37"
			}
			power := ""
			if !steel && hash32(uint32(index*5000+r*97+c*17))%100 < 12 {
				power = powerKinds[hash32(uint32(index+r*11+c*7))%uint32(len(powerKinds))]
			}
			bricks = append(bricks, Brick{ID: brickID(index, r, c), X: startX + float64(c)*(brickW+4), Y: 154 + float64(r)*26, W: brickW, H: 20, HP: hp, MaxHP: hp, Steel: steel, Row: r, Col: c, Palette: d.colors[(r+c)%len(d.colors)], Special: special, Power: power})
			if !steel {
				breakable++
			}
		}
	}
	bossIndex := -1
	if (index+1)%6 == 0 {
		bd := bossDefs[index/6]
		b := makeBoss(index, bd, float64(int((WorldWidth-bd.w)/2)), 154+float64(rows)*26+24)
		b.Row, b.Col, b.Palette = rows+1, cols/2, d.colors[0]
		bricks = append(bricks, b)
		bossIndex, breakable = len(bricks)-1, breakable+1
	}
	return Level{index, index + 1, d.name, d.subtitle, d.speed, append([]string(nil), d.colors...), bricks, breakable, bossIndex, nil}
}

func makeBoss(index int, d bossDef, x, y float64) Brick {
	return Brick{ID: bossID(index), X: x, Y: y, W: d.w, H: d.h, HP: d.hp, MaxHP: d.hp, Palette: levelDefs[index].colors[0], Boss: true, BossName: d.name, BossKind: d.kind, BossStage: 1, BossStages: 3, BossDirection: 1, BossThreatX: 400}
}

func brickID(level, row, col int) string {
	return "L" + itoa(level+1) + "-" + itoa(row) + "-" + itoa(col)
}
func bossID(level int) string { return "L" + itoa(level+1) + "-BOSS" }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func clampInt(v, lo, hi int) int {
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
