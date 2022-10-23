package main

import (
	"flag"
	"goout"
	"net"
)

var addr string
var server string

func handleConnection(conn net.Conn) {
	defer func() {
		err := recover()
		if err != nil {
			goout.LogInfo(conn.RemoteAddr().String())
			goout.LogError(err)
		}
	}()
	defer conn.Close()
	// parse socks5 connect
	buf := make([]byte, 1024)
	conn.Read(buf)
	if buf[0] == 5 {
		handleSocks5(conn, buf)
	} else if buf[0] == 'C' {
		handleHttp(conn, buf)
	} else {
		goout.LogInfo("not supported protocol")
	}
}
func startClient() {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		goout.LogInfo(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			goout.LogInfo("accept " + err.Error())
		}
		go handleConnection(conn)
	}
}

func main() {
	flag.StringVar(&addr, "addr", ":8080", "address to listen to(socks5、http)")
	flag.StringVar(&server, "server", "localhost:80", "goout server")
	flag.Parse()
	startClient()
}
