// scp.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Client struct {
	IP           string //IP地址
	User         string //用户名
	Dir          string
	Passwd       string
	Home         string
	Port         int //端口号
	*sftp.Client     //ssh客户端
}

func newClient() *Client {
	return &Client{
		IP:     "192.168.0.76",
		User:   "root",
		Passwd: "jhadmin",
		Port:   22,
	}
}

func (c *Client) Title() string {
	return c.User + "@" + c.IP
}

func (c *Client) isLinkClose() bool {
	if c.Client == nil {
		return true
	}

	if _, e := c.Getwd(); e != nil {
		c.Client = nil
		return true
	}

	return false
}
func (c *Client) newLink() *sftp.Client {
	client, err := c.Connect()
	if err != nil {
		log.Fatal("Link:", err)
		return nil
	}
	c.Client = client

	return client
}
func (c *Client) IsDir(path string) bool {
	info, err := c.Stat(path)
	if err == nil && info.IsDir() {
		return true
	}
	return false
}

func (c *Client) IsFile(path string) bool {
	info, err := c.Stat(path)
	if err == nil && !info.IsDir() {
		return true
	}
	return false
}

func (c *Client) IsExist(path string) bool {
	_, err := c.Stat(path)
	return err == nil
}
func readKeyFile(path string) (ssh.Signer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(b)
}
func (c Client) Connect() (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		client       *sftp.Client
		err          error
	)
	user, password, host, port := c.User, c.Passwd, c.IP, c.Port
	// get auth method
	auth = make([]ssh.AuthMethod, 0, 1)
	if password != "" {
		auth = append(auth, ssh.Password(password))
	} else {
		keys := []ssh.Signer{}
		if Agent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			signers, err := agent.NewClient(Agent).Signers()
			if err == nil {
				keys = append(keys, signers...)
			}
		}

		hname, _ := os.Hostname()
		rsa := filepath.Join(c.Home, ".ssh", "id_rsa."+hname)
		pk, err := readKeyFile(rsa)
		if err == nil {
			keys = append(keys, pk)
		}

		rsa = filepath.Join(c.Home, ".ssh", "id_rsa")
		pk, err = readKeyFile(rsa)
		if err == nil {
			keys = append(keys, pk)
		}

		if len(keys) > 0 {
			auth = append(auth, ssh.PublicKeys(keys...))
		} else {
			return nil, err
		}
	}

	clientConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		Timeout:         60 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// connet to ssh
	addr = fmt.Sprintf("%s:%d", host, port)
	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create sftp client
	if client, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) Upload(local string, remote string) (err error) {
	info, err := os.Stat(local)
	if err != nil {
		return errors.New("Upload(" + local + "):" + err.Error())
	}
	if info.IsDir() {
		return c.UploadDir(local, remote)
	}

	return c.UploadFile(local, remote)
}

func (c *Client) UploadFile(localFile, remote string) error {
	info, err := os.Stat(localFile)
	if err != nil || info.IsDir() {
		return errors.New("Upload File(" + localFile + ") is directory, ignore...")
	}

	l, err := os.Open(localFile)
	if err != nil {
		return errors.New("Upload Open(" + localFile + "):" + err.Error())
	}
	defer l.Close()

	var remoteFile, remoteDir string
	if info, err = c.Stat(remote); err == nil && info.IsDir() || remote[len(remote)-1] == '/' {
		remoteDir = remote
		remoteFile = filepath.ToSlash(filepath.Join(remote, filepath.Base(localFile)))
	} else {
		remoteDir = filepath.ToSlash(filepath.Dir(remote))
		remoteFile = remote
	}

	info, _ = os.Stat(localFile)
	log.Printf("Upload: %s(%s) --> %s\n", localFile, Size(info.Size()), remoteFile)

	if _, err := c.Stat(remoteDir); err != nil {
		log.Println("Mkdir:", remoteDir)
		c.MkdirAll(remoteDir)
	}

	r, err := c.Create(remoteFile)
	if err != nil || c.isLinkClose() {
		c.newLink()
		r, err = c.Create(remoteFile)
		if err != nil {
			return errors.New("Upload Create " + remoteFile + ": " + err.Error())
		}
	}

	_, err = io.Copy(r, l)

	return err
}

// UploadDir files without checking diff status
func (c *Client) UploadDir(localDir string, remoteDir string) (err error) {
	log.Println("UploadDir", localDir, "-->", remoteDir)

	rootLocal := filepath.Dir(localDir)
	if c.IsFile(remoteDir) {
		log.Println("Remove File:", remoteDir)
		c.Remove(remoteDir)
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("err:", err)
			return err
		}

		relSrc, err := filepath.Rel(rootLocal, path)
		if err != nil {
			return err
		}
		finalDst := filepath.Join(remoteDir, relSrc)
		finalDst = filepath.ToSlash(finalDst)
		if info.IsDir() {
			if c.IsExist(finalDst) {
				return nil
			}
			err := c.MkdirAll(finalDst)
			if err != nil {
				log.Println("Mkdir failed:", err)
			}
		} else {
			return c.UploadFile(path, finalDst)
		}
		return nil

	}
	return filepath.Walk(localDir, walkFunc)
}
func (c *Client) MkdirAll(dirpath string) error {
	parentDir := filepath.ToSlash(filepath.Dir(dirpath))
	_, err := c.Stat(parentDir)
	if err != nil {
		if err.Error() == "file does not exist" {
			err := c.MkdirAll(parentDir)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if c.isLinkClose() {
		if c.newLink() == nil {
			return nil
		}
	}

	err = c.Mkdir(filepath.ToSlash(dirpath))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Download(remote string, local string) (err error) {
	if c.IsDir(remote) {
		return c.downloadDir(remote, local)

	}
	return c.downloadFile(remote, local)

}

// downloadFile a file from the remote server like cp
func (c *Client) downloadFile(remoteFile, local string) error {
	if !c.IsFile(remoteFile) {
		return errors.New("Download File(" + remoteFile + ") is not exist or directory, ignore...")
	}

	var localFile, localDir string
	if info, err := os.Stat(local); err == nil && info.IsDir() {
		localDir = local
		localFile = filepath.Join(local, filepath.Base(remoteFile))
	} else {
		localDir = filepath.Dir(local)
		localFile = local

	}

	info, _ := c.Stat(remoteFile)
	log.Printf("Download: %s <-- %s(%s)", localFile, remoteFile, Size(info.Size()))

	if _, err := os.Stat(local); err != nil {
		if err = os.MkdirAll(localDir, os.ModePerm); err != nil {
			log.Println("MkdirAll", err)
			return err
		}
	}

	r, err := c.Open(remoteFile)
	if err != nil {
		return err
	}
	defer r.Close()

	l, err := os.Create(localFile)
	if err != nil {
		return err
	}
	defer l.Close()

	_, err = io.Copy(l, r)
	return err
}

func (c *Client) downloadDir(remote, local string) error {
	var localDir, remoteDir string

	if !c.IsDir(remote) {
		return errors.New("Download Dir(" + remote + ") is not exist")
	}

	remoteDir = remote

	if info, err := os.Stat(local); err != nil && !info.IsDir() {
		localDir = local
	} else {
		localDir = path.Join(local, path.Base(remote))
	}

	walker := c.Walk(remoteDir)

	for walker.Step() {
		if err := walker.Err(); err != nil {
			log.Println(err)
			continue
		}

		info := walker.Stat()

		relPath, err := filepath.Rel(remoteDir, walker.Path())
		if err != nil {
			return err
		}

		localPath := filepath.ToSlash(filepath.Join(localDir, relPath))

		localInfo, err := os.Stat(localPath)
		if os.IsExist(err) {
			if localInfo.IsDir() {
				if info.IsDir() {
					continue
				}

				err = os.RemoveAll(localPath)
				if err != nil {
					return err
				}
			} else if info.IsDir() {
				err = os.Remove(localPath)
				if err != nil {
					return err
				}
			}
		}

		if info.IsDir() {
			err = os.MkdirAll(localPath, os.ModePerm)
			if err != nil {
				return err
			}

			continue
		}

		c.downloadFile(walker.Path(), localPath)

	}
	return nil
}

const (
	UP_LOAD = iota
	DOWN_LOAD
)

func UpOrDown(l, r string) (int, string) {
	if strings.Index(l, ":") == -1 && strings.Index(r, ":") != -1 {
		return UP_LOAD, r
	}

	if strings.Index(l, ":") != -1 && strings.Index(r, ":") == -1 {
		return DOWN_LOAD, l
	}

	return -1, ""
}

func createClient(s string, pw string) *Client {
	var (
		u, ip, p, home string
	)
	if i := strings.Index(s, ":"); i != -1 {
		p = s[i+1:]
		ip = s[:i]
		if j := strings.Index(ip, "@"); j != -1 {
			u = ip[:j]
			ip = ip[j+1:]
		}
	}
	if u == "" {
		cu, _ := user.Current()
		u = cu.Username
		home = cu.HomeDir
	} else if pw == "" {

		cu, _ := user.Lookup(u)
		u = cu.Username
		home = cu.HomeDir

	}
	return &Client{
		IP:   ip,
		User: u,
		Dir:  p,
		Home: home,
		Port: 22,
	}
}

var (
	p  int
	pw string
	e  error
)

func main() {
	flag.IntVar(&p, "p", 22, "ssh port")
	flag.StringVar(&pw, "a", "", "user pasword")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Println("Usage: scp [-a passwd] [-p port] [[user@]host1:]file1 [[user@]host2:]file2")
		os.Exit(-1)
	}

	local := flag.Args()[0]
	remote := flag.Args()[1]

	opcode, path := UpOrDown(local, remote)
	c := createClient(path, pw)
	c.Port = p
	c.Passwd = pw
	//fmt.Println(c.User)
	c.newLink()

	switch opcode {
	case UP_LOAD:
		log.Println("Upload...")
		e = c.Upload(local, c.Dir)
	case DOWN_LOAD:
		log.Println("Download...")
		e = c.Download(c.Dir, remote)
	default:
		log.Println("Usage: scp [[user@]host1:]file1  [[user@]host2:]file2")
		os.Exit(-1)
	}
	if e != nil {
		log.Println(e)
	} else {
		log.Println("Successful")
	}
}
func Size(size int64) string {
	s := float64(size)
	d := "B"

	if s > 1024 {
		s = s / 1024.0
		d = "K"
	}

	if s > 1024 {
		s = s / 1024.0
		d = "M"
	}
	if s > 1024 {
		s = s / 1024.0
		d = "G"
	}
	return fmt.Sprintf("%.2f %s", s, d)
}
