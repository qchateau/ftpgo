package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// RunSession creates and runs a FTP session from a net.Conn
func RunSession(config Config, conn net.Conn) {
	fmt.Printf("new session from %v\n", conn.RemoteAddr().String())
	defer conn.Close()

	s := Session{
		piConn:  conn,
		jailDir: config.Jail,
		workDir: config.Jail,
		config:  config,
	}

	s.run()
}

// Session stores all information relative to a FTP session
type Session struct {
	piConn      net.Conn
	dtpListener net.Listener
	userDtpAddr net.TCPAddr
	config      Config
	loggedIn    bool
	quitting    bool
	passive     bool
	dataType    string
	jailDir     string
	workDir     string
	renameFrom  string
}

func (s *Session) run() {
	const lineMaxSize = 4096
	piConnReader := bufio.NewReaderSize(s.piConn, lineMaxSize)

	s.writeResponse(code220)

	for {
		buf, isPrefix, err := piConnReader.ReadLine()
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error: " + err.Error())
			}
			return
		}
		if isPrefix {
			fmt.Println("line too long")
			s.writeResponse(code500)
			continue
		}

		line := string(buf)
		fmt.Printf("-> %v\n", line)
		response := s.handleFtpCommand(line)

		s.writeResponse(response)

		if s.quitting {
			break
		}
	}

	// close listener if any
	if s.dtpListener != nil {
		s.dtpListener.Close()
	}
}

// TODO: handle error where writeResponse is called
func (s *Session) writeResponse(response string) (err error) {
	fmt.Printf("<- %v\n", response)
	data := []byte(response + "\r\n")
	n, err := s.piConn.Write(data)
	if n != len(data) {
		err = errors.New("incomplete write")
	}
	return
}

func (s *Session) createListener(resetIfExists bool) (err error) {
	if !resetIfExists && s.dtpListener != nil {
		return nil
	}

	if s.dtpListener != nil {
		s.dtpListener.Close()
	}

	s.dtpListener, err = net.Listen("tcp4", s.config.Addr+":")
	if err == nil {
		fmt.Printf("new DTP listener: %s\n", s.dtpListener.Addr().String())
	}
	return
}

func (s *Session) dialDtp() (net.Conn, error) {
	if s.passive {
		if s.dtpListener == nil {
			fmt.Printf("tried to write data but no active DT listener")
			return nil, errors.New("can't write data: no listener")
		}

		return s.dtpListener.Accept()
	}

	return net.DialTCP("tcp4", nil, &s.userDtpAddr)
}

func (s *Session) simpleWriteDtp(data []byte) string {
	conn, err := s.dialDtp()
	if err != nil {
		fmt.Printf("cannot connect to DTP: %v\n", err.Error())
		return code425
	}
	defer conn.Close()

	n, err := conn.Write(data)
	if n != len(data) {
		fmt.Printf("DTP write too short\n")
		return code426
	}
	if err != nil {
		fmt.Printf("error during DTP write: %v\n", err.Error())
		return code426
	}

	return code226
}

func (s *Session) realpath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.workDir, path)
	} else {
		// always clean the input path to remove .. shenanigans
		path = filepath.Clean(path)
	}

	realpath, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = realpath
	}

	if !filepath.HasPrefix(path, s.jailDir) {
		// trying to get out of jail
		return "", errors.New("path not allowed")
	}
	return path, nil
}

func (s *Session) getFileList(path string) (files []os.FileInfo, err error) {
	if path, err = s.realpath(path); err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Printf("stat(%v) failed: %v\n", path, err.Error())
		return
	}

	if fileInfo.IsDir() {
		files, err = ioutil.ReadDir(path)
		if err != nil {
			fmt.Printf("ReadDir(%v) failed: %v\n", path, err.Error())
			return
		}
	} else {
		files = []os.FileInfo{fileInfo}
	}

	return
}

func (s *Session) deletePath(path string, allowDir bool, allowFile bool) error {
	path, err := s.realpath(path)
	if err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Stat(%v) failed: %v\n", path, err.Error())
		return err
	}

	if (info.IsDir() && !allowDir) || (!info.IsDir() && !allowFile) {
		return errors.New("not allowed")
	}

	err = os.Remove(path)
	if err != nil {
		fmt.Printf("Remove(%v) failed: %v\n", path, err.Error())
		return err
	}

	return nil
}

func (s *Session) handleFtpCommand(line string) string {
	sliced := strings.SplitN(line, " ", 2)
	command := sliced[0]
	arguments := ""
	if len(sliced) >= 2 {
		arguments = sliced[1]
	}

	switch command {
	case "USER":
		return s.handleUser(arguments)
	case "PASS":
		return s.handlePassword(arguments)
	case "QUIT":
		return s.handleQuit()
	case "REIN":
		return s.handleReinit()
	case "PORT":
		return s.handlePort(arguments)
	case "PASV":
		return s.handlePassive()
	case "TYPE":
		return s.handleType(arguments)
	case "MODE":
		return s.handleMode(arguments)
	case "STRU":
		return s.handleStructure(arguments)
	case "PWD":
		return s.handlePwd()
	case "CWD":
		return s.handleCwd(arguments)
	case "RMD":
		return s.handleRmd(arguments)
	case "MKD":
		return s.handleMkd(arguments)
	case "DELE":
		return s.handleDelete(arguments)
	case "CDUP":
		return s.handleCdup()
	case "LIST":
		return s.handleList(arguments)
	case "NLST":
		return s.handleNlst(arguments)
	case "RNFR":
		return s.handleRenameFrom(arguments)
	case "RNTO":
		return s.handleRenameTo(arguments)
	case "RETR":
		return s.handleRetrieve(arguments)
	case "STOR":
		return s.handleStore(arguments, true)
	case "APPE":
		return s.handleStore(arguments, false)
	case "NOOP":
		return code200
	case "ACCT", "ALLO", "SITE":
		return code202
	default:
		return code502
	}
}

func (s *Session) handleUser(user string) string {
	fmt.Printf("user '%v'\n", user)
	s.loggedIn = false

	if !s.config.AllowAnyUser() && user != s.config.Login {
		return code530
	}

	if s.config.PasswordRequired() {
		return code331
	}

	s.loggedIn = true
	return code230
}

func (s *Session) handlePassword(pass string) string {
	s.loggedIn = false

	if !s.config.VerifyPassword(pass) {
		fmt.Printf("bad password\n")
		return code530
	}

	s.loggedIn = true
	return code230
}

func (s *Session) handleQuit() string {
	s.quitting = true
	s.loggedIn = false
	return code221
}

func (s *Session) handleReinit() string {
	s.loggedIn = false
	s.workDir = "/"
	return code220
}

func (s *Session) handleType(arguments string) string {
	if !s.loggedIn {
		return code530
	}

	parameters := strings.SplitN(arguments, " ", 2)
	dataType := parameters[0]
	formatControl := ""
	if len(parameters) >= 2 {
		formatControl = parameters[1]
	}

	// We don't care about format control, only accept
	// the default (nonPrint) or unspecified
	if formatControl != "" && formatControl != nonPrint {
		return code504
	}

	// Check the type is supported
	if dataType != ascii && dataType != image {
		return code504

	}

	s.dataType = dataType
	return code200
}

func (s *Session) handleMode(mode string) string {
	if !s.loggedIn {
		return code530
	}

	switch mode {
	case stream:
		return code200
	default:
		return code504
	}
}

func (s *Session) handleStructure(structure string) string {
	if !s.loggedIn {
		return code530
	}

	switch structure {
	case file:
		return code200
	default:
		return code504
	}
}

func (s *Session) handlePort(addr string) string {
	if !s.loggedIn {
		return code530
	}

	var h1, h2, h3, h4 byte
	var p1, p2 int
	n, err := fmt.Sscanf(addr, "%d,%d,%d,%d,%d,%d", &h1, &h2, &h3, &h4, &p1, &p2)

	if err != nil || n != 6 {
		fmt.Printf("bad client DTP addr\n")
		return code501
	}

	s.userDtpAddr.Port = p1<<8 + p2
	s.userDtpAddr.IP = []byte{h1, h2, h3, h4}
	s.passive = false

	return code200
}

func (s *Session) handlePassive() string {
	if !s.loggedIn {
		return code530
	}

	if err := s.createListener(false); err != nil {
		fmt.Printf("failed to open data port: " + err.Error())
		s.quitting = true
		return code421
	}

	tcpAddr, ok := s.dtpListener.Addr().(*net.TCPAddr)
	if !ok {
		fmt.Printf("unexpected listener address: %v", s.dtpListener.Addr())
		s.quitting = true
		return code421
	}

	ip := tcpAddr.IP.To4()
	h1, h2, h3, h4 := ip[0], ip[1], ip[2], ip[3]
	p1, p2 := (tcpAddr.Port >> 8 & 0xff), tcpAddr.Port&0xff
	s.passive = true

	return fmt.Sprintf(code227, h1, h2, h3, h4, p1, p2)
}

func (s *Session) handlePwd() string {
	if !s.loggedIn {
		return code530
	}

	return fmt.Sprintf(code257, s.workDir)
}

func (s *Session) handleCwd(pathname string) string {
	if !s.loggedIn {
		return code530
	}

	pathname, err := s.realpath(pathname)
	if err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return code550
	}

	if info, err := os.Stat(pathname); err != nil || !info.IsDir() {
		fmt.Printf("path %v is not a directory\n", pathname)
		return code550
	}

	s.workDir = pathname
	return code250
}

func (s *Session) handleCdup() string {
	if !s.loggedIn {
		return code530
	}

	newWorkDir, err := s.realpath(filepath.Join(s.workDir, ".."))
	if err != nil || newWorkDir == s.workDir {
		return code550
	}

	s.workDir = newWorkDir
	return code200
}

func (s *Session) handleList(path string) string {
	if !s.loggedIn {
		return code530
	}

	files, err := s.getFileList(path)
	if err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return code450
	}

	resp := ""
	for _, file := range files {
		user := "unknown" // TODO: implement this
		group := "unknown"

		line := fmt.Sprintf(
			"%v 1 %v %v %v %v %v\r\n",
			file.Mode().String(),
			user,
			group,
			file.Size(),
			file.ModTime().Format("Jan 01 2006"),
			file.Name())
		resp += line
	}

	s.writeResponse(code150)
	return s.simpleWriteDtp([]byte(resp))
}

func (s *Session) handleNlst(path string) string {
	if !s.loggedIn {
		return code530
	}

	files, err := s.getFileList(path)
	if err != nil {
		return code450
	}

	resp := ""
	for _, file := range files {
		resp += file.Name() + "\r\n"
	}

	s.writeResponse(code150)
	return s.simpleWriteDtp([]byte(resp))
}

func (s *Session) handleRetrieve(path string) string {
	if !s.loggedIn {
		return code530
	}

	path, err := s.realpath(path)
	if err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return code450
	}

	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Open(%v) failed: %v\n", path, err.Error())
		return code450
	}

	conn, err := s.dialDtp()
	if err != nil {
		fmt.Printf("cannot connect to DTP: %v\n", err.Error())
		return code425
	}
	defer conn.Close()

	buf := make([]byte, 4096)
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			return code226
		} else if err != nil {
			fmt.Printf("read error: %v\n", err.Error())
			return code426
		}

		written, werr := conn.Write(buf[:n])
		if werr != nil {
			fmt.Printf("write error: %v\n", werr.Error())
			return code426
		}
		if n != written {
			fmt.Printf("write too short\n")
			return code426
		}
	}
}

func (s *Session) handleStore(path string, truncate bool) string {
	if !s.loggedIn {
		return code530
	}

	path, err := s.realpath(path)
	if err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return code553
	}

	flags := os.O_RDWR | os.O_CREATE
	if truncate {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}
	file, err := os.OpenFile(path, flags, 0666)
	if err != nil {
		fmt.Printf("Create(%v) failed: %v\n", path, err.Error())
		return code553
	}

	s.writeResponse(code150)

	conn, err := s.dialDtp()
	if err != nil {
		fmt.Printf("cannot connect to DTP: %v\n", err.Error())
		return code425
	}
	defer conn.Close()

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err == io.EOF {
			return code226
		} else if err != nil {
			fmt.Printf("read error: %v\n", err.Error())
			return code426
		}

		written, werr := file.Write(buf[:n])
		if werr != nil {
			fmt.Printf("write error: %v\n", werr.Error())
			return code426
		}
		if n != written {
			fmt.Printf("write too short\n")
			return code426
		}
	}

}

func (s *Session) handleMkd(path string) string {
	if !s.loggedIn {
		return code530
	}

	path, err := s.realpath(path)
	if err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return code550
	}

	err = os.Mkdir(path, os.ModePerm)
	if err != nil {
		fmt.Printf("Mkdir(%v) failed: %v\n", path, err.Error())
		return code550
	}
	return fmt.Sprintf(code257created, path)
}

func (s *Session) handleRmd(path string) string {
	if !s.loggedIn {
		return code530
	}

	err := s.deletePath(path, true, false)
	if err != nil {
		return code550
	}
	return code250
}

func (s *Session) handleDelete(path string) string {
	if !s.loggedIn {
		return code530
	}

	err := s.deletePath(path, false, true)
	if err != nil {
		return code550
	}
	return code250
}

func (s *Session) handleRenameFrom(path string) string {
	if !s.loggedIn {
		return code530
	}

	path, err := s.realpath(path)
	if err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return code550
	}

	_, err = os.Stat(path)
	if err != nil {
		fmt.Printf("Stat(%v) failed: %v\n", path, err.Error())
		return code550
	}

	s.renameFrom = path
	return code350
}

func (s *Session) handleRenameTo(path string) string {
	if !s.loggedIn {
		return code530
	}

	path, err := s.realpath(path)
	if err != nil {
		fmt.Printf("failed: %v\n", err.Error())
		return code553
	}

	err = os.Rename(s.renameFrom, path)
	if err != nil {
		fmt.Printf("Rename(%v, %v) failed: %v\n",
			s.renameFrom, path, err.Error())
		return code553
	}
	return code250
}
