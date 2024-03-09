package scpw

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/vbauerster/mpb/v8"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var log = GetLogger("scpw")

type CommandType = string

var ONCE_LEN = 128 * 1024

const (
	C    CommandType = "C"
	D    CommandType = "D"
	E    CommandType = "E"
	T    CommandType = "T"
	NULL CommandType = "NULL"
)

type Context struct {
	Ctx context.Context
	Bar *mpb.Bar
}

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
	mode, err := ParseUnit32(str)
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
	var auth []ssh.AuthMethod
	if node.KeyPath != "" {
		privateKeyBytes, err := os.ReadFile(node.KeyPath)
		if err != nil {
			return nil, err
		}

		// Parse the private key
		privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(privateKey))
	} else {
		auth = append(auth, ssh.Password(node.Password))
	}
	config := &ssh.ClientConfig{
		User: node.User,
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Always accept server's public key.
			// In real world usage, don't do this! You should validate the key.
			return nil
		},
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

func (scp *SCP) SwitchScpwFunc(ctx Context, localPath, remotePath string, typ SCPWType) (err error) {
	excludeRootDir := false
	if typ == PUT {
		if localPath[len(localPath)-1] == '*' {
			excludeRootDir = true
			localPath = localPath[:len(localPath)-1]
		}
		stat, err1 := os.Stat(localPath)
		if err1 != nil {
			err = err1
			return
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
		localTmp := filepath.Join(filepath.Dir(localPath), uuid.NewString())
		last := remotePath[len(remotePath)-1]
		if last == '\\' || last == '/' {
			remotePath = remotePath[:len(remotePath)-1]
			if err = os.Mkdir(localTmp, os.FileMode(0755)); err != nil {
				return err
			}
			if err = scp.GetAll(ctx, localTmp, remotePath); err == nil {
				return scp.replaceDir(localTmp, localPath, remotePath)
			} else {
				return err
			}
		} else {
			if err = scp.Get(ctx, localTmp, remotePath); err == nil {
				return scp.replace(localTmp, localPath)
			} else {
				return err
			}
		}
	}
}

func (scp *SCP) replace(tmp, local string) error {
	newTmp := filepath.Join(filepath.Dir(local), uuid.NewString())
	if _, err := os.Stat(local); err == nil {
		if err = os.Rename(local, newTmp); err != nil {
			return err
		}
	}
	return os.Rename(tmp, local)
}

func (scp *SCP) replaceDir(tmp, local, remote string) error {
	dirname := filepath.Base(filepath.Clean(remote))
	old := filepath.Join(local, dirname)
	newTmp := filepath.Join(local, uuid.NewString())
	if _, err := os.Stat(old); err == nil {
		if err = os.Rename(old, newTmp); err != nil {
			return err
		}
	}
	if err := os.Rename(filepath.Join(tmp, dirname), old); err != nil {
		return err
	}
	return os.RemoveAll(tmp)
}

func (scp *SCP) PutAllExcludeRoot(ctx Context, srcPath, dstPath string) error {
	var err error
	child, err := StatDirChild(srcPath)
	if err != nil {
		return err
	}
	for _, entry := range child {
		l, r := filepath.Join(srcPath, entry.Name()), filepath.Join(dstPath, entry.Name())
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

func (scp *SCP) PutAll(ctx Context, srcPath, dstPath string) error {
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
		if err = WalkTree(ctx, scpCh, srcPath, srcPath, dstPath); err != nil {
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
					sizeNum, err1 := ParseInt64(size)
					if err1 != nil {
						errChan <- err
						return
					}
					open, err1 := os.Open(file.LocalPath)
					err1 = parseContent(ctx.Bar, stdin, open, sizeNum)
					open.Close()
					if err1 != nil {
						errChan <- err1
						return
					}

					if _, err = fmt.Fprint(stdin, "\x00"); err != nil {
						errChan <- err
						return
					}

					if err = checkResponse(stdout); err != nil {
						errChan <- err
						return
					}
				}
				//fmt.Printf("    file:[%40s] size:[%15s]\n", file.Name, file.Size)
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
		if e := session.Run(fmt.Sprintf("scp -rt%s%q", scp.TimeOption, dstPath)); e != nil {
			errChan <- e
			return
		}

		if e := checkResponse(stdout); e != nil {
			errChan <- e
			return
		}
	}()

	wg.Wait()
	close(errChan)
	for err = range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func WalkTree(ctx Context, scpChan *scpChan, rootParent, root, dstPath string) error {
	select {
	case <-ctx.Ctx.Done():
		return ctx.Ctx.Err()
	default:
		child, name, mode, atime, mtime, _, err := StatDir(root)
		if err != nil {
			return err
		}
		scpChan.fileChan <- NewFile(name, root, dstPath, mode, atime, mtime, "0", true)
		var dirs []os.DirEntry
		for _, obj := range child {
			if !obj.IsDir() {
				filePath := filepath.Join(root, obj.Name())
				if cName, cMode, cSize, cAtime, cMtime, cErr := StatFile(filePath); cErr != nil {
					return fmt.Errorf("WalkTree failed! root: %s e: %v", root, cErr)
				} else {
					scpChan.fileChan <- NewFile(cName, filePath, filepath.Join(dstPath, cName), cMode, cAtime, cMtime, cSize, false)
				}
			} else {
				dirs = append(dirs, obj)
			}
		}

		if err != nil {
			return err
		}
		for _, dir := range dirs {
			if err = WalkTree(ctx, scpChan, rootParent, filepath.Join(root, dir.Name()), filepath.Join(dstPath, dir.Name())); err != nil {
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

func (scp *SCP) Put(ctx Context, srcPath, dstPath string) error {
	stat, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return errors.New(fmt.Sprintf("local:[%s] is dir", srcPath))
	}
	var atime, mtime string
	if scp.KeepTime {
		atime, mtime = StatTimeV2(stat)
	}
	mode, err := FileModeV1(srcPath)
	if err != nil {
		return err
	}
	open, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	return scp.put(ctx, dstPath, open, mode, stat.Size(), atime, mtime)
}

func (scp *SCP) put(ctx Context, dstPath string, in io.Reader, mode string, size int64, atime, mtime string) error {
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
		//fmt.Printf("    file:[%40s] size:[%15d]\n", fileName, size)
	}()

	var err2 error
	go func() {
		defer wg.Done()
		err2 = session.Run(fmt.Sprintf("scp -t%s%q", scp.TimeOption, dstPath))
		if err2 != nil {
			errChan <- err2
			return
		}

		err2 = checkResponse(stdout)
		if err2 != nil {
			errChan <- err2
			return
		}
	}()
	wg.Wait()
	close(errChan)
	for err = range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (scp *SCP) Get(ctx Context, srcPath, dstPath string) error {
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

		stdin, err1 := session.StdinPipe()
		if err1 != nil {
			errChan <- err1
			return
		}
		defer stdin.Close()

		stdout, err1 := session.StdoutPipe()
		if err1 != nil {
			errChan <- err1
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
		in, err1 := os.Create(srcPath)
		if err1 != nil {
			errChan <- err1
			return
		}

		if err = os.Chmod(srcPath, attr.Mode); err != nil {
			os.Remove(srcPath)
			errChan <- err
			return
		}

		if scp.KeepTime {
			err = os.Chtimes(srcPath, attr.Atime, attr.Mtime)
			if err != nil {
				os.Remove(srcPath)
				errChan <- err
				return
			}
		}

		if err = parseContent(ctx.Bar, in, stdout, attr.Size); err != nil {
			os.Remove(srcPath)
			errChan <- err
			return
		}

		if err = ack(stdin); err != nil {
			os.Remove(srcPath)
			errChan <- err
			return
		}

		if err = checkResponse(stdout); err != nil {
			os.Remove(srcPath)
			errChan <- err
			return
		}

		err = session.Wait()
		if err != nil {
			errChan <- err
			os.Remove(srcPath)
			return
		}
		//fmt.Printf("    file:[%40s] size:[%15d]\n", filepath.Base(srcPath), attr.Size)
	}()
	wg.Wait()
	close(errChan)
	for err = range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (scp *SCP) GetAll(ctx Context, localPath, remotePath string) error {
	session, err := scp.NewSession()
	if err != nil {
		return err
	}
	wg := &sync.WaitGroup{}
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		stdin, e := session.StdinPipe()
		if e != nil {
			errChan <- e
			return
		}
		defer stdin.Close()

		stdout, e := session.StdoutPipe()
		if e != nil {
			errChan <- e
			return
		}

		if e = session.Start(fmt.Sprintf("scp -rf%s%q", scp.TimeOption, remotePath)); e != nil {
			errChan <- e
			return
		}

		if e = ack(stdin); e != nil {
			errChan <- e
			return
		}

		curLocal, curRemote := localPath, filepath.Dir(filepath.Clean(remotePath))
		for {
			var attr Attr
			if scp.KeepTime {
				if e = parseMeta(stdout, &attr); e != nil {
					if e.Error() == "EOF" {
						break
					}
					errChan <- e
					return
				}

				if e = ack(stdin); e != nil {
					errChan <- e
					return
				}
			}

			if attr.Typ != E {
				if e = parseMeta(stdout, &attr); e != nil {
					errChan <- e
					return
				}

				if e = ack(stdin); e != nil {
					errChan <- e
					return
				}
				curLocal = filepath.Join(curLocal, attr.Name)
				curRemote = filepath.Join(curRemote, attr.Name)
			}

			var in *os.File
			if attr.Typ == C {
				// create file
				in, e = os.Create(curLocal)
				if e != nil {
					errChan <- e
					return
				}

				if e = os.Chmod(curLocal, attr.Mode); e != nil {
					os.Remove(curLocal)
					errChan <- err
				}
				//fmt.Printf("    file:[%40s] size:[%15d]\n", attr.Name, attr.Size)

			} else if attr.Typ == D {
				// mkdir dir
				if e = os.Mkdir(curLocal, attr.Mode); e != nil {
					errChan <- e
					return
				}
				//fmt.Printf("    file:[%40s] size:[%15d]\n", attr.Name, attr.Size)

			} else if attr.Typ == E {
				// cd ../
				curLocal = filepath.Dir(filepath.Clean(curLocal))
				curRemote = filepath.Dir(filepath.Clean(curRemote))
			} else {
				// maybe error
				errChan <- errors.New(fmt.Sprintf("invalid type:[%s]", attr.Typ))
				return
			}

			if scp.KeepTime && attr.Typ != E {
				if e = os.Chtimes(curLocal, attr.Atime, attr.Mtime); e != nil {
					os.Remove(curLocal)
					errChan <- e
					return
				}
			}
			if attr.Typ == C {
				if e = parseContent(ctx.Bar, in, stdout, attr.Size); e != nil {
					os.Remove(curLocal)
					errChan <- e
					return
				}

				if e = ack(stdin); e != nil {
					os.Remove(curLocal)
					errChan <- e
					return
				}

				if e = checkResponse(stdout); e != nil {

					os.Remove(curLocal)
					errChan <- e
					return
				}
				curLocal = filepath.Dir(curLocal)
				curRemote = filepath.Dir(curRemote)
			}
		}
		if e = session.Wait(); e != nil {
			os.Remove(curLocal)
			errChan <- e
			return
		}
	}()
	wg.Wait()
	close(errChan)
	for err = range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func checkResponse(out io.Reader) error {
	bytes := make([]uint8, 1)
	if _, err := out.Read(bytes); err != nil {
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
	if n, err := in.Write(bytes); err != nil {
		return err
	} else if n != 1 {
		return fmt.Errorf("ack fail")
	}
	return nil
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

func parseContent(bar *mpb.Bar, in io.Writer, out io.Reader, size int64) error {
	var read int64
	for read < size {
		if readN, err := io.CopyN(in, out, size); err != nil {
			return err
		} else {
			read += readN
			bar.IncrBy(int(readN))
			bar.SetTotal(bar.Current()+readN, false)
		}
	}
	return nil
}
