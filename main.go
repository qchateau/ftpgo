package main

import (
	"fmt"
	"net"
)

const (
	bindAddr = "127.0.0.1"
	bindPort = "10021"
)

func main() {
	fmt.Println("starting ...")

	ln, err := net.Listen("tcp", bindAddr+":"+bindPort)
	if err != nil {
		panic("failed to open: " + err.Error())
	}

	fmt.Printf("listening %s\n", ln.Addr().String())
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("accept failed: " + err.Error())
		}

		go RunSession(conn)
	}
}
