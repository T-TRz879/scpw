package scpw

import (
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
	local, remote := RandName(baseLocalDir), RandName("/tmp")
	assert.Nil(t, writeFile(local))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestPutFileRemoteNotExist(t *testing.T) {
	local, remote := filepath.Join(baseLocalDir, "not-exist"), RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutFileLocalPermissionDeny(t *testing.T) {
	local, remote := noPermissionFile, RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutFileRemotePermissionDeny(t *testing.T) {
	local, remote := RandName(baseRemoteDir), RandName(noPermissionDir)
	assert.Nil(t, writeFile(local))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutFileLocalIsDir(t *testing.T) {
	local, remote := baseLocalDir, RandName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutAll(t *testing.T) {
	local, remote := baseLocalDir, RandName("/tmp")
	assert.Nil(t, mkdir(remote))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.PutAll(context.Background(), local, remote)
	assert.Nil(t, err)
}

//func TestGetSwitch(t *testing.T) {
//	// keep remote server has remoteFile
//	local, remote := RandName(baseDir), filepath.Join(baseDir, "file1")
//	ssh, err := NewSSH(testNode)
//	assert.Nil(t, err)
//	scpwCli := NewSCP(ssh, true)
//	err = scpwCli.SwitchScpwFunc(context.Background(), local, remote, GET)
//	assert.Nil(t, err)
//}

func TestGetFile(t *testing.T) {
	local, remote := RandName("/tmp"), RandName(baseRemoteDir)
	assert.Nil(t, mkdir(local))
	assert.Nil(t, writeFile(remote))
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestGetFileNotExist(t *testing.T) {
	local, remote := RandName("/tmp"), RandName(baseRemoteDir)
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestGetFilePermissionDeny(t *testing.T) {
	// SSH login user does not have remote permission
	local, remote := RandName("/tmp"), noPermissionFile
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestGetFileRemoteIsDir(t *testing.T) {
	local, remote := RandName("/tmp"), baseLocalDir
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.NotNil(t, err)
}

//func TestGetAllSwitch(t *testing.T) {
//	local := RandName("/tmp")
//	os.Mkdir(local, os.FileMode(uint32(0777)))
//	log.Infof("local:%s", local)
//	remote := filepath.Join(baseDir, "dir1/")
//	ssh, err := NewSSH(testNode)
//	assert.Nil(t, err)
//	scpwCli := NewSCP(ssh, true)
//	err = scpwCli.SwitchScpwFunc(context.Background(), local, remote, GET)
//	assert.NotNil(t, err)
//}

func TestGetAll(t *testing.T) {
	local := RandName("/tmp")
	assert.Nil(t, mkdir(local))
	remote := baseLocalDir
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.GetAll(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestWalkTree(t *testing.T) {
	scpCh := &scpChan{fileChan: make(chan File), exitChan: make(chan struct{}), closeChan: make(chan struct{})}
	path := "./"
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
