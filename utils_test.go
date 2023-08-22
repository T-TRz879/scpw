package scpw

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMinInt(t *testing.T) {
	a, b := 1, 2
	require.Equal(t, a, MinInt(a, b))
	require.Equal(t, a, MinInt(b, a))
}

func TestMaxInt(t *testing.T) {
	a, b := 1, 2
	require.Equal(t, b, MaxInt(a, b))
	require.Equal(t, b, MaxInt(b, a))
}

func TestMinInt64(t *testing.T) {
	a, b := int64(1), int64(2)
	require.Equal(t, a, MinInt64(a, b))
	require.Equal(t, a, MinInt64(b, a))
}

func TestMaxInt64(t *testing.T) {
	a, b := int64(1), int64(2)
	require.Equal(t, b, MaxInt64(a, b))
	require.Equal(t, b, MaxInt64(b, a))
}

func TestAddr(t *testing.T) {
	require.Equal(t, "127.0.0.1:80", Addr("127.0.0.1", "80"))
}

func TestFileModeV1(t *testing.T) {
	mode, err := FileModeV1("./cmd/scpw/main.go")
	require.Nil(t, err)
	log.Infof(mode)

	mode, err = FileModeV1("./notexist.go")
	require.NotNil(t, err)
}

func TestFileModeV2(t *testing.T) {
	stat, err := os.Stat("./cmd/scpw/main.go")
	require.Nil(t, err)
	v2 := FileModeV2(stat)
	log.Infof(v2)
}

func TestStatDirMeta(t *testing.T) {
	_, _, _, _, _, err := StatDirMeta("./notexist")
	require.NotNil(t, err)

	name, mode, atime, mtime, dir, err := StatDirMeta("./cmd")
	require.Nil(t, err)
	log.Infof("name:%s mode:%s atime:%s mtime:%s dir:%v", name, mode, atime, mtime, dir)
}

func TestStatDirChild(t *testing.T) {
	child, err := StatDirChild("./cmd")
	require.Nil(t, err)
	log.Infof("child:%v", child)
}

func TestStatDir(t *testing.T) {
	entries, name, mode, atime, mtime, isDir, err := StatDir("./cmd")
	require.Nil(t, err)
	log.Infof("entries:%v name:%s mode:%s atime:%s mtime:%s isDir:%v", entries, name, mode, atime, mtime, isDir)

	_, _, _, _, _, _, err = StatDir("./notexist")
	require.NotNil(t, err)
}

func TestFile(t *testing.T) {
	name, mode, atime, mtime, size, err := StatFile("./cmd/scpw/main.go")
	require.Nil(t, err)
	log.Infof("name:%s mode:%s atime:%s mtime:%s size:%s", name, mode, atime, mtime, size)

	_, _, _, _, _, err = StatFile("./notexist")
	require.NotNil(t, err)
}

func TestParseUnit32(t *testing.T) {
	unit32, err := ParseUnit32("0777")
	require.Nil(t, err)
	require.Equal(t, uint32(511), unit32)

	_, err = ParseUnit32("3gadfsgasd")
	require.NotNil(t, err)
}

func TestParseInt64(t *testing.T) {
	res, err := ParseInt64("1446425371")
	require.Nil(t, err)
	require.Equal(t, int64(1446425371), res)
}

func TestStatTimeV2(t *testing.T) {
	open, _ := os.Stat("./cmd/scpw/main.go")
	atime, mtime := StatTimeV2(open)
	log.Infof("atime:%s mtime:%s", atime, mtime)
}

func TestParseInt8(t *testing.T) {
	a, err := ParseUnit32("0777")
	if err != nil {
		panic(err)
	}
	mode := os.FileMode(a)
	b := fmt.Sprintf("%o", mode)
	log.Infof("mode:%d %s", a, b)
}

func TestStampToTime(t *testing.T) {
	timeA := time.Unix(1446425371, 0)
	timeB := time.Unix(1669346386, 0)
	log.Infof("timeA:%d timeB:%d", timeA.Unix(), timeB.Unix())
}

func TestPathFunc(t *testing.T) {
	dir, file := filepath.Split("/root/dir")
	log.Infof("dir:%s file:%s", dir, file)
	newDir := filepath.Join(dir, "a")
	log.Infof("newDir:%s", newDir)
}

func TestPath(t *testing.T) {
	//返回路径的最后一个元素
	fmt.Println(filepath.Base("./a/b/c"))
	//如果路径为空字符串，返回.
	fmt.Println(filepath.Base(""))
	//如果路径只有斜线，返回/
	fmt.Println(filepath.Base("///"))

	//返回等价的最短路径
	//1.用一个斜线替换多个斜线
	//2.清除当前路径.
	//3.清除内部的..和他前面的元素
	//4.以/..开头的，变成/
	fmt.Println(filepath.Clean("./a/b/../"))

	//返回路径最后一个元素的目录
	//路径为空则返回.
	fmt.Println(filepath.Dir("./a/b/c"))
	fmt.Println(filepath.Dir("/a/b/c"))
	fmt.Println(filepath.Dir("/a/b/c/"))

	//返回路径中的扩展名
	//如果没有点，返回空
	fmt.Println(filepath.Ext("./a/b/c/d.jpg"))

	//判断路径是不是绝对路径
	fmt.Println(filepath.IsAbs("./a/b/c"))
	fmt.Println(filepath.IsAbs("/a/b/c"))

	//连接路径，返回已经clean过的路径
	fmt.Println(filepath.Join("./a", "b/c", "../d/"))

	//匹配文件名，完全匹配则返回true
	fmt.Println(filepath.Match("*", "a"))
	fmt.Println(filepath.Match("*", "a/b/c"))
	fmt.Println(filepath.Match("\\b", "b"))

	//分割路径中的目录与文件
	fmt.Println(filepath.Split("./a/b/c/d.jpg"))
}

func TestRandByte(t *testing.T) {
	b1, b2 := make([]byte, 100), make([]byte, 100)
	rand.Read(b1)
	rand.Read(b2)
	diff := 0
	for i := 0; i < len(b1); i++ {
		if b1[i] != b2[i] {
			diff++
		}
	}
	log.Infof("different byte count:%d", diff)
}
