package main

import (
	"crypto/tls"
	"fmt"
	"net"
)

func servePlain(config Config) {
	addr := fmt.Sprintf("%v:%v", config.Addr, config.PortPlain)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic("failed to open: " + err.Error())
	}

	serveLoop(config, ln)
}

func serveTLS(config Config) {
	addr := fmt.Sprintf("%v:%v", config.Addr, config.PortTLS)
	ln, err := tls.Listen("tcp", addr, config.tlsConfig)
	if err != nil {
		panic("failed to open: " + err.Error())
	}

	serveLoop(config, ln)
}

func serveLoop(config Config, listener net.Listener) {
	fmt.Printf("listening %s\n", listener.Addr().String())
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("accept failed: " + err.Error())
		}

		go RunSession(config, conn)
	}
}
