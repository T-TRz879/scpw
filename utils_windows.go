package scpw

import (
	"os"
	"strconv"
	"syscall"
	"time"
)

func StatTimeV2(file os.FileInfo) (string, string) {
	stat := file.Sys().(*syscall.Win32FileAttributeData)
	atime := time.Unix(0, stat.LastAccessTime.Nanoseconds()).Unix()
	mtime := time.Unix(0, stat.LastWriteTime.Nanoseconds()).Unix()
	return strconv.FormatInt(atime, 10), strconv.FormatInt(mtime, 10)
}
