// Package size 提供人类可读的字节数格式化。
package size

import "fmt"

// Human 把字节数格式化成 "1.72 GB" 这样的形式。
func Human(n int64) string {
	if n < 0 {
		return "-" + Human(-n)
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n)
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	i := -1
	for value >= unit && i < len(units)-1 {
		value /= unit
		i++
	}
	if value >= 100 {
		return fmt.Sprintf("%.0f %s", value, units[i])
	}
	return fmt.Sprintf("%.2f %s", value, units[i])
}
