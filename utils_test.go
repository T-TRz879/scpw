package scpw

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"
)

func TestFileMode(t *testing.T) {
	mode, err := FileModeV1("./cmd/scpw/main.go")
	if err != nil {
		panic(err)
	}
	log.Infof("mode:%s", mode)
}

func TestStatTimeV2(t *testing.T) {
	open, _ := os.Stat("./cmd/scpw/main.go")
	atime, mtime := StatTimeV2(open)
	log.Infof("atime:%s mtime:%s", atime, mtime)
}

func TestParseInt64(t *testing.T) {
	res, err := ParseInt64("1446425371")
	if err != nil {
		panic(err)
	}
	time := res
	log.Infof("time:%d", time)
}

func TestParseInt8(t *testing.T) {
	a, err := ParseOctal("0777")
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
	dir, file := path.Split("/root/dir")
	log.Infof("dir:%s file:%s", dir, file)
	newDir := path.Join(dir, "a")
	log.Infof("newDir:%s", newDir)
}

func TestPath(t *testing.T) {
	//返回路径的最后一个元素
	fmt.Println(path.Base("./a/b/c"))
	//如果路径为空字符串，返回.
	fmt.Println(path.Base(""))
	//如果路径只有斜线，返回/
	fmt.Println(path.Base("///"))

	//返回等价的最短路径
	//1.用一个斜线替换多个斜线
	//2.清除当前路径.
	//3.清除内部的..和他前面的元素
	//4.以/..开头的，变成/
	fmt.Println(path.Clean("./a/b/../"))

	//返回路径最后一个元素的目录
	//路径为空则返回.
	fmt.Println(path.Dir("./a/b/c"))
	fmt.Println(path.Dir("/a/b/c"))
	fmt.Println(path.Dir("/a/b/c/"))

	//返回路径中的扩展名
	//如果没有点，返回空
	fmt.Println(path.Ext("./a/b/c/d.jpg"))

	//判断路径是不是绝对路径
	fmt.Println(path.IsAbs("./a/b/c"))
	fmt.Println(path.IsAbs("/a/b/c"))

	//连接路径，返回已经clean过的路径
	fmt.Println(path.Join("./a", "b/c", "../d/"))

	//匹配文件名，完全匹配则返回true
	fmt.Println(path.Match("*", "a"))
	fmt.Println(path.Match("*", "a/b/c"))
	fmt.Println(path.Match("\\b", "b"))

	//分割路径中的目录与文件
	fmt.Println(path.Split("./a/b/c/d.jpg"))
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
