// Command sweep 是一个面向 Windows 的磁盘清理命令行工具。
//
// 它把"电脑里的垃圾"拆成一条条可解释的规则：先只读扫描告诉你
// 每类能清出多少空间，再由你确认后删除。
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/superZs10/sweep/internal/clean"
	"github.com/superZs10/sweep/internal/rules"
	"github.com/superZs10/sweep/internal/scan"
	"github.com/superZs10/sweep/internal/size"
	"github.com/superZs10/sweep/internal/term"
)

// version 由 -ldflags 注入，源码里保留开发版本号。
var version = "0.1.0-dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

// options 保存解析后的命令行参数。
type options struct {
	cmd        string
	categories []string
	ruleIDs    []string
	extraPaths []string
	asJSON     bool
	dryRun     bool
	yes        bool
	force      bool
	debug      bool
	minAgeDays int
	help       bool
	showVer    bool
}

func run(argv []string) int {
	o, err := parseArgs(argv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n\n", err)
		printUsage(os.Stderr)
		return 2
	}
	if o.showVer {
		fmt.Printf("sweep %s\n", version)
		return 0
	}
	if o.help || o.cmd == "" {
		printUsage(os.Stdout)
		return 0
	}

	switch o.cmd {
	case "scan":
		return cmdScan(o)
	case "clean":
		return cmdClean(o)
	case "list", "rules":
		return cmdList()
	case "version":
		fmt.Printf("sweep %s\n", version)
		return 0
	case "help":
		printUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "未知命令 %q\n\n", o.cmd)
		printUsage(os.Stderr)
		return 2
	}
}

func parseArgs(argv []string) (*options, error) {
	o := &options{}
	if len(argv) == 0 {
		return o, nil
	}
	o.cmd = argv[0]

	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		value := ""
		hasValue := false
		if strings.HasPrefix(arg, "--") {
			if k, v, ok := strings.Cut(arg, "="); ok {
				arg, value, hasValue = k, v, true
			}
		}

		take := func() (string, error) {
			if hasValue {
				return value, nil
			}
			if i+1 < len(argv) {
				i++
				return argv[i], nil
			}
			return "", fmt.Errorf("选项 %s 缺少参数", arg)
		}

		switch arg {
		case "-h", "--help":
			o.help = true
		case "-v", "--version":
			o.showVer = true
		case "--json":
			o.asJSON = true
		case "--dry-run", "-n":
			o.dryRun = true
		case "-y", "--yes":
			o.yes = true
		case "-f", "--force", "--all":
			o.force = true
		case "--debug":
			o.debug = true
		case "-c", "--category":
			v, err := take()
			if err != nil {
				return nil, err
			}
			o.categories = append(o.categories, splitList(v)...)
		case "-r", "--rule":
			v, err := take()
			if err != nil {
				return nil, err
			}
			o.ruleIDs = append(o.ruleIDs, splitList(v)...)
		case "-p", "--path":
			v, err := take()
			if err != nil {
				return nil, err
			}
			o.extraPaths = append(o.extraPaths, v)
		case "--min-age":
			v, err := take()
			if err != nil {
				return nil, err
			}
			n, convErr := strconv.Atoi(v)
			if convErr != nil || n < 0 {
				return nil, fmt.Errorf("--min-age 需要非负整数（单位：天）")
			}
			o.minAgeDays = n
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("未知选项 %s", arg)
			}
			// 位置参数当作类别名，例如 `sweep clean temp cache`
			o.categories = append(o.categories, arg)
		}
	}
	return o, nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// collect 扫描所有被选中的规则。
//
// 只给了 -p 而没有指定类别/规则时，含义是"只处理这些目录"，
// 避免用户以为在清临时文件、结果把浏览器缓存也一起删了。
func collect(o *options) []scan.Item {
	onlyExtra := len(o.extraPaths) > 0 && len(o.categories) == 0 && len(o.ruleIDs) == 0

	var items []scan.Item
	for _, r := range rules.All() {
		if onlyExtra {
			break
		}
		if !matchRule(r, o) {
			continue
		}
		if o.minAgeDays > 0 && r.Action == rules.DeleteFiles {
			r.MinAge = time.Duration(o.minAgeDays) * 24 * time.Hour
		}
		item, err := scan.Run(r)
		if err != nil || item.Empty() {
			continue
		}
		items = append(items, item)
	}

	for i, p := range o.extraPaths {
		r := rules.Rule{
			ID:       fmt.Sprintf("extra.%d", i+1),
			Name:     "自定义目录",
			Category: "temp",
			Risk:     rules.Caution,
			Action:   rules.DeleteContents,
			Paths:    []string{p},
			Note:     "由 --path 指定",
		}
		item, err := scan.Run(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "扫描 %s 失败: %v\n", p, err)
			continue
		}
		if item.Empty() {
			continue
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].Bytes > items[j].Bytes })
	return items
}

func matchRule(r rules.Rule, o *options) bool {
	if len(o.ruleIDs) > 0 {
		found := false
		for _, id := range o.ruleIDs {
			if strings.EqualFold(id, r.ID) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(o.categories) > 0 {
		found := false
		for _, c := range o.categories {
			if strings.EqualFold(c, r.Category) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func cmdScan(o *options) int {
	start := time.Now()
	items := collect(o)
	if o.asJSON {
		return printJSON(items)
	}
	if len(items) == 0 {
		fmt.Println("没有发现可清理的内容。")
		return 0
	}
	fmt.Printf("\n扫描完成，用时 %s\n\n", time.Since(start).Round(time.Millisecond))
	renderTable(items)
	return 0
}

func cmdList() int {
	t := term.Table{
		Headers: []string{"规则", "名称", "类别", "风险", "说明"},
		Indent:  2,
	}
	for _, r := range rules.All() {
		t.Rows = append(t.Rows, []term.Cell{
			term.Text(r.ID),
			term.Text(r.Name),
			term.Text(r.Category),
			term.Colored(r.Risk.String(), r.Risk.Color()),
			term.Text(r.Note),
		})
	}
	fmt.Println()
	fmt.Print(t.Render())
	fmt.Println()
	return 0
}

func cmdClean(o *options) int {
	start := time.Now()
	items := collect(o)
	if len(items) == 0 {
		fmt.Println("没有发现可清理的内容。")
		return 0
	}
	fmt.Printf("\n扫描完成，用时 %s\n\n", time.Since(start).Round(time.Millisecond))
	renderTable(items)

	var runnable, blocked []scan.Item
	for _, it := range items {
		if it.Rule.Risk == rules.Risky && !o.force {
			blocked = append(blocked, it)
			continue
		}
		runnable = append(runnable, it)
	}

	if len(blocked) > 0 {
		var b int64
		for _, it := range blocked {
			b += it.Bytes
		}
		fmt.Printf("\n%s 已跳过 %d 条高风险规则（共 %s）。确认无误请加 --force。\n",
			term.Paint("提示:", term.Yellow), len(blocked), size.Human(b))
	}
	if len(runnable) == 0 {
		return 0
	}

	var total int64
	for _, it := range runnable {
		total += it.Bytes
	}

	if o.dryRun {
		fmt.Printf("\n%s 将释放 %s，未做任何修改。\n",
			term.Paint("[dry-run]", term.Cyan), size.Human(total))
		return 0
	}

	if !o.yes {
		fmt.Printf("\n即将清理 %d 项，预计释放 %s。\n", len(runnable), size.Human(total))
		fmt.Print("确认执行？[y/N] ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		ans := strings.ToLower(strings.TrimSpace(line))
		if ans != "y" && ans != "yes" {
			fmt.Println("已取消，未删除任何内容。")
			return 1
		}
	}

	var freed int64
	var deleted, skipped, denied int
	var failures []string
	var debugLines []string

	fmt.Println()
	for _, it := range runnable {
		res := clean.Apply(it, clean.Options{})
		freed += res.Freed
		deleted += res.Deleted
		skipped += res.Skipped
		denied += res.Denied
		if res.Err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", it.Rule.ID, res.Err))
		}
		debugLines = append(debugLines, res.Debug...)
		fmt.Printf("  %-20s %10s  %d 个文件\n", it.Rule.ID, size.Human(res.Freed), res.Deleted)
	}

	fmt.Println()
	fmt.Println(term.Paint(strings.Repeat("─", 46), term.Dim))
	fmt.Printf("完成：释放 %s，删除 %d 个文件\n", term.Paint(size.Human(freed), term.Green+";"+term.Bold), deleted)
	if skipped > 0 {
		fmt.Printf("跳过 %d 项（被占用或权限不足）\n", skipped)
	}
	if denied > 0 {
		fmt.Printf("%s 其中 %d 处需要管理员权限，建议以管理员身份重新运行。\n",
			term.Paint("注意:", term.Yellow), denied)
	}
	for _, f := range failures {
		fmt.Printf("%s %s\n", term.Paint("失败:", term.Red), f)
	}
	if o.debug && len(debugLines) > 0 {
		fmt.Println()
		fmt.Println(term.Paint("调试信息:", term.Dim))
		for _, d := range debugLines {
			fmt.Println("  " + term.Paint(d, term.Dim))
		}
	}

	return 0
}

func renderTable(items []scan.Item) {
	t := term.Table{
		Headers: []string{"规则", "名称", "类别", "大小", "文件", "风险"},
		Right:   []bool{false, false, false, true, true, false},
		Indent:  2,
	}
	var total int64
	var files, denied int
	var needAdmin int

	for _, it := range items {
		total += it.Bytes
		files += it.Files
		denied += it.Denied
		if it.Rule.NeedsAdmin {
			needAdmin++
		}
		t.Rows = append(t.Rows, []term.Cell{
			term.Text(it.Rule.ID),
			term.Text(it.Rule.Name),
			term.Text(it.Rule.Category),
			term.Text(size.Human(it.Bytes)),
			term.Text(strconv.Itoa(it.Files)),
			term.Colored(it.Rule.Risk.String(), it.Rule.Risk.Color()),
		})
	}
	fmt.Print(t.Render())

	// 合计行要按显示宽度补空格，不能用 %-40s（它按字符数算）
	w := t.Widths()
	prefix := w[0] + 2 + w[1] + 2 + w[2]
	totalLine := strings.Repeat(" ", t.Indent) +
		term.Paint(term.Pad("合计", prefix), term.Bold) + "  " +
		term.Paint(term.PadLeft(size.Human(total), w[3]), term.Bold) + "  " +
		term.Paint(term.PadLeft(strconv.Itoa(files), w[4]), term.Bold)
	fmt.Println(strings.TrimRight(totalLine, " "))
	if needAdmin > 0 {
		fmt.Printf("  其中 %d 项通常需要管理员权限才能完整清理\n", needAdmin)
	}
	if denied > 0 {
		fmt.Printf("  有 %d 处路径因权限不足未能统计\n", denied)
	}
	fmt.Println()
}

type jsonItem struct {
	Rule       string   `json:"rule"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Risk       string   `json:"risk"`
	Bytes      int64    `json:"bytes"`
	Human      string   `json:"human"`
	Files      int      `json:"files"`
	Paths      []string `json:"paths"`
	NeedsAdmin bool     `json:"needs_admin"`
	Note       string   `json:"note,omitempty"`
}

func printJSON(items []scan.Item) int {
	out := make([]jsonItem, 0, len(items))
	var total int64
	for _, it := range items {
		paths := make([]string, 0, len(it.Roots))
		for _, r := range it.Roots {
			paths = append(paths, r.Path)
		}
		total += it.Bytes
		out = append(out, jsonItem{
			Rule:       it.Rule.ID,
			Name:       it.Rule.Name,
			Category:   it.Rule.Category,
			Risk:       it.Rule.Risk.String(),
			Bytes:      it.Bytes,
			Human:      size.Human(it.Bytes),
			Files:      it.Files,
			Paths:      paths,
			NeedsAdmin: it.Rule.NeedsAdmin,
			Note:       it.Rule.Note,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(struct {
		Total int64      `json:"total"`
		Human string     `json:"total_human"`
		Items []jsonItem `json:"items"`
	}{total, size.Human(total), out}); err != nil {
		fmt.Fprintf(os.Stderr, "输出 JSON 失败: %v\n", err)
		return 1
	}
	return 0
}

func printUsage(w *os.File) {
	fmt.Fprint(w, `sweep — Windows 磁盘清理工具

用法:
  sweep scan  [选项]        扫描并列出可清理项（只读，不删除）
  sweep clean [选项]        扫描后确认清理
  sweep list                列出所有内置规则
  sweep version             显示版本

选项:
  -c, --category <名称>     只处理指定类别，可重复或用逗号分隔
                          类别: temp cache system installers dev
  -r, --rule <id>           只处理指定规则（见 sweep list）
  -p, --path <目录>         额外扫描一个目录
      --min-age <天数>      覆盖"旧文件"的时间阈值
      --json                以 JSON 输出扫描结果（仅 scan）
  -n, --dry-run             只显示将要清理的内容，不实际删除
  -y, --yes                 跳过确认提示
  -f, --force               允许执行高风险规则
  -h, --help                显示帮助
  -v, --version             显示版本

示例:
  sweep scan
  sweep scan --json | jq '.items[] | select(.bytes > 1e9)'
  sweep clean temp cache -y
  sweep clean --category system --dry-run
  sweep clean -p D:\temp --force
`)
}
