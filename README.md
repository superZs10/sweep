# sweep

> 先看清楚，再删干净。

[![CI](https://github.com/superZs10/sweep/actions/workflows/ci.yml/badge.svg)](https://github.com/superZs10/sweep/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/superZs10/sweep?color=blue)](https://github.com/superZs10/sweep/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/superZs10/sweep)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Windows 上的磁盘清理工具通常走两个极端：`cleanmgr` 是一个黑箱进度条，你
不知道它到底删了什么；而"一键清理"工具又太容易顺手删掉你还想要的东西。

**sweep 把"垃圾"拆成一条条可读的规则。** 它先只读扫描，告诉你每一类占多少
空间、在哪个路径、风险多大；再由你决定删不删。

```console
$ sweep scan

扫描完成，用时 193ms

  规则                  名称              类别            大小  文件  风险
  ──────────────────────────────────────────────────────────────────────────
  cache.browser         浏览器缓存        cache         705 MB  7146  安全
  dev.pnpm              pnpm 缓存         dev           587 MB  8971  安全
  dev.pip               pip 缓存          dev         33.20 MB   134  安全
  temp.user             用户临时文件      temp        32.93 MB   159  安全
  cache.gpu             显卡着色器缓存    cache        6.90 MB   126  安全
  installers.downloads  下载目录旧安装包  installers   5.38 MB     1  高风险
  cache.thumbnails      缩略图缓存        cache        3.01 MB    15  安全
  system.recyclebin     回收站            system         387 B     3  谨慎
  合计                                                 1.34 GB  16555
  有 13 处路径因权限不足未能统计
```

## 特性

- **扫描是只读的。** `sweep scan` 只统计，不碰任何文件。
- **每条规则都能解释自己。** 路径、风险等级、一句人话说明，全部打印出来。
- **安全默认。** 高风险规则默认跳过（要 `--force`）；删除前二次确认；
  `--dry-run` 可以先看后删。
- **失败不中断。** 被占用的文件跳过并计数，权限不足的单独提示"请用管理员"。
- **零依赖单文件。** 编译出来一个 3 MB 的 exe，扔哪儿都能跑。
- **JSON 输出。** 方便接脚本、接定时任务、接自己的面板。

## 安装

### 下载预编译版本

到 [Releases](https://github.com/superZs10/sweep/releases) 下载对应平台的压缩包，
解压后把 `sweep.exe` 放到 `PATH` 里的任意目录。

### 用 go install

```bash
go install github.com/superZs10/sweep@latest
```

### 从源码编译

```bash
git clone https://github.com/superZs10/sweep.git
cd sweep
go build -trimpath -ldflags "-s -w" -o sweep.exe .
```

需要 Go 1.23 或更高版本。

## 使用

```bash
sweep scan                    # 只读扫描，看看能清多少
sweep scan --json             # 机器可读输出
sweep clean                   # 扫描 → 确认 → 清理
sweep clean temp cache -y     # 只清临时文件和缓存，跳过确认
sweep clean --dry-run         # 演练：只打印，不删除
sweep list                    # 列出所有规则
```

清理时的输出：

```console
$ sweep clean temp

即将清理 1 项，预计释放 32.93 MB。
确认执行？[y/N] y

  temp.user               32.93 MB  159 个文件

──────────────────────────────────────────────
完成：释放 32.93 MB，删除 159 个文件
```

### 常用选项

| 选项 | 说明 |
| --- | --- |
| `-c, --category <名称>` | 只处理指定类别：`temp` `cache` `system` `installers` `dev` |
| `-r, --rule <id>` | 只处理指定规则，ID 见 `sweep list` |
| `-p, --path <目录>` | 额外清理一个目录（当作自定义规则，按"谨慎"处理） |
| `--min-age <天数>` | 覆盖"旧文件"阈值，默认 90 天 |
| `--json` | 以 JSON 输出扫描结果 |
| `-n, --dry-run` | 只打印将要清理的内容 |
| `-y, --yes` | 跳过确认 |
| `-f, --force` | 允许执行高风险规则 |
| `--debug` | 清理后打印每个失败项的原始错误 |

> `-p` 单独使用时表示**只处理这些目录**，不会顺带跑内置规则；
> 和 `-c` / `-r` 一起用时才会叠加。

`scan` 和 `clean` 都可以只处理部分规则：

```bash
sweep scan cache                 # 只看缓存类
sweep clean dev -y               # 只清开发缓存
sweep clean --rule system.recyclebin --dry-run
```

## 内置规则

| 规则 ID | 名称 | 类别 | 风险 | 说明 |
| --- | --- | --- | --- | --- |
| `temp.user` | 用户临时文件 | temp | 安全 | `%TEMP%` 内容，程序会重新生成 |
| `temp.windows` | Windows 临时文件 | temp | 安全 | `%WINDIR%\Temp`，部分需管理员 |
| `cache.gpu` | 显卡着色器缓存 | cache | 安全 | NVIDIA / AMD / DirectX 的 shader 缓存 |
| `cache.thumbnails` | 缩略图缓存 | cache | 安全 | `thumbcache_*.db` |
| `cache.crash` | 崩溃转储 | cache | 安全 | `%LOCALAPPDATA%\CrashDumps` |
| `cache.browser` | 浏览器缓存 | cache | 安全 | Edge / Chrome 的 Cache、Code Cache、GPUCache |
| `system.recyclebin` | 回收站 | system | 谨慎 | 调用 `SHEmptyRecycleBinW` 清空 |
| `system.updatecache` | Windows 更新缓存 | system | 谨慎 | `SoftwareDistribution\Download`，需管理员 |
| `system.hibernate` | 休眠文件 | system | 高风险 | `powercfg /h off`，之后无法休眠 |
| `installers.downloads` | 下载目录旧安装包 | installers | 高风险 | 仅 90 天前的 `.exe` `.msi` `.zip` 等 |
| `dev.npm` | npm 缓存 | dev | 安全 | `npm-cache` |
| `dev.pnpm` | pnpm 缓存 | dev | 安全 | `pnpm\store` |
| `dev.pip` | pip 缓存 | dev | 安全 | `pip\Cache` |

风险等级的含义：

- **安全** —— 删了会自动重建，最多下次启动慢一点。
- **谨慎** —— 不可恢复，但通常确实是用不上的东西。
- **高风险** —— 可能删掉你还想要的东西，`clean` 时必须显式 `--force`。

## 安全设计

这一点比功能重要，所以单独说：

1. **`scan` 绝不删除。** 扫描和清理是两条完全独立的代码路径，清理时才调用删除。
2. **高风险规则默认不执行。** `clean` 会先扫描、列出、然后跳过它们并提示加 `--force`。
3. **删除前一定确认。** 直接回车就是取消；只有 `-y/--yes` 才跳过。
4. **只清内容，不删目录。** `DeleteContents` 会保留目录本身，避免破坏目录权限
   或让程序找不到自己的缓存目录。
5. **占用即跳过。** 被其他进程锁住的文件不会让整次清理失败，只会记入"跳过"。
6. **权限问题单独报。** 删不掉时提示是"权限不足"，并建议以管理员身份重跑，
   而不是甩一个原始错误码。

## JSON 输出

```bash
sweep scan --json
```

```json
{
  "total": 1440282695,
  "total_human": "1.34 GB",
  "items": [
    {
      "rule": "cache.browser",
      "name": "浏览器缓存",
      "category": "cache",
      "risk": "安全",
      "bytes": 739421874,
      "human": "705 MB",
      "files": 7146,
      "paths": ["C:\\Users\\me\\AppData\\Local\\Microsoft\\Edge\\User Data\\Default\\Cache"],
      "needs_admin": false,
      "note": "不影响书签和登录状态，网页首次加载会变慢"
    }
  ]
}
```

配合 `jq` 可以只挑出超过 1 GB 的项目：

```bash
sweep scan --json | jq '.items[] | select(.bytes > 1e9) | {rule, human}'
```

## 已知限制

- **主要面向 Windows。** 在 Linux / macOS 上能编译、能扫缓存目录，但回收站和
  休眠这两条规则不可用。
- **统计的是逻辑大小**，不是磁盘实际占用（没有按簇对齐计算）。
- **不处理 OneDrive 按需文件**，扫描时会当作普通文件计入。
- **没有增量索引**，每次都是重新遍历；在几十万文件的目录上会慢几秒。

## 项目结构

```
sweep/
├── main.go                       # CLI：参数解析、输出、确认流程
├── internal/
│   ├── rules/rules.go            # 规则表（纯数据）+ 环境变量/通配符展开
│   ├── scan/scan.go              # 只读测量：每条规则能清多少
│   ├── clean/clean.go            # 执行删除，处理占用与权限
│   │   ├── platform_windows.go   # SHEmptyRecycleBinW 直接调用
│   │   └── platform_other.go
│   ├── term/term.go              # ANSI 颜色 + 中文宽度对齐的表格
│   └── size/size.go              # 1.72 GB 这种人类可读格式
└── .github/workflows/            # CI + 发版
```

### 加一条规则

只需要在 `internal/rules/rules.go` 的 `All()` 里追加一项，扫描、清理、
`list`、README 就都有了：

```go
{
    ID:       "cache.foo",
    Name:     "某个缓存",
    Category: "cache",
    Risk:     Safe,
    Action:   DeleteContents,
    Paths:    []string{"%LOCALAPPDATA%\\Foo\\Cache"},
    Note:     "删了会自动重建",
},
```

Path 支持 `%ENV%` 变量和 `*` 通配符。加完记得跑一下
`sweep list` 和 `sweep scan --rule cache.foo`。

## 开发

```bash
go test ./...     # 单元测试
go vet ./...      # 静态检查
go build ./...
```

测试覆盖了几个容易出错的地方：中英文混排的表格对齐、`%ENV%` 展开、
按文件年龄过滤、dry-run 不落盘。

## License

[MIT](LICENSE)
