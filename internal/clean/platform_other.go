//go:build !windows

package clean

import (
	"errors"
	"fmt"
)

var errUnsupported = errors.New("当前平台不支持该操作")

type cmdError struct {
	cmd string
	out string
	err error
}

func (e *cmdError) Error() string {
	return fmt.Sprintf("%s 失败: %v", e.cmd, e.err)
}

func emptyRecycleBin() error { return errUnsupported }
