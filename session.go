package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

// RunSession creates and runs a FTP session from a net.Conn
func RunSession(conn net.Conn) {
	fmt.Printf("new session from %v\n", conn.RemoteAddr().String())
	defer conn.Close()

	s := session{
		piConn:     conn,
		dtListener: nil,
		user:       "",
		loggedIn:   false,
		quitting:   false,
		pwd:        "/",
	}

	s.run()
}

type session struct {
	piConn     net.Conn
	dtListener net.Listener
	user       string
	loggedIn   bool
	quitting   bool
	dataType   string
	pwd        string
}

func (s *session) run() {

	const lineMaxSize = 4096
	piConnReader := bufio.NewReaderSize(s.piConn, lineMaxSize)

	s.writeResponse(readyForNewUser)

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
			s.writeResponse(syntaxError)
			continue
		}

		line := string(buf)
		fmt.Printf("-> %v\n", line)
		response := s.handleFtpCommand(line)

		s.writeResponse(response)

		if s.quitting {
			// no data connection yet, just close
			break
		}
	}

	// close listener if any
	if s.dtListener != nil {
		s.dtListener.Close()
	}
}

// TODO: handle error where writeResponse is called
func (s *session) writeResponse(response string) error {
	fmt.Printf("<- %v\n", response)
	_, err := s.piConn.Write([]byte(response + "\r\n"))
	return err
}

func (s *session) writeData(data []byte) error {
	if s.dtListener == nil {
		fmt.Printf("tried to write data but no active DT listener")
		return errors.New("can't write data: no listener")
	}

	conn, err := s.dtListener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()

	conn.Write(data)
	return nil
}

func (s *session) handleFtpCommand(line string) string {
	sliced := strings.SplitN(line, " ", 2)
	command := sliced[0]
	arguments := ""
	if len(sliced) >= 2 {
		arguments = sliced[1]
	}

	switch command {
	case "USER":
		return s.handleUser(arguments)
	case "QUIT":
		return s.handleQuit()
	case "PORT":
		// Standard says we should implement this
		// but I don't intend on supporting active mode
		return commandNotImplemented
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
	case "LIST":
		return s.handleList(arguments)
	case "RETR":
		return commandNotImplemented
	case "STOR":
		return commandNotImplemented
	case "NOOP":
		return commandOk
	default:
		return commandNotImplemented
	}
}

func (s *session) handleUser(user string) string {
	fmt.Printf("login as '%v'\n", user)
	s.user = user
	s.loggedIn = true
	return loggedIn
}

func (s *session) handleQuit() string {
	s.quitting = true
	s.loggedIn = false
	s.user = ""
	return loggedOut
}

func (s *session) handleType(arguments string) string {
	if !s.loggedIn {
		return notLoggedIn
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
		return parameterNotImplemented
	}

	// Check the type is supported
	if dataType != ascii && dataType != image {
		return parameterNotImplemented

	}

	s.dataType = dataType
	return commandOk
}

func (s *session) handleMode(mode string) string {
	if !s.loggedIn {
		return notLoggedIn
	}

	switch mode {
	case stream:
		return commandOk
	default:
		return parameterNotImplemented
	}
}

func (s *session) handleStructure(structure string) string {
	if !s.loggedIn {
		return notLoggedIn
	}

	switch structure {
	case file:
		return commandOk
	// TODO: handle record ?
	default:
		return parameterNotImplemented
	}
}

func (s *session) handlePassive() string {
	if !s.loggedIn {
		return notLoggedIn
	}

	var err error
	s.dtListener, err = net.Listen("tcp4", bindAddr+":")
	if err != nil {
		fmt.Printf("failed to open data port: " + err.Error())
		s.quitting = true
		return closing
	}
	tcpAddr, ok := s.dtListener.Addr().(*net.TCPAddr)
	if !ok {
		fmt.Printf("unexpected listener address: %v", s.dtListener.Addr())
		s.quitting = true
		return closing
	}

	fmt.Printf("new DTP listener: %s\n", tcpAddr.String())

	ip := tcpAddr.IP.To4()
	h1, h2, h3, h4 := ip[0], ip[1], ip[2], ip[3]
	p1, p2 := (tcpAddr.Port >> 8 & 0xff), tcpAddr.Port&0xff

	return fmt.Sprintf(passiveMode, h1, h2, h3, h4, p1, p2)
}

func (s *session) handlePwd() string {
	if !s.loggedIn {
		return notLoggedIn
	}

	return fmt.Sprintf(currentDir, s.pwd)
}

func (s *session) handleCwd(path string) string {
	if !s.loggedIn {
		return notLoggedIn
	}

	if len(path) == 0 || path[:1] != "/" {
		path = "/" + path
	}

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		fmt.Printf("path %v is not a directory\n", path)
		return actionNotTaken
	}

	s.pwd = path
	return actionOk
}

func (s *session) handleList(path string) string {
	if !s.loggedIn {
		return notLoggedIn
	}

	if path == "" {
		path = s.pwd
	}

	// TODO: verify how we should handle symlinks
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Printf("stat(%v) failed: %v\n", path, err.Error())
		return actionNotTaken
	}

	var files []os.FileInfo
	if fileInfo.IsDir() {
		files, err = ioutil.ReadDir(path)
		if err != nil {
			fmt.Printf("ReadDir(%v) failed: %v\n", path, err.Error())
			return actionNotTaken
		}
	} else {
		files = []os.FileInfo{fileInfo}
	}

	resp := ""
	for _, file := range files {
		user := "unknown" // TODO: implement this
		group := "unknown"

		line := fmt.Sprintf(
			"%v %v %v %v %v %v\n",
			file.Mode().String(),
			user,
			group,
			file.Size(),
			file.ModTime().Format("2006-01-02T15:04:05"),
			file.Name())
		resp += line
	}

	s.writeResponse(okOpenDt)
	err = s.writeData([]byte(resp))
	if err != nil {
		fmt.Printf("error during data write: %v\n", err.Error())
	}

	return okCloseDt
}
