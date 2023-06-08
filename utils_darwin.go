package scpw

import (
	"os"
	"strconv"
	"syscall"
)

func StatTimeV2(file os.FileInfo) (string, string) {
	stat := file.Sys().(*syscall.Stat_t)
	atime := strconv.FormatInt(stat.Atimespec.Sec, 10)
	mtime := strconv.FormatInt(stat.Mtimespec.Sec, 10)
	return atime, mtime
}
