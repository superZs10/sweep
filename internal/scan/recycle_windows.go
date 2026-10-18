//go:build windows

package scan

import (
	"syscall"
	"unsafe"
)

// rbInfo 对应 Windows 的 SHQUERYRBINFO 结构体。
// 注意 C 结构里 cbSize 后面有 4 字节对齐填充，Go 的字段顺序刚好一致。
type rbInfo struct {
	CbSize   uint32
	Size     int64
	NumItems int64
}

var (
	shell32         = syscall.NewLazyDLL("shell32.dll")
	procQueryRbInfo = shell32.NewProc("SHQueryRecycleBinW")
)

// RecycleBinStats 返回回收站的真实大小（字节）和项目数。
//
// 用 SHQueryRecycleBinW 而不是遍历 $Recycle.Bin：后者会把残留的
// desktop.ini 之类元数据也算进去，导致"回收站是空的"却报告有垃圾。
func RecycleBinStats() (int64, int64) {
	var info rbInfo
	info.CbSize = uint32(unsafe.Sizeof(info))
	// pszRootPath 传 NULL 表示统计所有驱动器
	r, _, _ := procQueryRbInfo.Call(0, uintptr(unsafe.Pointer(&info)))
	if uint32(r) != 0 {
		return 0, 0
	}
	return info.Size, info.NumItems
}
