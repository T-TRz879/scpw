package scpw

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

var (
	testNode = &Node{Host: "127.0.0.1", Port: "22", User: "scpwuser", Password: "scpwuser123"}
	baseDir  = "/tmp/scpw-test-dir"
)

func TestPut(t *testing.T) {
	local, remote := filepath.Join(baseDir, "file1"), RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestPutNotExist(t *testing.T) {
	local, remote := filepath.Join(baseDir, "not-exist"), RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutPermissionDeny(t *testing.T) {
	// SSH login user does not have remote permission
	local, remote := filepath.Join(baseDir, "file1"), RandName("/root")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutLocalIsDir(t *testing.T) {
	local, remote := filepath.Join(baseDir), RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutAll(t *testing.T) {
	local, remote := baseDir, "/tmp"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.PutAll(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestGetSwitch(t *testing.T) {
	// keep remote server has remoteFile
	local, remote := RandName(baseDir), filepath.Join(baseDir, "file1")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.SwitchScpwFunc(context.Background(), local, remote, GET)
	assert.Nil(t, err)
}

func TestGet(t *testing.T) {
	// keep remote server has remoteFile
	local, remote := RandName(baseDir), filepath.Join(baseDir, "file1")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestGetNotExist(t *testing.T) {
	local, remote := RandName(baseDir), "/tmp/fad321f2312f"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestGetPermissionDeny(t *testing.T) {
	// SSH login user does not have remote permission
	local, remote := RandName(baseDir), "/root/a"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestGetRemoteIsDir(t *testing.T) {
	local, remote := "/tmp/a.txt", "/tmp/bb"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestGetAllSwitch(t *testing.T) {
	local := RandName("/tmp")
	os.Mkdir(local, os.FileMode(uint32(0700)))
	log.Infof("local:%s", local)
	remote := baseDir
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.SwitchScpwFunc(context.Background(), local, remote, GET)
	assert.Nil(t, err)
}

func TestGetAll(t *testing.T) {
	local := RandName("/tmp")
	os.Mkdir(local, os.FileMode(uint32(0700)))
	log.Infof("local:%s", local)
	remote := baseDir
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.GetAll(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestWalkTree(t *testing.T) {
	scpCh := &scpChan{fileChan: make(chan File), exitChan: make(chan struct{}), closeChan: make(chan struct{})}
	path := baseDir
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := WalkTree(ctx, scpCh, path, path, "/tmp/")
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
