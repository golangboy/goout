package main

import (
	"goout"
	"io"
	"net"
	"strconv"
)

func handleSocks5(conn net.Conn, buf []byte) {
	conn.Write([]byte{0x05, 0x00})
	conn.Read(buf)
	var host string
	var port string
	switch buf[3] {
	case 0x01:
		host = net.IP(buf[4:8]).String()
		port = strconv.Itoa(int(buf[8])<<8 + int(buf[9]))
	case 0x03:
		hostLen := buf[4]
		host = string(buf[5 : 5+hostLen])
		port = strconv.Itoa(int(buf[5+hostLen])<<8 + int(buf[5+hostLen+1]))
	case 0x04:
		host = net.IP(buf[4:20]).String()
		port = strconv.Itoa(int(buf[20])<<8 + int(buf[21]))
	default:
		goout.LogInfo("not socks5 connect")
	}
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	g := goout.GoOutCli{}
	err := g.Dial(server, host+":"+port)
	if err != nil {
		goout.LogInfo(host + ":" + port + " " + err.Error())
		return
	}
	go func() {
		defer func() {
			err := recover()
			if err != nil {
				goout.LogInfo(err)
			}
		}()
		io.Copy(&g, conn)
	}()
	io.Copy(conn, &g)
}
