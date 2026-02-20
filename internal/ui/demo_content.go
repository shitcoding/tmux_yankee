package ui

import (
	"fmt"
	"strings"
)

// DemoPageName returns the human-readable name for each demo page index.
var DemoPageNames = []string{
	"Shell Session",
	"Code Snippet",
	"Color Test Card",
	"Long Lines",
}

// DemoPages returns all demo content pages as a slice of line slices.
func DemoPages() [][]string {
	return [][]string{
		DemoShellSession(),
		DemoCodeSnippet(),
		DemoColorTestCard(),
		DemoLongLines(),
	}
}

// ansi helpers for demo content construction
func bold(s string) string    { return "\x1b[1m" + s + "\x1b[0m" }
func dim(s string) string     { return "\x1b[2m" + s + "\x1b[0m" }
func italic(s string) string  { return "\x1b[3m" + s + "\x1b[0m" }
func uline(s string) string   { return "\x1b[4m" + s + "\x1b[0m" }
func red(s string) string     { return "\x1b[31m" + s + "\x1b[0m" }
func green(s string) string   { return "\x1b[32m" + s + "\x1b[0m" }
func yellow(s string) string  { return "\x1b[33m" + s + "\x1b[0m" }
func blue(s string) string    { return "\x1b[34m" + s + "\x1b[0m" }
func magenta(s string) string { return "\x1b[35m" + s + "\x1b[0m" }
func cyan(s string) string    { return "\x1b[36m" + s + "\x1b[0m" }
func gray(s string) string    { return "\x1b[90m" + s + "\x1b[0m" }
func white(s string) string   { return "\x1b[97m" + s + "\x1b[0m" }
func bgRed(s string) string   { return "\x1b[41m" + s + "\x1b[0m" }
func bgGreen(s string) string { return "\x1b[42m" + s + "\x1b[0m" }
func bgBlue(s string) string  { return "\x1b[44m" + s + "\x1b[0m" }

func fg256(n int, s string) string { return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", n, s) }
func fgRGB(r, g, b int, s string) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, s)
}

// DemoShellSession returns ~80 lines of simulated shell output with ANSI colors.
func DemoShellSession() []string {
	var lines []string

	// Prompt + ls with colored output
	lines = append(lines, bold(green("user@host"))+":"+bold(blue("~/projects/tmux-yankee"))+"$ ls -la")
	lines = append(lines, "total 48")
	lines = append(lines, "drwxr-xr-x  12 user staff  384 Feb 19 10:30 "+bold(blue(".")))
	lines = append(lines, "drwxr-xr-x   5 user staff  160 Feb 18 09:15 "+bold(blue("..")))
	lines = append(lines, "drwxr-xr-x   8 user staff  256 Feb 19 10:30 "+bold(blue(".git")))
	lines = append(lines, "-rw-r--r--   1 user staff  1.2K Feb 19 10:28 LICENSE")
	lines = append(lines, "-rw-r--r--   1 user staff   847 Feb 15 14:22 Makefile")
	lines = append(lines, "-rw-r--r--   1 user staff   312 Feb 14 11:00 go.mod")
	lines = append(lines, "-rw-r--r--   1 user staff  2.4K Feb 19 10:25 go.sum")
	lines = append(lines, "drwxr-xr-x   3 user staff   96 Feb 18 16:40 "+bold(blue("cmd")))
	lines = append(lines, "drwxr-xr-x   6 user staff  192 Feb 19 10:30 "+bold(blue("internal")))
	lines = append(lines, "drwxr-xr-x   4 user staff  128 Feb 17 12:00 "+bold(blue("scripts")))
	lines = append(lines, "-rwxr-xr-x   1 user staff  3.8K Feb 19 09:45 "+bold(green("yank.tmux")))
	lines = append(lines, "")

	// git status
	lines = append(lines, bold(green("user@host"))+":"+bold(blue("~/projects/tmux-yankee"))+"$ git status")
	lines = append(lines, "On branch "+green("main"))
	lines = append(lines, "Your branch is up to date with '"+green("origin/main")+"'.")
	lines = append(lines, "")
	lines = append(lines, "Changes not staged for commit:")
	lines = append(lines, "  (use \"git add <file>...\" to update what will be committed)")
	lines = append(lines, "")
	lines = append(lines, "	"+red("modified:   internal/theme/types.go"))
	lines = append(lines, "	"+red("modified:   internal/theme/presets.go"))
	lines = append(lines, "	"+red("modified:   internal/ui/tui.go"))
	lines = append(lines, "")
	lines = append(lines, "Untracked files:")
	lines = append(lines, "  (use \"git add <file>...\" to include in what will be committed)")
	lines = append(lines, "")
	lines = append(lines, "	"+red("internal/ui/demo_content.go"))
	lines = append(lines, "")
	lines = append(lines, "no changes added to commit (use \"git add\" and/or \"git commit -a\")")
	lines = append(lines, "")

	// git diff (truncated)
	lines = append(lines, bold(green("user@host"))+":"+bold(blue("~/projects/tmux-yankee"))+"$ git diff --stat")
	lines = append(lines, " internal/theme/types.go   | 42 "+green("+++++++++++++++++++++")+red("-----"))
	lines = append(lines, " internal/theme/presets.go  | 18 "+green("++++++++")+red("---"))
	lines = append(lines, " internal/ui/tui.go         | 96 "+green("+++++++++++++++++++++++++++++++++++++")+red("--------"))
	lines = append(lines, " 3 files changed, 112 insertions(+), 44 deletions(-)")
	lines = append(lines, "")

	// git log
	lines = append(lines, bold(green("user@host"))+":"+bold(blue("~/projects/tmux-yankee"))+"$ git log --oneline -5")
	lines = append(lines, yellow("596e57d")+" feat(config): support root key table for yankee binding")
	lines = append(lines, yellow("d932eb5")+" feat(ui): improve wrap mode viewport and rendering")
	lines = append(lines, yellow("f2925c9")+" feat(config): add wrap mode setting (scroll/wrap)")
	lines = append(lines, yellow("4dc3562")+" feat(ui): add word wrapping and wrap-aware viewport for wrap mode")
	lines = append(lines, yellow("c0f34d8")+" build: rebuild binary with horizontal scroll")
	lines = append(lines, "")

	// grep with color
	lines = append(lines, bold(green("user@host"))+":"+bold(blue("~/projects/tmux-yankee"))+"$ grep -rn 'TextStyle' internal/theme/")
	lines = append(lines, magenta("internal/theme/types.go")+":"+green("18")+":type "+red("TextStyle")+" struct {")
	lines = append(lines, magenta("internal/theme/types.go")+":"+green("24")+":	Style "+red("TextStyle"))
	lines = append(lines, magenta("internal/theme/types.go")+":"+green("37")+":	CursorStyle   "+red("TextStyle"))
	lines = append(lines, magenta("internal/theme/types.go")+":"+green("38")+":	AbsoluteStyle "+red("TextStyle"))
	lines = append(lines, magenta("internal/theme/types.go")+":"+green("39")+":	RelativeStyle "+red("TextStyle"))
	lines = append(lines, magenta("internal/theme/presets.go")+":"+green("9")+":		CursorStyle: "+red("TextStyle")+"{Bold: true},")
	lines = append(lines, "")

	// make build
	lines = append(lines, bold(green("user@host"))+":"+bold(blue("~/projects/tmux-yankee"))+"$ make build")
	lines = append(lines, gray("go build -o bin/tmux-yankee ./cmd/tmux-yankee"))
	lines = append(lines, green("Build successful")+" → bin/tmux-yankee")
	lines = append(lines, "")

	// go test
	lines = append(lines, bold(green("user@host"))+":"+bold(blue("~/projects/tmux-yankee"))+"$ go test ./...")
	lines = append(lines, "ok  	github.com/shitcoding/tmux_yankee/internal/config	0.43s")
	lines = append(lines, "ok  	github.com/shitcoding/tmux_yankee/internal/input	0.35s")
	lines = append(lines, "ok  	github.com/shitcoding/tmux_yankee/internal/linenums	0.34s")
	lines = append(lines, "ok  	github.com/shitcoding/tmux_yankee/internal/theme	0.33s")
	lines = append(lines, "ok  	github.com/shitcoding/tmux_yankee/internal/ui	0.67s")
	lines = append(lines, "")

	// Final prompt
	lines = append(lines, bold(green("user@host"))+":"+bold(blue("~/projects/tmux-yankee"))+"$ █")

	// Pad to ~80 lines
	for len(lines) < 80 {
		lines = append(lines, "")
	}

	return lines
}

// DemoCodeSnippet returns ~60 lines of Go code with syntax highlighting.
func DemoCodeSnippet() []string {
	kw := func(s string) string { return blue(s) }        // keywords
	str := func(s string) string { return green(s) }      // strings
	cmt := func(s string) string { return gray(s) }       // comments
	typ := func(s string) string { return yellow(s) }     // types
	fn := func(s string) string { return cyan(s) }        // function names
	num := func(s string) string { return magenta(s) }    // numbers

	var lines []string

	lines = append(lines, kw("package")+" main")
	lines = append(lines, "")
	lines = append(lines, kw("import")+" (")
	lines = append(lines, "	"+str("\"fmt\""))
	lines = append(lines, "	"+str("\"strings\""))
	lines = append(lines, "	"+str("\"unicode/utf8\""))
	lines = append(lines, ")")
	lines = append(lines, "")
	lines = append(lines, cmt("// TextStyle holds per-element text decoration flags."))
	lines = append(lines, kw("type")+" "+typ("TextStyle")+" "+kw("struct")+" {")
	lines = append(lines, "	"+typ("Bold")+"      "+typ("bool"))
	lines = append(lines, "	"+typ("Dim")+"       "+typ("bool"))
	lines = append(lines, "	"+typ("Italic")+"    "+typ("bool"))
	lines = append(lines, "	"+typ("Underline")+" "+typ("bool"))
	lines = append(lines, "}")
	lines = append(lines, "")
	lines = append(lines, cmt("// styleCodes returns SGR escape codes for the active styles."))
	lines = append(lines, kw("func")+" "+fn("styleCodes")+"(s "+typ("TextStyle")+") []"+typ("string")+" {")
	lines = append(lines, "	"+kw("var")+" codes []"+typ("string"))
	lines = append(lines, "	"+kw("if")+" s.Bold {")
	lines = append(lines, "		codes = "+fn("append")+"(codes, "+str("\"1\"")+")")
	lines = append(lines, "	}")
	lines = append(lines, "	"+kw("if")+" s.Dim {")
	lines = append(lines, "		codes = "+fn("append")+"(codes, "+str("\"2\"")+")")
	lines = append(lines, "	}")
	lines = append(lines, "	"+kw("if")+" s.Italic {")
	lines = append(lines, "		codes = "+fn("append")+"(codes, "+str("\"3\"")+")")
	lines = append(lines, "	}")
	lines = append(lines, "	"+kw("if")+" s.Underline {")
	lines = append(lines, "		codes = "+fn("append")+"(codes, "+str("\"4\"")+")")
	lines = append(lines, "	}")
	lines = append(lines, "	"+kw("return")+" codes")
	lines = append(lines, "}")
	lines = append(lines, "")
	lines = append(lines, cmt("// renderGutter renders a formatted gutter line with styled number."))
	lines = append(lines, kw("func")+" "+fn("renderGutter")+"(lineNum, cursorLine "+typ("int")+", width "+typ("int")+") "+typ("string")+" {")
	lines = append(lines, "	num := "+fn("fmt.Sprintf")+"("+str("\"%*d\"")+", width, lineNum)")
	lines = append(lines, "	dist := "+fn("abs")+"(lineNum - cursorLine)")
	lines = append(lines, "	"+kw("if")+" dist == "+num("0")+" {")
	lines = append(lines, "		"+kw("return")+" "+fn("bold")+"(num) + "+str("\" │ \""))
	lines = append(lines, "	}")
	lines = append(lines, "	"+kw("return")+" num + "+str("\" │ \""))
	lines = append(lines, "}")
	lines = append(lines, "")
	lines = append(lines, kw("func")+" "+fn("main")+"() {")
	lines = append(lines, "	style := "+typ("TextStyle")+"{Bold: "+kw("true")+", Italic: "+kw("true")+"}")
	lines = append(lines, "	codes := "+fn("styleCodes")+"(style)")
	lines = append(lines, "	"+fn("fmt.Println")+"("+str("\"SGR codes:\"")+", "+fn("strings.Join")+"(codes, "+str("\";\"")+")"+")")
	lines = append(lines, "")
	lines = append(lines, "	"+cmt("// Verify rune-based width measurement"))
	lines = append(lines, "	sep := "+str("\"│\""))
	lines = append(lines, "	"+fn("fmt.Printf")+"("+str("\"bytes=%d runes=%d\\n\"")+", "+fn("len")+"(sep), "+fn("utf8.RuneCountInString")+"(sep))")
	lines = append(lines, "")
	lines = append(lines, "	"+kw("for")+" i := "+num("1")+"; i <= "+num("20")+"; i++ {")
	lines = append(lines, "		"+fn("fmt.Println")+"("+fn("renderGutter")+"(i, "+num("10")+", "+num("3")+"))")
	lines = append(lines, "	}")
	lines = append(lines, "}")
	lines = append(lines, "")

	// Pad to ~60 lines
	for len(lines) < 60 {
		lines = append(lines, "")
	}

	return lines
}

// DemoColorTestCard returns a diagnostic color test card.
func DemoColorTestCard() []string {
	var lines []string

	lines = append(lines, bold("  Color Test Card — ANSI / 256 / TrueColor"))
	lines = append(lines, "")

	// 16 basic ANSI colors
	lines = append(lines, bold("  Standard Colors (0-7):"))
	var row strings.Builder
	row.WriteString("  ")
	for i := 0; i < 8; i++ {
		row.WriteString(fmt.Sprintf("\x1b[48;5;%dm  %2d  \x1b[0m", i, i))
	}
	lines = append(lines, row.String())

	lines = append(lines, bold("  Bright Colors (8-15):"))
	row.Reset()
	row.WriteString("  ")
	for i := 8; i < 16; i++ {
		row.WriteString(fmt.Sprintf("\x1b[48;5;%dm  %2d  \x1b[0m", i, i))
	}
	lines = append(lines, row.String())
	lines = append(lines, "")

	// 256-color palette (6×6×6 cube)
	lines = append(lines, bold("  256-Color Cube (16-231):"))
	for g := 0; g < 6; g++ {
		row.Reset()
		row.WriteString("  ")
		for r := 0; r < 6; r++ {
			for b := 0; b < 6; b++ {
				idx := 16 + r*36 + g*6 + b
				row.WriteString(fmt.Sprintf("\x1b[48;5;%dm \x1b[0m", idx))
			}
			row.WriteString(" ")
		}
		lines = append(lines, row.String())
	}
	lines = append(lines, "")

	// Grayscale ramp
	lines = append(lines, bold("  Grayscale (232-255):"))
	row.Reset()
	row.WriteString("  ")
	for i := 232; i <= 255; i++ {
		row.WriteString(fmt.Sprintf("\x1b[48;5;%dm \x1b[0m", i))
	}
	lines = append(lines, row.String())
	lines = append(lines, "")

	// TrueColor gradient
	lines = append(lines, bold("  TrueColor Gradient (24-bit):"))
	row.Reset()
	row.WriteString("  ")
	for i := 0; i < 60; i++ {
		r := int(float64(i) / 60.0 * 255.0)
		g := int(float64(60-i) / 60.0 * 255.0)
		b := 128
		row.WriteString(fmt.Sprintf("\x1b[48;2;%d;%d;%dm \x1b[0m", r, g, b))
	}
	lines = append(lines, row.String())
	row.Reset()
	row.WriteString("  ")
	for i := 0; i < 60; i++ {
		r := 128
		g := int(float64(i) / 60.0 * 255.0)
		b := int(float64(60-i) / 60.0 * 255.0)
		row.WriteString(fmt.Sprintf("\x1b[48;2;%d;%d;%dm \x1b[0m", r, g, b))
	}
	lines = append(lines, row.String())
	lines = append(lines, "")

	// Text decoration samples
	lines = append(lines, bold("  Text Decorations:"))
	lines = append(lines, "  Normal    "+bold("Bold")+"    "+dim("Dim")+"    "+italic("Italic")+"    "+uline("Underline"))
	lines = append(lines, "  "+bold(italic("Bold+Italic"))+"    "+dim(uline("Dim+Underline"))+"    "+bold(uline("Bold+Underline")))
	lines = append(lines, "")

	// FG color samples
	lines = append(lines, bold("  Foreground Colors:"))
	lines = append(lines, "  "+red("Red")+"  "+green("Green")+"  "+yellow("Yellow")+"  "+blue("Blue")+"  "+magenta("Magenta")+"  "+cyan("Cyan")+"  "+white("White")+"  "+gray("Gray"))
	lines = append(lines, "  "+bold(red("Bold Red"))+"  "+bold(green("Bold Green"))+"  "+bold(blue("Bold Blue"))+"  "+bold(magenta("Bold Magenta")))
	lines = append(lines, "")

	// BG color samples
	lines = append(lines, bold("  Background Colors:"))
	lines = append(lines, "  "+bgRed(" Red BG ")+"  "+bgGreen(" Green BG ")+"  "+bgBlue(" Blue BG "))
	lines = append(lines, "")

	// 256-color foreground samples
	lines = append(lines, bold("  256-Color Foreground Samples:"))
	row.Reset()
	row.WriteString("  ")
	for i := 0; i < 16; i++ {
		row.WriteString(fg256(i*16+8, fmt.Sprintf("%3d ", i*16+8)))
	}
	lines = append(lines, row.String())
	lines = append(lines, "")

	// TrueColor foreground
	lines = append(lines, bold("  TrueColor Foreground:"))
	row.Reset()
	row.WriteString("  ")
	for i := 0; i < 40; i++ {
		r := int(float64(i) / 40.0 * 255.0)
		row.WriteString(fgRGB(r, 100, 255-r, "█"))
	}
	lines = append(lines, row.String())
	lines = append(lines, "")

	return lines
}

// DemoLongLines returns content with very long lines, wide Unicode, and tabs.
func DemoLongLines() []string {
	var lines []string

	lines = append(lines, bold("  Long Lines — Horizontal Scroll / Wrap Test"))
	lines = append(lines, "")

	// Long ASCII line
	lines = append(lines, "  "+strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 ", 3))

	// Long colored line
	longColored := "  "
	for i := 0; i < 30; i++ {
		longColored += fg256(i+1, fmt.Sprintf("color%02d ", i))
	}
	lines = append(lines, longColored)

	// Tab-heavy line
	lines = append(lines, "  col1\tcol2\tcol3\tcol4\tcol5\tcol6\tcol7\tcol8\tcol9\tcol10\tcol11\tcol12")

	// Wide Unicode characters (CJK)
	lines = append(lines, "  漢字テスト：吾輩は猫である。名前はまだ無い。どこで生れたかとんと見当がつかぬ。何でも薄暗いじめじめした所でニャーニャー泣いていた事だけは記憶している。")

	// Emoji line
	lines = append(lines, "  🎨 Theme preview 🖥️  Terminal colors 🔧 Configuration ✅ Tests passing 🚀 Release ready 📦 Package built 🎯 Targeting tmux 3.2+ 💡 Tip: use Tab to cycle pages")

	// Mixed width + ANSI
	lines = append(lines, "  "+red("ERROR")+" at "+cyan("main.go")+":42 — "+yellow("variable")+` "cfg" declared but not used; did you mean `+green("`config`")+`? Full path: ~/projects/tmux-yankee/cmd/tmux-yankee/main.go:42:6`)

	lines = append(lines, "")

	// Path-like long lines
	lines = append(lines, "  PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/home/user/.cargo/bin:/home/user/go/bin:/opt/homebrew/bin:/opt/homebrew/sbin")
	lines = append(lines, "  GOPATH=/home/user/go GOROOT=/usr/local/go GOBIN=/home/user/go/bin GOPROXY=https://proxy.golang.org,direct GONOSUMCHECK=off GOFLAGS=-mod=vendor")

	lines = append(lines, "")

	// Very long JSON-like line
	lines = append(lines, `  {"id":"a1b2c3d4","timestamp":"2026-02-19T10:30:00Z","level":"info","message":"Theme resolved successfully","theme":"dracula","overrides":{"cursor_fg":"#ff79c6","cursor_bg":"#282a36","selection_fg":"#f8f8f2","selection_bg":"#44475a","gutter_fg":"#6272a4","linenum_cursor_bold":"on"},"duration_ms":0.42}`)

	lines = append(lines, "")

	// Repeated pattern for scroll testing
	for i := 1; i <= 30; i++ {
		pad := strings.Repeat("=", i*5)
		lines = append(lines, fmt.Sprintf("  Line %3d: %s [end]", i, pad))
	}

	lines = append(lines, "")

	// Box-drawing characters (narrow but multi-byte)
	lines = append(lines, "  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐")
	lines = append(lines, "  │ Box drawing: ─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼   Double: ═ ║ ╔ ╗ ╚ ╝ ╠ ╣ ╦ ╩ ╬   Heavy: ━ ┃ ┏ ┓ ┗ ┛ ┣ ┫ ┳ ┻ ╋ │")
	lines = append(lines, "  └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘")

	return lines
}
