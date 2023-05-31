package scpw

import (
	"errors"
	"fmt"
	"github.com/go-cmd/cmd"
	"github.com/mattn/go-isatty"
	"golang.org/x/crypto/ssh"
	"os"
	"runtime"
	"strconv"
)

var unit = []string{"B", "KB", "GB", "TB"}

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

func SupportANSIColor(fd uintptr) bool {
	return isatty.IsTerminal(fd) && runtime.GOOS != "windows"
}

func HostKey(ip string) (ssh.PublicKey, error) {
	findCmd := cmd.NewCmd("ssh-keygen", "-F", ip)
	statusChan := findCmd.Start()
	finalStatus := <-statusChan
	if finalStatus.Error != nil || len(finalStatus.Stdout) == 0 {
		log.Errorf("cannot find ip:{%s} HostKey", ip)
		return nil, errors.New("find HostKey fail")
	}
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(finalStatus.Stdout[1]))
	return hostKey, err
}

func FileModeV1(path string) (string, error) {
	file, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0%o", file.Mode().Perm()), nil
}

func FileModeV2(file os.FileInfo) string {
	return fmt.Sprintf("0%o", file.Mode().Perm())
}

func StatDir(path string) ([]os.DirEntry, string, string, string, string, error) {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, "", "", "", "", err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, "", "", "", "", err
	}
	name := stat.Name()
	mode := FileModeV2(stat)
	atime, mtime := StatTimeV2(stat)
	return dirs, name, mode, atime, mtime, nil
}

func StatFile(path string) (string, string, string, string, string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", "", "", "", "", err
	}
	name := stat.Name()
	mode := FileModeV2(stat)
	atime, mtime := StatTimeV2(stat)
	size := strconv.FormatInt(stat.Size(), 10)
	return name, mode, size, atime, mtime, nil
}

func ParseOctal(str string) (res int64, err error) {
	res, err = strconv.ParseInt(str, 0, 64)
	return res, err
}

func ParseInt64(str string) (int64, error) {
	num, err := strconv.Atoi(str)
	//res, err = strconv.ParseInt(str, 10, 64)
	//return res, err
	return int64(num), err
}

func ParseMode(s string) os.FileMode {
	mode := make([]byte, 3)
	for i := 0; i < 3; i++ {
		cur := 0
		if s[i*3] != '-' {
			cur += 4
		}
		if s[i*3+1] != '-' {
			cur += 2
		}
		if s[i*3+2] != '-' {
			cur += 1
		}
		mode[i] = byte(cur + '0')
	}
	res, _ := strconv.Atoi(string(mode))
	return os.FileMode(res)
}
