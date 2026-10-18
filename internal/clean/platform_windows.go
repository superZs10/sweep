//go:build windows

package clean

import (
	"errors"
	"fmt"
	"syscall"
)

// errUnsupported 表示当前平台不支持该操作。
var errUnsupported = errors.New("当前平台不支持该操作")

// cmdError 包装外部命令失败。
type cmdError struct {
	cmd string
	out string
	err error
}

func (e *cmdError) Error() string {
	if e.out == "" {
		return fmt.Sprintf("%s 失败: %v", e.cmd, e.err)
	}
	return fmt.Sprintf("%s 失败: %v (%s)", e.cmd, e.err, trimSpace(e.out))
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

var (
	shell32             = syscall.NewLazyDLL("shell32.dll")
	procEmptyRecycleBin = shell32.NewProc("SHEmptyRecycleBinW")
)

// emptyRecycleBin 直接调用 shell32 的 SHEmptyRecycleBinW，
// 避免依赖 PowerShell 之类的外部程序。
func emptyRecycleBin() error {
	// SHERB_NOCONFIRMATION(0x1) | SHERB_NOPROGRESSUI(0x2) | SHERB_NOSOUND(0x4)
	// 不传标志的话会弹出系统确认框，在命令行里会一直卡住。
	const flags = 0x1 | 0x2 | 0x4
	// (HWND, LPCWSTR 根路径, DWORD 标志) -> HRESULT
	r, _, _ := procEmptyRecycleBin.Call(0, 0, flags)
	hr := uint32(r)
	// E_UNEXPECTED：回收站本来就是空的，不算失败
	if hr == 0 || hr == 0x8000FFFF {
		return nil
	}
	return fmt.Errorf("SHEmptyRecycleBinW 返回 0x%X", hr)
}
