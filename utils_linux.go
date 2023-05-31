package scpw

import (
	"os"
	"strconv"
	"syscall"
)

func StatTimeV2(file os.FileInfo) (string, string) {
	stat := file.Sys().(*syscall.Stat_t)
	atime := strconv.FormatInt(stat.Atim.Sec, 10)
	mtime := strconv.FormatInt(stat.Mtim.Sec, 10)
	return atime, mtime
}
