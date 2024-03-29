package scpw

import (
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strconv"
)

var (
	letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
)

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func Addr(ip, port string) string {
	return fmt.Sprintf("%s:%s", ip, port)
}

func FileModeV1(root string) (string, error) {
	file, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0%o", file.Mode().Perm()), nil
}

func FileModeV2(file os.FileInfo) string {
	return fmt.Sprintf("0%o", file.Mode().Perm())
}

func StatDirMeta(root string) (name, mode, atime, mtime string, isDir bool, err error) {
	stat, err := os.Stat(root)
	if err != nil {
		return
	}
	name = stat.Name()
	mode = FileModeV2(stat)
	atime, mtime = StatTimeV2(stat)
	isDir = stat.IsDir()
	return
}

func StatDirChild(root string) ([]os.DirEntry, error) {
	dirs, err := os.ReadDir(root)
	return dirs, err
}

func StatDir(root string) (entries []os.DirEntry, name, mode, atime, mtime string, isDir bool, err error) {
	name, mode, atime, mtime, isDir, err = StatDirMeta(root)
	if err != nil {
		return
	}
	entries, err = StatDirChild(root)
	return
}

func StatFile(root string) (name string, mode string, atime string, mtime string, size string, err error) {
	stat, err := os.Stat(root)
	if err != nil {
		return "", "", "", "", "", err
	}
	name = stat.Name()
	mode = FileModeV2(stat)
	atime, mtime = StatTimeV2(stat)
	size = strconv.FormatInt(stat.Size(), 10)
	return name, mode, size, atime, mtime, nil
}

func ParseUnit32(str string) (uint32, error) {
	res, err := strconv.ParseUint(str, 8, 32)
	if err != nil {
		return uint32(0), err
	}
	return uint32(res), err
}

func ParseInt64(str string) (int64, error) {
	num, err := strconv.Atoi(str)
	//res, err = strconv.ParseInt(str, 10, 64)
	//return res, err
	return int64(num), err
}

func RandName(root string) string {
	return filepath.Join(root, uuid.NewString())
}
