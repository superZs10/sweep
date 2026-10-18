// Package term 负责终端输出：ANSI 颜色、中文宽度对齐、表格渲染。
//
// Windows 传统控制台不支持 ANSI 转义，因此颜色只在标准输出是终端
// 且未设置 NO_COLOR 时启用。
package term

import (
	"os"
	"strings"
)

// ANSI 颜色代码。
const (
	Reset   = "0"
	Bold    = "1"
	Dim     = "2"
	Red     = "31"
	Green   = "32"
	Yellow  = "33"
	Blue    = "34"
	Magenta = "35"
	Cyan    = "36"
	Gray    = "90"
)

// Color 表示是否应该输出 ANSI 颜色。
var Color = isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Paint 用 ANSI 代码包裹文本；颜色关闭时原样返回。
func Paint(s, code string) string {
	if !Color || code == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[" + Reset + "m"
}

// RuneWidth 返回一个字符在终端里占的列数（东亚宽字符算 2 列）。
func RuneWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 32 || (r >= 0x7f && r < 0xa0):
		return 0
	case r == 0x200b: // 零宽空格
		return 0
	case r >= 0x1100 && (r <= 0x115f ||
		r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1f64f) ||
		(r >= 0x1f900 && r <= 0x1f9ff) ||
		(r >= 0x20000 && r <= 0x3fffd)):
		return 2
	}
	return 1
}

// Width 返回字符串的显示宽度。
func Width(s string) int {
	w := 0
	for _, r := range s {
		w += RuneWidth(r)
	}
	return w
}

// Pad 在右侧补空格到指定宽度。
func Pad(s string, w int) string {
	if d := w - Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

// PadLeft 在左侧补空格到指定宽度。
func PadLeft(s string, w int) string {
	if d := w - Width(s); d > 0 {
		return strings.Repeat(" ", d) + s
	}
	return s
}

// Cell 是表格中的一个单元格，text 参与宽度计算，code 决定颜色。
type Cell struct {
	Text string
	Code string
}

// Text 构造一个无色单元格。
func Text(s string) Cell { return Cell{Text: s} }

// Colored 构造一个带颜色的单元格。
func Colored(s, code string) Cell { return Cell{Text: s, Code: code} }

// Table 是一个简单的等宽表格。
type Table struct {
	Headers []string
	Rows    [][]Cell
	// Right 标记哪些列右对齐（通常是数字列）。
	Right []bool
	// Indent 是每行前面的缩进空格数。
	Indent int
}

// Widths 计算每一列的实际显示宽度。
func (t Table) Widths() []int {
	cols := len(t.Headers)
	widths := make([]int, cols)
	for i, h := range t.Headers {
		widths[i] = Width(h)
	}
	for _, row := range t.Rows {
		for i := 0; i < cols && i < len(row); i++ {
			if w := Width(row[i].Text); w > widths[i] {
				widths[i] = w
			}
		}
	}
	return widths
}

// Render 把表格渲染成字符串。
func (t Table) Render() string {
	cols := len(t.Headers)
	widths := t.Widths()
	indent := strings.Repeat(" ", t.Indent)

	var lines []string

	line := func(cells []Cell, header bool) {
		var b strings.Builder
		b.WriteString(indent)
		for i, c := range cells {
			text := c.Text
			if t.Right != nil && i < len(t.Right) && t.Right[i] {
				text = PadLeft(text, widths[i])
			} else {
				text = Pad(text, widths[i])
			}
			code := c.Code
			if header {
				code = Bold
			}
			b.WriteString(Paint(text, code))
			if i < len(cells)-1 {
				b.WriteString("  ")
			}
		}
		// 去掉行尾多余空格，方便复制
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}

	header := make([]Cell, cols)
	for i, h := range t.Headers {
		header[i] = Cell{Text: h}
	}
	line(header, true)

	total := 0
	for _, w := range widths {
		total += w
	}
	total += 2 * (cols - 1)
	lines = append(lines, indent+Paint(strings.Repeat("─", total), Dim))

	for _, row := range t.Rows {
		line(row, false)
	}
	return strings.Join(lines, "\n") + "\n"
}
