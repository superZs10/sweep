//go:build !windows

package scan

// RecycleBinStats 在非 Windows 平台上没有回收站概念。
func RecycleBinStats() (int64, int64) { return 0, 0 }
