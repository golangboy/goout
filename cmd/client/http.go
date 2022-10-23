package main

import (
	"goout"
	"io"
	"net"
	"strings"
)

func handleHttp(conn net.Conn, buf []byte) {
	//parse HTTP connect request
	headers := strings.Split(string(buf), "\r\n")
	//fmt.Println(headers)
	target := strings.Split(headers[0], " ")[1]
	//fmt.Println(target)
	authPass := true
	if authPass {
		conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	} else {
		conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n\r\n"))
	}
	g := goout.GoOutCli{}
	err := g.Dial(server, target)
	if err != nil {
		goout.LogInfo(target + " " + err.Error())
		return
	}
	go func() {
		defer func() {
			err := recover()
			if err != nil {
				goout.LogError(err)
			}
		}()
		io.Copy(&g, conn)
	}()
	io.Copy(conn, &g)
}
