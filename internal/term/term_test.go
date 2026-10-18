package term

import (
	"strings"
	"testing"
)

func TestRuneWidth(t *testing.T) {
	cases := []struct {
		r    rune
		want int
	}{
		{'a', 1},
		{'中', 2},
		{'文', 2},
		{'Ａ', 2}, // 全角 A
		{'，', 2}, // 全角逗号
		{' ', 1},
		{'\t', 0},
		{0, 0},
	}
	for _, c := range cases {
		if got := RuneWidth(c.r); got != c.want {
			t.Errorf("RuneWidth(%q) = %d, want %d", c.r, got, c.want)
		}
	}
}

func TestWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"abc", 3},
		{"中文", 4},
		{"a中b", 4},
		{"", 0},
	}
	for _, c := range cases {
		if got := Width(c.in); got != c.want {
			t.Errorf("Width(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestTableAlignment 验证中英文混排时每一列都真正对齐。
//
// 做法：给每个单元格一个不含空格的唯一 token，渲染后按 token 反查
// 它在行内的显示起点，同名列在不同行必须落在同一个显示宽度上。
func TestTableAlignment(t *testing.T) {
	Color = false

	rows := [][]Cell{
		{Text("cache.gpu"), Text("显卡着色器缓存"), Text("5.91GB")},
		{Text("temp.user"), Text("临时"), Text("1.72GB")},
		{Text("a"), Text("x"), Text("0B")},
	}
	tbl := Table{
		Headers: []string{"rule", "名称", "size"},
		Right:   []bool{false, false, true},
		Rows:    rows,
	}
	out := tbl.Render()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("unexpected line count %d:\n%s", len(lines), out)
	}

	startOf := func(line, token string) (int, bool) {
		idx := strings.Index(line, token)
		if idx < 0 {
			return 0, false
		}
		return Width(line[:idx]), true
	}

	header := []string{"rule", "名称", "size"}
	headerStart := make([]int, len(header))
	for ci, tok := range header {
		w, ok := startOf(lines[0], tok)
		if !ok {
			t.Fatalf("表头找不到 %q", tok)
		}
		headerStart[ci] = w
	}

	for li := 2; li < len(lines); li++ {
		row := rows[li-2]
		for ci, cell := range row {
			got, ok := startOf(lines[li], cell.Text)
			if !ok {
				t.Fatalf("line %d 找不到 token %q: %q", li, cell.Text, lines[li])
			}
			if ci == len(row)-1 {
				// 末列右对齐：只要求右边界与表头一致
				wantEnd := headerStart[ci] + Width(header[ci])
				if got+Width(cell.Text) != wantEnd {
					t.Errorf("line %d 末列右边界 %d, want %d\n%s", li, got+Width(cell.Text), wantEnd, out)
				}
				continue
			}
			if got != headerStart[ci] {
				t.Errorf("line %d column %d 起点 %d, want %d\n%s", li, ci, got, headerStart[ci], out)
			}
		}
	}

	// 分隔线宽度必须等于表格实际宽度
	if w, h := Width(strings.TrimSpace(lines[1])), Width(strings.TrimRight(lines[0], " ")); w != h {
		t.Errorf("分隔线宽度 %d, 表头宽度 %d", w, h)
	}
}

func TestTableWithInternalSpaces(t *testing.T) {
	Color = false
	tbl := Table{
		Headers: []string{"a", "b"},
		Rows: [][]Cell{
			{Text("x"), Text("705 MB")},
			{Text("yy"), Text("1.72 GB")},
		},
	}
	out := tbl.Render()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("unexpected output:\n%s", out)
	}
	// 含空格的内容不应把列挤歪：两行第二列都以 "GB" 结尾
	for _, line := range lines[1:] {
		if strings.Contains(line, "MB") || strings.Contains(line, "GB") {
			if !strings.HasSuffix(strings.TrimRight(line, " "), "B") {
				t.Errorf("列被内容里的空格挤歪: %q", line)
			}
		}
	}
}

func TestPadAndPadLeft(t *testing.T) {
	if got := Pad("中", 5); Width(got) != 5 {
		t.Errorf("Pad width = %d, want 5", Width(got))
	}
	if got := PadLeft("中", 5); Width(got) != 5 {
		t.Errorf("PadLeft width = %d, want 5", Width(got))
	}
}

func TestPaintDisabled(t *testing.T) {
	Color = false
	if got := Paint("x", Red); got != "x" {
		t.Errorf("Paint with color disabled = %q", got)
	}
	Color = true
	if got := Paint("x", Red); !strings.Contains(got, "\x1b[31m") {
		t.Errorf("Paint with color enabled = %q", got)
	}
	Color = false
}
