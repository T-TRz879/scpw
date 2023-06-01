package scpw

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var log = GetLogger("scpw")

type CommandType = string

const (
	C CommandType = "C"
	D CommandType = "D"
	E CommandType = "E"
	T CommandType = "T"
)

type File struct {
	Name       string
	LocalPath  string
	RemotePath string
	Mode       string
	Atime      string
	Mtime      string
	Size       string
	IsDir      bool
}

func NewFile(name, localPath, remotePath, mode, atime, mtime, size string, dir bool) File {
	return File{
		Name:       name,
		LocalPath:  localPath,
		RemotePath: remotePath,
		Mode:       mode,
		Atime:      atime,
		Mtime:      mtime,
		Size:       size,
		IsDir:      dir,
	}
}

type Attr struct {
	Name  string
	Mode  os.FileMode
	Size  int64
	Atime time.Time
	Mtime time.Time
	Typ   CommandType
}

func (a *Attr) SetMode(str string) error {
	mode, err := ParseOctal(str)
	if err != nil {
		return err
	}
	a.Mode = os.FileMode(mode)
	return nil
}

func (a *Attr) SetSize(str string) error {
	size, err := ParseInt64(str)
	if err != nil {
		return err
	}
	a.Size = size
	return nil
}

func (a *Attr) SetTime(aStr, mStr string) error {
	atime, err := ParseInt64(aStr)
	if err != nil {
		return err
	}
	mtime, err := ParseInt64(mStr)
	if err != nil {
		return err
	}
	a.Atime = time.Unix(atime, 0)
	a.Mtime = time.Unix(mtime, 0)
	return nil
}

type scpChan struct {
	fileChan  chan File
	exitChan  chan struct{}
	closeChan chan struct{}
}

type SCP struct {
	*ssh.Client
	KeepTime   bool
	TimeOption string
}

func NewSSH(node *Node) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: node.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(node.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", Addr(node.Host, node.Port), config)
	return client, err
}

func NewSCP(cli *ssh.Client, keepTime bool) *SCP {
	timeOption := " "
	if keepTime {
		timeOption = "p" + timeOption
	}
	return &SCP{
		Client:     cli,
		KeepTime:   keepTime,
		TimeOption: timeOption,
	}
}

func (scp *SCP) SwitchScpwFunc(ctx context.Context, localPath, remotePath string, typ SCPWType) error {
	excludeRootDir := false
	if typ == PUT {
		if localPath[len(localPath)-1] == '*' {
			excludeRootDir = true
			localPath = localPath[:len(localPath)-1]
		}
		stat, err := os.Stat(localPath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			return scp.PutAll(ctx, localPath, remotePath, excludeRootDir)
		} else {
			return scp.Put(ctx, localPath, remotePath)
		}
	} else {
		if remotePath[len(remotePath)-1] == '*' {
			excludeRootDir = true
			remotePath = remotePath[:len(remotePath)-1]
		}
		if excludeRootDir {
			return scp.GetAll(ctx, localPath, remotePath)
		} else {
			return scp.Get(ctx, localPath, remotePath)
		}
	}
}

func (scp *SCP) PutAll(ctx context.Context, srcPath, dstPath string, excludeRootDir bool) error {
	wg := sync.WaitGroup{}
	wg.Add(2)
	session, err := scp.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	errChan := make(chan error, 1)
	scpCh := &scpChan{fileChan: make(chan File), exitChan: make(chan struct{}), closeChan: make(chan struct{})}
	go func() {
		err := WalkTree(ctx, scpCh, srcPath, srcPath, path.Join(dstPath, path.Base(srcPath)), excludeRootDir)
		if err != nil {
			errChan <- err
		}
	}()

	go func() {
		defer stdin.Close()
		defer wg.Done()
	loop:
		for {
			select {
			case file := <-scpCh.fileChan:
				switch file.Name {
				case "":
					_, err = fmt.Fprintln(stdin, "E")
					if err != nil {
						errChan <- err
						return
					}

					err = checkResponse(stdout)
					if err != nil {
						errChan <- err
						return
					}
				default:
					if scp.KeepTime {
						_, err = fmt.Fprintln(stdin, "T"+file.Mtime, "0", file.Atime, "0")
						if err != nil {
							errChan <- err
							return
						}

						err = checkResponse(stdout)
						if err != nil {
							errChan <- err
							return
						}
					}

					typ, size := "C", file.Size
					if file.IsDir {
						typ = "D"
						size = "0"
					}
					_, err = fmt.Fprintln(stdin, typ+file.Mode, size, file.Name)
					if err != nil {
						errChan <- err
						return
					}

					err = checkResponse(stdout)
					if err != nil {
						errChan <- err
						return
					}

					if !file.IsDir {
						sizeNum, err := ParseInt64(size)
						if err != nil {
							errChan <- err
							return
						}
						open, err := os.Open(file.LocalPath)
						err = parseContent(stdin, open, sizeNum)
						open.Close()
						if err != nil {
							errChan <- err
							return
						}

						_, err = fmt.Fprint(stdin, "\x00")
						if err != nil {
							errChan <- err
							return
						}

						err = checkResponse(stdout)
						if err != nil {
							errChan <- err
							return
						}
					}

					log.Printf("file:[%40s] size:[%15s]\n", file.Name, file.Size)
				}
			case <-scpCh.exitChan:
				_, err = fmt.Fprintln(stdin, E)
				if err != nil {
					errChan <- err
					return
				}

				err = checkResponse(stdout)
				if err != nil {
					errChan <- err
					return
				}
			case <-scpCh.closeChan:
				break loop
			}
		}
	}()

	go func() {
		defer wg.Done()
		err := session.Run(fmt.Sprintf("scp -rt%s%q", scp.TimeOption, dstPath))
		if err != nil {
			errChan <- err
		}

		err = checkResponse(stdout)
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func WalkTree(ctx context.Context, scpChan *scpChan, rootParent, root, dstPath string, excludeRootDir bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		child, name, mode, atime, mtime, err := StatDir(root)
		if err != nil {
			return err
		}
		if rootParent == root && !excludeRootDir {
			scpChan.fileChan <- NewFile(name, root, dstPath, mode, atime, mtime, "0", true)
		}
		var dirs []os.DirEntry
		for _, obj := range child {
			if !obj.IsDir() {
				filePath := path.Join(root, obj.Name())
				name, mode, size, atime, mtime, err := StatFile(filePath)
				if err != nil {
					return nil
				}
				scpChan.fileChan <- NewFile(name, filePath, path.Join(dstPath, name), mode, atime, mtime, size, false)
			} else {
				dirs = append(dirs, obj)
			}
		}

		if err != nil {
			return err
		}
		for _, dir := range dirs {
			err := WalkTree(ctx, scpChan, rootParent, path.Join(root, dir.Name()), path.Join(dstPath, dir.Name()), excludeRootDir)
			if err != nil {
				return err
			}
		}
		scpChan.exitChan <- struct{}{}
		if rootParent == root {
			scpChan.closeChan <- struct{}{}
		}
		return nil
	}
}

func (scp *SCP) PutDir(dstPath string, mode string, atime, mtime string) error {
	wg := sync.WaitGroup{}
	session, err := scp.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	errChan := make(chan error, 1)

	fileName := filepath.Base(dstPath)

	wg.Add(2)
	go func() {
		defer func() {
			wg.Done()
		}()
		defer stdin.Close()

		if scp.KeepTime {
			// T+Mtime 0 Atime 0
			_, err = fmt.Fprintln(stdin, "T"+mtime, "0", atime, "0")
			if err != nil {
				errChan <- err
				return
			}

			err = checkResponse(stdout)
			if err != nil {
				errChan <- err
				return
			}
		}

		// C+MODE SIZE NAME
		_, err = fmt.Fprintln(stdin, "D"+mode, 0, fileName)
		if err != nil {
			errChan <- err
			return
		}

		err = checkResponse(stdout)
		if err != nil {
			errChan <- err
			return
		}

		_, err = fmt.Fprintln(stdin, "E")
		if err != nil {
			errChan <- err
			return
		}

		err = checkResponse(stdout)
		if err != nil {
			errChan <- err
			return
		}

	}()

	go func() {
		defer wg.Done()
		err = session.Run(fmt.Sprintf("scp -rt%s%q", scp.TimeOption, dstPath))
		if err != nil {
			errChan <- err
		}

		err = checkResponse(stdout)
		if err != nil {
			errChan <- err
			return
		}
	}()
	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (scp *SCP) Put(ctx context.Context, srcPath, dstPath string) error {
	resource, err := NewResource(srcPath)
	if err != nil {
		return err
	}
	var atime, mtime string
	if scp.KeepTime {
		atime, mtime = StatTimeV2(resource.FileInfo)
	}
	mode, err := FileModeV1(resource.Path)
	if err != nil {
		panic(err)
	}
	open, err := os.Open(resource.Path)
	if err != nil {
		panic(err)
	}
	return scp.put(ctx, dstPath, open, mode, resource.Size(), atime, mtime)
}

func (scp *SCP) put(ctx context.Context, dstPath string, in io.Reader, mode string, size int64, atime, mtime string) error {
	wg := sync.WaitGroup{}
	session, err := scp.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	errChan := make(chan error, 1)

	fileName := filepath.Base(dstPath)

	wg.Add(2)
	go func() {
		defer wg.Done()
		defer stdin.Close()

		if scp.KeepTime {
			// T+Mtime 0 Atime 0
			_, err = fmt.Fprintln(stdin, "T"+mtime, "0", atime, "0")
			if err != nil {
				errChan <- err
				return
			}

			err = checkResponse(stdout)
			if err != nil {
				errChan <- err
				return
			}
		}

		// C+MODE SIZE NAME
		_, err = fmt.Fprintln(stdin, "C"+mode, size, fileName)
		if err != nil {
			errChan <- err
			return
		}

		err = checkResponse(stdout)
		if err != nil {
			errChan <- err
			return
		}

		_, err = io.Copy(stdin, in)
		if err != nil {
			errChan <- err
			return
		}

		_, err = fmt.Fprint(stdin, "\x00")
		if err != nil {
			errChan <- err
			return
		}

		err = checkResponse(stdout)
		if err != nil {
			errChan <- err
			return
		}
		log.Printf("file:[%40s] size:[%15d]\n", fileName, size)
	}()

	go func() {
		defer wg.Done()
		err = session.Run(fmt.Sprintf("scp -t%s%q", scp.TimeOption, dstPath))
		if err != nil {
			errChan <- err
		}

		err = checkResponse(stdout)
		if err != nil {
			errChan <- err
			return
		}
	}()
	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (scp *SCP) Get(ctx context.Context, srcPath, dstPath string) error {
	session, err := scp.NewSession()
	if err != nil {
		return err
	}
	wg := &sync.WaitGroup{}
	errChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()

		stdin, err := session.StdinPipe()
		if err != nil {
			errChan <- err
			return
		}
		defer stdin.Close()

		stdout, err := session.StdoutPipe()
		if err != nil {
			errChan <- err
			return
		}

		err = session.Start(fmt.Sprintf("scp -f%s%q", scp.TimeOption, dstPath))
		if err != nil {
			errChan <- err
			return
		}

		err = ack(stdin)
		if err != nil {
			errChan <- err
			return
		}

		var attr Attr

		if scp.KeepTime {
			err = parseTime(stdout, &attr)
			if err != nil {
				errChan <- err
			}

			err = ack(stdin)
			if err != nil {
				errChan <- err
				return
			}
		}

		err = parseAttr(stdout, &attr)
		if err != nil {
			errChan <- err
		}

		err = ack(stdin)
		if err != nil {
			errChan <- err
			return
		}

		// create file
		in, err := os.Create(srcPath)
		if err != nil {
			errChan <- err
			return
		}

		err = os.Chmod(srcPath, attr.Mode)
		if err != nil {
			errChan <- err
			os.Remove(srcPath)
			return
		}

		if scp.KeepTime {
			err = os.Chtimes(srcPath, attr.Atime, attr.Mtime)
			if err != nil {
				errChan <- err
				os.Remove(srcPath)
				return
			}
		}

		err = parseContent(in, stdout, attr.Size)
		if err != nil {
			errChan <- err
			os.Remove(srcPath)
			return
		}

		err = ack(stdin)
		if err != nil {
			errChan <- err
			os.Remove(srcPath)
			return
		}

		err = session.Wait()
		if err != nil {
			errChan <- err
			os.Remove(srcPath)
			return
		}
		log.Printf("file:[%40s] size:[%15d]\n", path.Base(srcPath), attr.Size)
	}()
	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (scp *SCP) GetDir(localPath, remotePath string) error {
	session, err := scp.NewSession()
	if err != nil {
		return err
	}
	wg := &sync.WaitGroup{}
	errChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()

		stdin, err := session.StdinPipe()
		if err != nil {
			errChan <- err
			return
		}
		defer stdin.Close()

		stdout, err := session.StdoutPipe()
		if err != nil {
			errChan <- err
			return
		}

		err = session.Start(fmt.Sprintf("scp -f%s%q", scp.TimeOption, remotePath))
		if err != nil {
			errChan <- err
			return
		}

		err = ack(stdin)
		if err != nil {
			errChan <- err
			return
		}

		var attr Attr

		if scp.KeepTime {
			err = parseTime(stdout, &attr)
			if err != nil {
				errChan <- err
			}

			err = ack(stdin)
			if err != nil {
				errChan <- err
				return
			}
		}

		err = parseAttr(stdout, &attr)
		if err != nil {
			errChan <- err
		}

		err = ack(stdin)
		if err != nil {
			errChan <- err
			return
		}

		// mkdir dir
		err = os.Mkdir(localPath, attr.Mode)
		if err != nil {
			errChan <- err
			return
		}

		if scp.KeepTime {
			err = os.Chtimes(localPath, attr.Atime, attr.Mtime)
			if err != nil {
				errChan <- err
				os.Remove(localPath)
				return
			}
		}

		err = session.Wait()
		if err != nil {
			errChan <- err
			os.Remove(localPath)
			return
		}
	}()
	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (scp *SCP) GetAll(ctx context.Context, localPath, remotePath string) error {
	session, err := scp.NewSession()
	if err != nil {
		return err
	}
	wg := &sync.WaitGroup{}
	errChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()

		stdin, err := session.StdinPipe()
		if err != nil {
			errChan <- err
			return
		}
		defer stdin.Close()

		stdout, err := session.StdoutPipe()
		if err != nil {
			errChan <- err
			return
		}

		err = session.Start(fmt.Sprintf("scp -rf%s%q", scp.TimeOption, remotePath))
		if err != nil {
			errChan <- err
			return
		}

		err = ack(stdin)
		if err != nil {
			errChan <- err
			return
		}

		curLocalPath, curRemotePath := localPath, path.Dir(remotePath)

		for {
			attr, err := parseResponse(stdout)
			if err != nil && err.Error() == "EOF" {
				log.Infof("scp remote all success!!")
				break
			}
			if err != nil {
				errChan <- err
				return
			}
			if attr.Typ == T {

			} else if attr.Typ == C {
				curLocalPath = path.Join(curLocalPath, attr.Name)
				curRemotePath = path.Join(curRemotePath, attr.Name)
				log.Infof("file localPath:%s remotePath:%s", curLocalPath, curRemotePath)
				//log.Printf("file:[%40s] size:[%15s]\n", file.Name, file.Size)

				err = ack(stdin)
				if err != nil {
					errChan <- err
					return
				}

				in, err := os.Create(curLocalPath)
				if err != nil {
					errChan <- err
					return
				}

				err = os.Chmod(curLocalPath, attr.Mode)
				if err != nil {
					errChan <- err
					return
				}

				var cur int64
				for cur < attr.Size {
					readN, err := io.CopyN(in, stdout, attr.Size)
					if err != nil {
						errChan <- err
						return
					}
					cur += readN
				}

				buffer := make([]uint8, 1)
				n, err := stdout.Read(buffer)
				if err != nil {
					errChan <- err
					return
				}
				if n != 1 {
					errChan <- errors.New("check byte after read data")
					return
				}
				if buffer[0] != 0 {
					errChan <- errors.New("response attr fail")
					return
				}
				l, _ := path.Split(curLocalPath)
				r, _ := path.Split(curRemotePath)
				curLocalPath, curRemotePath = l[:len(l)-1], r[:len(r)-1]
			} else if attr.Typ == D {
				curLocalPath = path.Join(curLocalPath, attr.Name)
				curRemotePath = path.Join(curRemotePath, attr.Name)
				err := os.Mkdir(curLocalPath, attr.Mode)
				if err != nil {
					errChan <- err
					return
				}
				log.Infof("dir localPath:%s remotePath:%s", curLocalPath, curRemotePath)
			} else {
				l, _ := path.Split(curLocalPath)
				r, _ := path.Split(curRemotePath)
				curLocalPath, curRemotePath = l[:len(l)-1], r[:len(r)-1]
				log.Infof("E localPath:%s remotePath:%s", curLocalPath, curRemotePath)
			}
			err = ack(stdin)
			if err != nil {
				errChan <- err
				return
			}
		}

		err = session.Wait()
		if err != nil {
			errChan <- err
			os.Remove(localPath)
			return
		}
	}()
	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func wait(wg *sync.WaitGroup, ctx context.Context) error {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func checkResponse(out io.Reader) error {
	bytes := make([]uint8, 1)
	_, err := out.Read(bytes)
	//log.Infof("checkResponse:%d", n)
	if err != nil {
		return err
	}
	if bytes[0] == 0 {
		return nil
	}
	return errors.New("checkResponse fail")
}

func ack(in io.Writer) error {
	bytes := make([]uint8, 1)
	bytes[0] = 0
	n, err := in.Write(bytes)
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("ack fail")
	}
	return nil
}

func parseResponse(out io.Reader) (Attr, error) {
	var attr Attr

	bufferedReader := bufio.NewReader(out)
	message, err := bufferedReader.ReadString('\n')
	if err != nil {
		return attr, err
	}
	message = strings.ReplaceAll(message, "\n", "")
	parts := strings.Split(message, " ")
	commandTyp := string(parts[0][0])
	attr.Typ = commandTyp
	switch commandTyp {
	case T:
		err := attr.SetTime(parts[0][1:], parts[2])
		if err != nil {
			return attr, err
		}
	case C, D:
		err := attr.SetMode(parts[0][1:])
		if err != nil {
			return attr, err
		}
		err = attr.SetSize(parts[1])
		if err != nil {
			return attr, err
		}
		attr.Name = parts[2]
	case E:
		log.Infof("E")
	default:
		return attr, errors.New(fmt.Sprintf("parse steam fail message%s", message))
	}
	return attr, nil
}

func parseTime(out io.Reader, attr *Attr) error {
	bufferedReader := bufio.NewReader(out)
	message, err := bufferedReader.ReadString('\n')
	if err != nil {
		return err
	}
	message = strings.ReplaceAll(message, "\n", "")
	parts := strings.Split(message, " ")
	if len(parts) != 4 || (len(parts) > 0 && !strings.HasPrefix(parts[0], T)) {
		return errors.New(fmt.Sprintf("unable to parse message as time infos, message:%s", message))
	}

	err = attr.SetTime(parts[0][1:], parts[2])
	if err != nil {
		return err
	}
	return nil
}

func parseAttr(out io.Reader, attr *Attr) error {
	bufferedReader := bufio.NewReader(out)
	message, err := bufferedReader.ReadString('\n')
	if err != nil {
		return err
	}
	message = strings.ReplaceAll(message, "\n", "")
	parts := strings.Split(message, " ")
	if len(parts) != 3 || (len(parts) > 0 && !strings.HasPrefix(parts[0], C)) {
		return errors.New(fmt.Sprintf("unable to parse message as attr infos,message:%s", message))
	}

	err = attr.SetMode(parts[0][1:])
	if err != nil {
		return err
	}
	err = attr.SetSize(parts[1])
	if err != nil {
		return err
	}
	attr.Name = parts[2]
	return nil
}

func parseContent(in io.Writer, out io.Reader, size int64) error {
	var read int64
	for read < size {
		readN, err := io.CopyN(in, out, size)
		if err != nil {
			return err
		}
		if readN == 0 {
			return errors.New(fmt.Sprintf("parseContent fail readN:%d size:%d", readN, size))
		}
		read += readN
	}
	return nil
}
