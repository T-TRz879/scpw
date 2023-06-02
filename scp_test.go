package scpw

import (
	"context"
	"testing"
)

//func TestPut(t *testing.T) {
//	resource := NewTestResource(false)
//	log.Infof("resource:%v", resource)
//	sshSetting, err := setting.ReadSSHSetting()
//	if err != nil {
//		panic(err)
//	}
//	ssh, err := NewSSH(sshSetting)
//	if err != nil {
//		panic(err)
//	}
//	defer ssh.Conn.Close()
//	defer ssh.Close()
//	if err != nil {
//		panic(err)
//	}
//	scp := NewSCP(ssh, false)
//	err = scp.Put(context.Background(), resource.Path, path.Join(utils2.TmpRoot, resource.Name()))
//	log.Infof("%v", err)
//}
//
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
