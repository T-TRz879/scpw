package scpw

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var (
	testNode         = &Node{Host: "host", Port: "port", User: "user", Password: "password"}
	remoteDir        = []string{"/tmp/scpw", "/tmp/scpw/dir1", "/tmp/scpw/dir2", "/tmp/scpw/dir3"}
	remoteFile       = []string{"/tmp/scpw/a.txt", "/tmp/scpw/b.txt", "/tmp/scpw/c.txt"}
	noPermissionPath = "/root"
)

//func gops() {
//	if err := agent.Listen(agent.Options{
//		ShutdownCleanup: true,
//	}); err != nil {
//		log.Fatal(err)
//	}
//}

func TestAAInit(t *testing.T) {
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	for _, remote := range remoteDir {
		err := scpwCli.PutDir(context.Background(), "./testfile", remote)
		assert.Nil(t, err)
	}
	for _, remote := range remoteFile {
		err := scpwCli.Put(context.Background(), "./tmp.go", remote)
		assert.Nil(t, err)
	}
}

func TestPut(t *testing.T) {
	local, remote := "./testfile/a.txt", randName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestPutNotExist(t *testing.T) {
	local, remote := "./testfile/not-exist-file", randName("/tmp")
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutPermissionDeny(t *testing.T) {
	// SSH login user does not have remote permission
	local, remote := "./testfile/a.txt", randName(noPermissionPath)
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutLocalIsDir(t *testing.T) {
	local, remote := "./testfile", "/tmp/testfile"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutAll(t *testing.T) {
	local, remote := "./testfile", "/tmp/testfile"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.PutAll(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestGetSwitch(t *testing.T) {
	// keep remote server has remoteFile
	local, remote := "/tmp/a.txt", "/tmp/a.txt"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.SwitchScpwFunc(context.Background(), local, remote, GET)
	assert.Nil(t, err)
}

func TestGet(t *testing.T) {
	// keep remote server has remoteFile
	local, remote := "/tmp/a.txt", "/tmp/a.txt"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestGetNotExist(t *testing.T) {
	local, remote := "/tmp/a.txt", "/tmp/fad321f2312f"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestGetPermissionDeny(t *testing.T) {
	// SSH login user does not have remote permission
	local, remote := randName("/tmp"), noPermissionPath+"/a.txt"
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
	local := randName("/tmp")
	os.Mkdir(local, os.FileMode(uint32(0700)))
	log.Infof("local:%s", local)
	remote := "/tmp/scpw/"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.SwitchScpwFunc(context.Background(), local, remote, GET)
	assert.Nil(t, err)
}

func TestGetAll(t *testing.T) {
	local := randName("/tmp")
	os.Mkdir(local, os.FileMode(uint32(0700)))
	log.Infof("local:%s", local)
	remote := "/tmp/scpw"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.GetAll(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestWalkTree(t *testing.T) {
	scpCh := &scpChan{fileChan: make(chan File), exitChan: make(chan struct{}), closeChan: make(chan struct{})}
	path := "./.git/"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := WalkTree(ctx, scpCh, path, path, "/tmp/.git/")
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
