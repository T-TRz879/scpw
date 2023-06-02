package scpw

import (
	"context"
	"github.com/google/gops/agent"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testNode = &Node{Host: "10.0.16.18", Port: "22", User: "root", Password: "312"}
)

func gops() {
	if err := agent.Listen(agent.Options{
		ShutdownCleanup: true, // automatically closes on os.Interrupt
	}); err != nil {
		log.Fatal(err)
	}
}

func TestPut(t *testing.T) {
	local, remote := "./testfile/a.txt", "/tmp/a.txt"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.Nil(t, err)
}

func TestPutNotExist(t *testing.T) {
	local, remote := "./testfile/c.txt", "/tmp/a.txt"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestPutButLocalIsDir(t *testing.T) {
	local, remote := "./testfile", "/tmp/testfile"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Put(context.Background(), local, remote)
	assert.NotNil(t, err)
}

func TestGet(t *testing.T) {
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

func TestGetButRemoteIsDir(t *testing.T) {
	local, remote := "/tmp/a.txt", "/tmp/bb"
	ssh, err := NewSSH(testNode)
	assert.Nil(t, err)
	scpwCli := NewSCP(ssh, true)
	err = scpwCli.Get(context.Background(), local, remote)
	assert.NotNil(t, err)
}

//func TestPutDir(t *testing.T) {
//	resource := utils2.NewTestResource(true)
//	log.Infof("resource:%v", resource)
//	sshSetting, err := setting.ReadSSHSetting()
//	if err != nil {
//		panic(err)
//	}
//	ssh, err := NewSSH(sshSetting)
//	defer ssh.Conn.Close()
//	defer ssh.Close()
//	if err != nil {
//		panic(err)
//	}
//	scp := NewSCP(ssh, false)
//	atime, mtime := utils2.StatTimeV2(resource.FileInfo)
//	mode, err := utils2.FileModeV1(resource.Path)
//	if err != nil {
//		panic(err)
//	}
//	err = scp.PutDir(context.Background(), path.Join(utils2.TmpRoot, resource.Name()), mode, atime, mtime)
//	log.Infof("%v", err)
//}
//
//func TestPutAll(t *testing.T) {
//	sshSetting, err := setting.ReadSSHSetting()
//	if err != nil {
//		panic(err)
//	}
//	ssh, err := NewSSH(sshSetting)
//	defer ssh.Conn.Close()
//	defer ssh.Close()
//	if err != nil {
//		panic(err)
//	}
//	scp := NewSCP(ssh, false)
//	err = scp.PutAll(context.Background(), "/home/trz/lib/", "/root/dos-webservice/lib/", false)
//	if err != nil {
//		panic(err)
//	}
//}
//
//func TestGet(t *testing.T) {
//	sshSetting, err := setting.ReadSSHSetting()
//	if err != nil {
//		panic(err)
//	}
//	ssh, err := NewSSH(sshSetting)
//	defer ssh.Conn.Close()
//	defer ssh.Close()
//	if err != nil {
//		panic(err)
//	}
//	scp := NewSCP(ssh, false)
//	err = scp.Get(context.Background(), "/home/trz/install.sh", "/root/install.sh")
//	if err != nil {
//		panic(err)
//	}
//}
//
//func TestGetAll(t *testing.T) {
//	sshSetting, err := setting.ReadSSHSetting()
//	if err != nil {
//		panic(err)
//	}
//	ssh, err := NewSSH(sshSetting)
//	defer ssh.Conn.Close()
//	defer ssh.Close()
//	if err != nil {
//		panic(err)
//	}
//	scp := NewSCP(ssh, false)
//	os.RemoveAll("/home/trz/jdk1.8.0_351")
//	err = scp.GetAll(context.Background(), "/home/trz", "/root/environment/jdk1.8.0_351")
//	if err != nil {
//		panic(err)
//	}
//}

func TestWalkTree(t *testing.T) {
	scpCh := &scpChan{fileChan: make(chan File), exitChan: make(chan struct{}), closeChan: make(chan struct{})}
	path := "../scpw"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := WalkTree(ctx, scpCh, path, path, "/tmp", false)
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
