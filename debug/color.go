package debug

// License Note: codes copied from github.com/gookit/color

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)


const (
	//SettingTpl   = "\x1b[%sm"
	FullColorTpl = "\x1b[%sm%s\x1b[0m"

	CodeExpr = `\033\[[\d;?]+m`
)

var (
	// match color codes
	codeRegex = regexp.MustCompile(CodeExpr)

	isSupportColor = IsSupportColor()
)


// IsSupportColor check current console is support color.
//
// Supported:
// 	linux, mac, or windows's ConEmu, Cmder, putty, git-bash.exe
// Not support:
// 	windows cmd.exe, powerShell.exe
func IsSupportColor() bool {
	// "TERM=xterm"  support color
	// "TERM=xterm-vt220" support color
	// "TERM=xterm-256color" support color
	// "TERM=cygwin" don't support color
	if strings.Contains(os.Getenv("TERM"), "xterm") {
		return true
	}

	// like on ConEmu software, e.g "ConEmuANSI=ON"
	if os.Getenv("ConEmuANSI") == "ON" {
		return true
	}

	// like on ConEmu software, e.g "ANSICON=189x2000 (189x43)"
	if os.Getenv("ANSICON") != "" {
		return true
	}

	return false
}




// RenderCode render message by color code.
// Usage:
// 	msg := RenderCode("3;32;45", "some", "message")
func RenderCode(code string, args ...interface{}) string {
	message := fmt.Sprint(args...)
	if len(code) == 0 {
		return message
	}

	if !isSupportColor {
		return ClearCode(message)
	}

	return fmt.Sprintf(FullColorTpl, code, message)
}

// RenderString render a string with color code.
// Usage:
// 	msg := RenderString("3;32;45", "a message")
func RenderString(code string, str string) string {
	// some check
	if len(code) == 0 || str == "" {
		return str
	}

	if !isSupportColor {
		return ClearCode(str)
	}

	return fmt.Sprintf(FullColorTpl, code, str)
}


// ClearCode clear color codes.
// eg:

func ClearCode(str string) string {
	return codeRegex.ReplaceAllString(str, "")
}



// convert colors to code. return like "32;45;3"
func colors2code(colors ...Color) string {
	if len(colors) == 0 {
		return ""
	}

	var codes []string
	for _, color := range colors {
		codes = append(codes, color.String())
	}

	return strings.Join(codes, ";")
}





// Color Color16, 16 color value type
// 3(2^3=8) OR 4(2^4=16) bite color.
type Color uint8

/*************************************************************
 * Basic 16 color definition
 *************************************************************/

// Foreground colors. basic foreground colors 30 - 37
const (
	FgBlack Color = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta  // 品红
	FgCyan     // 青色
	FgWhite
	// FgDefault revert default FG
	FgDefault Color = 39
)

// Extra foreground color 90 - 97(非标准)
const (
	FgDarkGray Color = iota + 90 // 亮黑（灰）
	FgLightRed
	FgLightGreen
	FgLightYellow
	FgLightBlue
	FgLightMagenta
	FgLightCyan
	FgLightWhite
	// FgGray is alias of FgDarkGray
	FgGray Color = 90 // 亮黑（灰）
)

var FgColors = []Color{
	FgRed, FgGreen, FgYellow, FgBlue, FgMagenta, FgCyan,
	//FgLightRed, FgLightGreen, FgLightYellow, FgLightBlue, FgLightMagenta, FgLightCyan,
	FgLightYellow, FgLightMagenta, FgLightRed, FgLightCyan, FgLightGreen, FgLightBlue,
}

// Background colors. basic background colors 40 - 47
const (
	BgBlack Color = iota + 40
	BgRed
	BgGreen
	BgYellow  // BgBrown like yellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
	// BgDefault revert default BG
	BgDefault Color = 49
)

// Extra background color 100 - 107(非标准)
const (
	BgDarkGray Color = iota + 99
	BgLightRed
	BgLightGreen
	BgLightYellow
	BgLightBlue
	BgLightMagenta
	BgLightCyan
	BgLightWhite
	// BgGray is alias of BgDarkGray
	BgGray Color = 100
)

// Option settings
const (
	OpReset         Color = iota // 0 重置所有设置
	OpBold                       // 1 加粗
	OpFuzzy                      // 2 模糊(不是所有的终端仿真器都支持)
	OpItalic                     // 3 斜体(不是所有的终端仿真器都支持)
	OpUnderscore                 // 4 下划线
	OpBlink                      // 5 闪烁
	OpFastBlink                  // 5 快速闪烁(未广泛支持)
	OpReverse                    // 7 颠倒的 交换背景色与前景色
	OpConcealed                  // 8 隐匿的
	OpStrikethrough              // 9 删除的，删除线(未广泛支持)
)

// There are basic and light foreground color aliases
const (
	Red     = FgRed
	Cyan    = FgCyan
	Gray    = FgDarkGray // is light Black
	Blue    = FgBlue
	Black   = FgBlack
	Green   = FgGreen
	White   = FgWhite
	Yellow  = FgYellow
	Magenta = FgMagenta
	// special
	Bold   = OpBold
	Normal = FgDefault
	// extra light
	LightRed     = FgLightRed
	LightCyan    = FgLightCyan
	LightBlue    = FgLightBlue
	LightGreen   = FgLightGreen
	LightWhite   = FgLightWhite
	LightYellow  = FgLightYellow
	LightMagenta = FgLightMagenta
)


// Render messages by color setting
// Usage:
// 		green := color.FgGreen.Render
// 		fmt.Println(green("message"))
func (c Color) Render(a ...interface{}) string {
	return RenderCode(c.String(), a...)
}

// String to code string. eg "35"
func (c Color) String() string {
	return fmt.Sprintf("%d", c)
}

// Sprint render messages by color setting. is alias of the Render()
func (c Color) Sprint(a ...interface{}) string {
	return RenderCode(c.String(), a...)
}

// Sprintf format and render message.
// Usage:
// 	green := color.Green.Sprintf
//  colored := green("message")
func (c Color) Sprintf(format string, args ...interface{}) string {
	return RenderString(c.String(), fmt.Sprintf(format, args...))
}

// Light current color. eg: 36(FgCyan) -> 96(FgLightCyan).
// Usage:
// 	lightCyan := Cyan.Light()
// 	lightCyan.Print("message")
func (c Color) Light() Color {
	val := int(c)
	if val >= 30 && val <= 47 {
		return Color(uint8(c) + 60)
	}

	// don't change
	return c
}

// Darken current color. eg. 96(FgLightCyan) -> 36(FgCyan)
// Usage:
// 	cyan := LightCyan.Darken()
// 	cyan.Print("message")
func (c Color) Darken() Color {
	val := int(c)
	if val >= 90 && val <= 107 {
		return Color(uint8(c) - 60)
	}

	// don't change
	return c
}





// Style a 16 color style
// can add: fg color, bg color, color options
// Example:
// 	color.Style(color.FgGreen).Print("message")
type Style []Color


// Render render text
// Usage:
//  color.New(color.FgGreen).Render("text")
//  color.New(color.FgGreen, color.BgBlack, color.OpBold).Render("text")
func (s Style) Render(a ...interface{}) string {
	return RenderCode(s.String(), a...)
}

// Sprint is alias of the 'Render'
func (s Style) Sprint(a ...interface{}) string {
	return RenderCode(s.String(), a...)
}

// Sprintf is alias of the 'Render'
func (s Style) Sprintf(format string, a ...interface{}) string {
	return RenderString(s.String(), fmt.Sprintf(format, a...))
}

// String convert to code string. returns like "32;45;3"
func (s Style) String() string {
	return colors2code(s...)
}

