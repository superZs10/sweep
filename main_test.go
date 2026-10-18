package main

import "testing"

func TestParseArgsHelpWithoutCommand(t *testing.T) {
	o, err := parseArgs([]string{"--help"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.help {
		t.Error("`sweep --help` 应该设置 help")
	}
	if o.cmd != "" {
		t.Errorf("不应把 --help 当成子命令，得到 %q", o.cmd)
	}
}

func TestParseArgsVersionWithoutCommand(t *testing.T) {
	o, err := parseArgs([]string{"-v"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.showVer {
		t.Error("`sweep -v` 应该设置 showVer")
	}
}

func TestParseArgsClean(t *testing.T) {
	o, err := parseArgs([]string{"clean", "temp", "cache", "-y", "--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	if o.cmd != "clean" {
		t.Errorf("cmd = %q, want clean", o.cmd)
	}
	if len(o.categories) != 2 || o.categories[0] != "temp" || o.categories[1] != "cache" {
		t.Errorf("categories = %v", o.categories)
	}
	if !o.yes || !o.dryRun {
		t.Errorf("yes=%v dryRun=%v", o.yes, o.dryRun)
	}
}

func TestParseArgsOptions(t *testing.T) {
	o, err := parseArgs([]string{
		"scan", "-r", "temp.user,cache.gpu", "--category=cache",
		"--min-age", "30", "-p", `D:\tmp`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(o.ruleIDs) != 2 {
		t.Errorf("ruleIDs = %v", o.ruleIDs)
	}
	if len(o.categories) != 1 || o.categories[0] != "cache" {
		t.Errorf("categories = %v", o.categories)
	}
	if o.minAgeDays != 30 {
		t.Errorf("minAgeDays = %d", o.minAgeDays)
	}
	if len(o.extraPaths) != 1 || o.extraPaths[0] != `D:\tmp` {
		t.Errorf("extraPaths = %v", o.extraPaths)
	}
}

func TestParseArgsUnknownFlag(t *testing.T) {
	if _, err := parseArgs([]string{"scan", "--nope"}); err == nil {
		t.Error("未知选项应返回错误")
	}
}

func TestParseArgsMissingValue(t *testing.T) {
	if _, err := parseArgs([]string{"clean", "--min-age"}); err == nil {
		t.Error("缺少参数值应返回错误")
	}
}
