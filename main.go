package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"golang.org/x/crypto/ssh/terminal"
)

var help bool

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  ftpgo serve config-file-path")
	fmt.Println("      start FTP server")
	fmt.Println("  ftpgo genpass")
	fmt.Println("      generate a password hash")
}

func main() {
	flag.BoolVar(&help, "h", false, "")
	flag.BoolVar(&help, "help", false, "")
	flag.Parse()

	if help {
		usage()
		os.Exit(0)
	}

	command := ""
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	switch command {
	case "serve":
		serve()
	case "genpass":
		genpass()
	default:
		usage()
		os.Exit(1)
	}
}

func genpass() {
	fmt.Println("password [empty for no password]:")
	pass, err := terminal.ReadPassword(0)
	if err != nil {
		panic("failure: " + err.Error())
	}
	fmt.Println("confirm password:")
	passConfirm, err := terminal.ReadPassword(0)
	if err != nil {
		panic("failure: " + err.Error())
	}
	if string(pass) != string(passConfirm) {
		fmt.Println("passwords do not match")
		os.Exit(1)
	}

	hash, err := EncryptPassword(pass)
	if err != nil {
		panic("failure: " + err.Error())
	}

	fmt.Println("\n\n" + string(hash))
}

func serve() {
	if flag.NArg() != 2 {
		usage()
		os.Exit(1)
	}
	configPath := flag.Arg(1)

	config, err := LoadConfig(configPath)
	if err != nil {
		panic("cannot open config file: " + err.Error())
	}

	var wg sync.WaitGroup

	if config.PortPlain > 0 {
		wg.Add(1)
		go func() {
			servePlain(config)
			wg.Done()
		}()
	}

	if config.PortTLS > 0 && config.tlsConfig != nil {
		wg.Add(1)
		go func() {
			serveTLS(config)
			wg.Done()
		}()
	}

	wg.Wait()
}
