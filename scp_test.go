package scpw

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

var (
	testNode         = &Node{Host: "127.0.0.1", Port: "22", User: "scpwuser", Password: "scpwuser123"}
	baseLocalDir     = "/tmp/scpw-local-dir"
	baseRemoteDir    = "/tmp/scpw-remote-dir"
	noPermissionDir  = "/tmp/no-permission-dir"
	noPermissionFile = "/tmp/no-permission-file"
)

func writeFile(name string) error {
	_, err := os.Create(name)
	if err != nil {
		return err
	}
	return os.WriteFile(name, []byte{1, 2, 3, 4}, os.FileMode(0777))
}

func mkdir(name string) error {
	err := os.Mkdir(name, os.FileMode(0777))
	if err != nil {
		return err
	}
	return os.Chmod(name, os.FileMode(0777)|os.FileMode(02))
}

func TestAttr(t *testing.T) {
	attr := Attr{}

	err := attr.SetMode("qwertyuio")
	require.NotNil(t, err)

	err = attr.SetSize("123a")
	require.NotNil(t, err)

	err = attr.SetTime("1446425371", "dsadas")
	require.NotNil(t, err)

	err = attr.SetTime("dsadas", "1446425371")
	require.NotNil(t, err)
}

func TestPutFile(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := RandName(baseLocalDir), RandName("/tmp")
	assert.Nil(t, writeFile(local))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context, local, remote)
	assert.Nil(t, err)
}

func TestPutFileRemoteNotExist(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := filepath.Join(baseLocalDir, "not-exist"), RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context, local, remote)
	assert.NotNil(t, err)
}

func TestPutFileLocalPermissionDeny(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := noPermissionFile, RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context, local, remote)
	assert.NotNil(t, err)
}

func TestPutFileRemotePermissionDeny(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := RandName(baseRemoteDir), RandName(noPermissionDir)
	assert.Nil(t, writeFile(local))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context, local, remote)
	assert.NotNil(t, err)
}

func TestPutFileLocalIsDir(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := baseLocalDir, RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context, local, remote)
	assert.NotNil(t, err)
}

func TestPutAll(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := baseLocalDir, RandName("/tmp")
	assert.Nil(t, mkdir(remote))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.PutAll(context, local, remote)
	assert.Nil(t, err)
}

func TestGetFile(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := RandName("/tmp"), RandName(baseRemoteDir)
	assert.Nil(t, writeFile(remote))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context, local, remote)
	assert.Nil(t, err)
}

func TestGetFileNotExist(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := RandName("/tmp"), RandName(baseRemoteDir)
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context, local, remote)
	assert.NotNil(t, err)
}

func TestGetFilePermissionDeny(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	// SSH login user does not have remote permission
	local, remote := RandName("/tmp"), noPermissionFile
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context, local, remote)
	assert.NotNil(t, err)
}

func TestGetFileRemoteIsDir(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local, remote := RandName("/tmp"), baseLocalDir
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context, local, remote)
	assert.NotNil(t, err)
}

func TestGetAll(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local := RandName("/tmp")
	assert.Nil(t, mkdir(local))
	remote := baseLocalDir
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.GetAll(context, local, remote)
	assert.Nil(t, err)
}

func TestPutSwitchScpwFunc(t *testing.T) {
	// put file
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local := RandName("/tmp")
	assert.Nil(t, writeFile(local))
	remote := RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.SwitchScpwFunc(context, local, remote, PUT)
	assert.Nil(t, err)

	local = RandName("/tmp")
	assert.Nil(t, mkdir(local))
	assert.Nil(t, writeFile(RandName(local)))
	assert.Nil(t, writeFile(RandName(local)))
	remote = RandName("/tmp")
	assert.Nil(t, mkdir(remote))
	// put dir all
	err = scpwCli.SwitchScpwFunc(context, local, remote, PUT)
	assert.Nil(t, err)

	// put dir exclude root
	remote = RandName("/tmp")
	assert.Nil(t, mkdir(remote))
	err = scpwCli.SwitchScpwFunc(context, local+"/*", remote, PUT)
	assert.Nil(t, err)

	// put file permission deny
	local = "/tmp/notexist"
	remote = RandName("/tmp")
	err = scpwCli.SwitchScpwFunc(context, local, remote, PUT)
	assert.NotNil(t, err)
}

func TestGetSwitchScpwFunc(t *testing.T) {
	// get file
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	local := RandName("/tmp")
	assert.Nil(t, mkdir(local))
	remote := RandName("/tmp")
	assert.Nil(t, writeFile(remote))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.SwitchScpwFunc(context, local, remote, GET)
	assert.Nil(t, err)

	// get dir all
	local = RandName("/tmp")
	assert.Nil(t, mkdir(local))
	remote = baseRemoteDir
	err = scpwCli.SwitchScpwFunc(context, local, remote+"/", GET)
	assert.Nil(t, err)

	// get file local permission deny
	local = noPermissionDir + "/"
	remote = baseRemoteDir + "/"
	err = scpwCli.SwitchScpwFunc(context, local, remote, GET)
	assert.NotNil(t, err)

	// get file remote permission deny
	local = RandName("/tmp")
	remote = noPermissionFile
	err = scpwCli.SwitchScpwFunc(context, local, remote, GET)
	assert.NotNil(t, err)
}

func TestWalkTree(t *testing.T) {
	p := NewProgress()
	context := Context{Ctx: context.Background(), Bar: p.NewInfiniteByesBar("")}
	scpCh := &scpChan{fileChan: make(chan File), exitChan: make(chan struct{}), closeChan: make(chan struct{})}
	path := "./"
	go func() {
		err := WalkTree(context, scpCh, path, path, "/tmp/")
		if err != nil {
			panic(err)
		}
	}()
	numF, numD := 0, 0
loop:
	for {
		select {
		case file := <-scpCh.fileChan:
			log.Infof("%v", file)
			if file.IsDir {
				numD++
			} else {
				numF++
			}
		case <-scpCh.exitChan:
			//cancel()
			log.Infof("E")
		case <-scpCh.closeChan:
			break loop
		}
	}
	log.Infof("file:%d dir:%d", numF, numD)
}

func TestAck(t *testing.T) {
	newBuffer := bytes.NewBuffer([]byte{})
	assert.Nil(t, ack(newBuffer))
}

func TestParseContent(t *testing.T) {
	p := NewProgress()
	in := bytes.NewBuffer([]byte{1, 2, 3, 4})
	out := bytes.NewReader(make([]byte, 4))

	// process success
	assert.Nil(t, parseContent(p.NewInfiniteByesBar(""), in, out, int64(4)))

	// EOF
	assert.NotNil(t, parseContent(p.NewInfiniteByesBar(""), in, out, int64(5)))
}
