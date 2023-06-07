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
	C    CommandType = "C"
	D    CommandType = "D"
	E    CommandType = "E"
	T    CommandType = "T"
	NULL CommandType = "NULL"
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
			if excludeRootDir {
				return scp.PutAllExcludeRoot(ctx, localPath, remotePath)
			} else {
				return scp.PutAll(ctx, localPath, remotePath)
			}
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

func (scp *SCP) PutAllExcludeRoot(ctx context.Context, srcPath, dstPath string) error {
	var err error
	child, err := StatDirChild(srcPath)
	if err != nil {
		return err
	}
	for _, entry := range child {
		l, r := path.Join(srcPath, entry.Name()), path.Join(dstPath, entry.Name())
		if entry.IsDir() {
			err = scp.PutAll(ctx, l, r)
		} else {
			err = scp.Put(ctx, l, r)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (scp *SCP) PutAll(ctx context.Context, srcPath, dstPath string) error {
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
	errChan := make(chan error, 3)
	scpCh := &scpChan{fileChan: make(chan File), exitChan: make(chan struct{}), closeChan: make(chan struct{})}
	go func() {
		err := WalkTree(ctx, scpCh, srcPath, srcPath, dstPath)
		if err != nil {
			errChan <- err
			return
		}
	}()

	go func() {
		defer stdin.Close()
		defer wg.Done()
	loop:
		for {
			select {
			case file := <-scpCh.fileChan:
				//log.Errorf("remote:%s", file.RemotePath)
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
				fmt.Printf("file:[%40s] size:[%15s]\n", file.Name, file.Size)
			case <-scpCh.exitChan:
				//log.Errorf("E")
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
			return
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

func WalkTree(ctx context.Context, scpChan *scpChan, rootParent, root, dstPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		child, name, mode, atime, mtime, _, err := StatDir(root)
		if err != nil {
			return err
		}
		scpChan.fileChan <- NewFile(name, root, dstPath, mode, atime, mtime, "0", true)
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
			err := WalkTree(ctx, scpChan, rootParent, path.Join(root, dir.Name()), path.Join(dstPath, dir.Name()))
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

func (scp *SCP) PutDir(ctx context.Context, localPath, remotePath string) error {
	_, mode, atime, mtime, isDir, err := StatDirMeta(localPath)
	if err != nil {
		return err
	}
	if !isDir {
		return errors.New(fmt.Sprintf("local:[%s] is not dir", localPath))
	}
	return scp.putDir(ctx, remotePath, mode, atime, mtime)
}

func (scp *SCP) putDir(ctx context.Context, dstPath string, mode string, atime, mtime string) error {
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
	errChan := make(chan error, 2)

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
			return
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
	if resource.IsDir() {
		return errors.New(fmt.Sprintf("local:[%s] is dir", srcPath))
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
	errChan := make(chan error, 2)

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
		fmt.Printf("file:[%40s] size:[%15d]\n", fileName, size)
	}()

	go func() {
		defer wg.Done()
		err = session.Run(fmt.Sprintf("scp -t%s%q", scp.TimeOption, dstPath))
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
	defer session.Close()
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
			err = parseMeta(stdout, &attr)
			if err != nil {
				errChan <- err
				return
			}

			err = ack(stdin)
			if err != nil {
				errChan <- err
				return
			}
		}

		err = parseMeta(stdout, &attr)
		if err != nil {
			errChan <- err
			return
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

		err = checkResponse(stdout)
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
		fmt.Printf("file:[%40s] size:[%15d]\n", path.Base(srcPath), attr.Size)
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
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		var err error
		defer func() {
			errChan <- err
			wg.Done()
		}()
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

		curLocal, curRemote := localPath, path.Dir(path.Clean(remotePath))
		for {
			var attr Attr
			if scp.KeepTime {
				err = parseMeta(stdout, &attr)
				if err != nil {
					if err.Error() == "EOF" {
						log.Debug("GetAll success")
						break
					}
					errChan <- err
					return
				}

				err = ack(stdin)
				if err != nil {
					errChan <- err
					return
				}
			}

			if attr.Typ != E {
				err = parseMeta(stdout, &attr)
				if err != nil {
					errChan <- err
					return
				}

				err = ack(stdin)
				if err != nil {
					errChan <- err
					return
				}
				curLocal = path.Join(curLocal, attr.Name)
				curRemote = path.Join(curRemote, attr.Name)
			}

			var in *os.File
			if attr.Typ == C {
				// create file
				in, err = os.Create(curLocal)
				if err != nil {
					errChan <- err
					return
				}

				err = os.Chmod(curLocal, attr.Mode)
				if err != nil {
					errChan <- err
					os.Remove(curLocal)
					return
				}

			} else if attr.Typ == D {
				// mkdir dir
				err := os.Mkdir(curLocal, attr.Mode)
				if err != nil {
					errChan <- err
					return
				}

			} else if attr.Typ == E {
				// cd ../
				curLocal = path.Dir(path.Clean(curLocal))
				curRemote = path.Dir(path.Clean(curRemote))
			} else {
				// maybe error
				errChan <- errors.New(fmt.Sprintf("invalid type:[%s]", attr.Typ))
				return
			}

			if scp.KeepTime && attr.Typ != E {
				err = os.Chtimes(curLocal, attr.Atime, attr.Mtime)
				if err != nil {
					errChan <- err
					os.Remove(curLocal)
					return
				}
			}
			if attr.Typ == C {
				err = parseContent(in, stdout, attr.Size)
				if err != nil {
					errChan <- err
					os.Remove(curLocal)
					return
				}

				err = ack(stdin)
				if err != nil {
					errChan <- err
					os.Remove(curLocal)
					return
				}

				err = checkResponse(stdout)
				if err != nil {
					errChan <- err
					os.Remove(curLocal)
					return
				}
				curLocal = path.Dir(curLocal)
				curRemote = path.Dir(curRemote)
			}
		}
		err = session.Wait()
		if err != nil {
			errChan <- err
			os.Remove(curLocal)
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
	if err != nil {
		return err
	}
	if bytes[0] == 0 {
		return nil
	} else {
		bufferedReader := bufio.NewReader(out)
		msg, err := bufferedReader.ReadString('\n')
		if err != nil {
			return err
		}
		return errors.New(msg)
	}
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

func parseResponse(out io.Reader) error {
	bytes := make([]uint8, 1)
	n, err := out.Read(bytes)
	if err != nil {
		return err
	}
	log.Infof("n:%d", n)
	if int(bytes[0]) != 0 {
		bufferedReader := bufio.NewReader(out)
		message, err := bufferedReader.ReadString('\n')
		if err != nil {
			return errors.New(message)
		}
	}
	return nil
}

func parseAttr1(out io.Reader) (Attr, error) {
	var attr Attr

	bufferedReader := bufio.NewReader(out)
	message, err := bufferedReader.ReadString('\n')
	if err != nil {
		return attr, err
	}
	log.Infof(message)
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
	default:
		return attr, errors.New(fmt.Sprintf("parse steam fail message%s", message))
	}
	return attr, nil
}

func parseMeta(out io.Reader, attr *Attr) error {
	bufferedReader := bufio.NewReader(out)
	message, err := bufferedReader.ReadString('\n')
	if err != nil {
		return err
	}
	message = strings.ReplaceAll(message, "\n", "")
	parts := strings.Split(message, " ")
	attr.Typ = parseCommandType(message)
	if attr.Typ == C || attr.Typ == D {
		err = attr.SetMode(parts[0][1:])
		if err != nil {
			return err
		}
		err = attr.SetSize(parts[1])
		if err != nil {
			return err
		}
		attr.Name = parts[2]
	} else if attr.Typ == E {

	} else if attr.Typ == T {
		err = attr.SetTime(parts[0][1:], parts[2])
		if err != nil {
			return err
		}
	} else {
		return errors.New(fmt.Sprintf("invalid commandType message:[%s]", message))
	}
	return nil
}

//func parseTime(out io.Reader, attr *Attr) error {
//	bufferedReader := bufio.NewReader(out)
//	message, err := bufferedReader.ReadString('\n')
//	if err != nil {
//		return err
//	}
//	message = strings.ReplaceAll(message, "\n", "")
//	parts := strings.Split(message, " ")
//	if len(parts) != 4 || (len(parts) > 0 && !strings.HasPrefix(parts[0], T)) {
//		return errors.New(fmt.Sprintf("unable to parse message as time infos, message:%s", message))
//	}
//
//	attr.Typ = parseCommandType(message)
//	if attr.Typ == NULL {
//		return errors.New(fmt.Sprintf("invalid commandType:[%s]", message))
//	}
//	err = attr.SetTime(parts[0][1:], parts[2])
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func parseAttr(out io.Reader, attr *Attr) error {
//	bufferedReader := bufio.NewReader(out)
//	message, err := bufferedReader.ReadString('\n')
//	if err != nil {
//		return err
//	}
//	message = strings.ReplaceAll(message, "\n", "")
//	parts := strings.Split(message, " ")
//	if len(parts) != 3 || (len(parts) > 0 && (!strings.HasPrefix(parts[0], C) && !strings.HasPrefix(parts[0], D))) {
//		return errors.New(fmt.Sprintf("unable to parse message as attr infos,message:%s", message))
//	}
//
//	err = attr.SetMode(parts[0][1:])
//	if err != nil {
//		return err
//	}
//	err = attr.SetSize(parts[1])
//	if err != nil {
//		return err
//	}
//	attr.Name = parts[2]
//	attr.Typ = parseCommandType(message)
//	if attr.Typ == NULL {
//		return errors.New(fmt.Sprintf("invalid commandType:[%s]", message))
//	}
//	return nil
//}

func parseCommandType(s string) CommandType {
	b := s[0]
	if b == 'T' {
		return T
	} else if b == 'C' {
		return C
	} else if b == 'D' {
		return D
	} else if b == 'E' {
		return E
	} else {
		return NULL
	}
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
