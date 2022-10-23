package main

import (
	"bytes"
	"flag"
	"goout"
	"net"
)

var addr string

func handleTCP(tcp *net.TCPConn) {
	var ioBuffer bytes.Buffer
	var tcpWithTarget *net.TCPConn
	for {
		req, ok := goout.ParseHttpRequest(tcp, &ioBuffer)
		if !ok {
			tcp.Close()
			if tcpWithTarget != nil {
				tcpWithTarget.Close()
			}
			return
		}
		path := req.Url
		if path == "/conn" {
			targetHost := string(req.Body)
			tcpAddr, err := net.ResolveTCPAddr("tcp4", targetHost)
			if err != nil {
				return
			}

			//repeat connect
			if tcpWithTarget != nil {
				tcpWithTarget.Close()
				return
			}
			tcpWithTarget, err = net.DialTCP("tcp4", nil, tcpAddr)
			if err != nil {
				return
			}
			_, err = goout.WriteHttpResponse(tcp, []byte("Done"))
			if err != nil {
				return
			}

			//Recv from remote
			go func(target *net.TCPConn, proxyClient *net.TCPConn) {
				for {
					var buff [10485]byte
					//target.SetReadDeadline(time.Now().Add(time.Second * 300))
					n, err := target.Read(buff[:])
					if err != nil {
						target.Close()
						proxyClient.Close()
						return
					}
					n, err = goout.WriteHttpResponse(proxyClient, buff[:n])
					if err != nil {
						target.Close()
						proxyClient.Close()
						return
					}
				}
			}(tcpWithTarget, tcp)
		} else if path == "/send" {
			_, err := tcpWithTarget.Write(req.Body)
			if err != nil {
				tcpWithTarget.Close()
				tcp.Close()
				return
			}
		} else if path == "/" {
			// 输出tcp对面客户端的地址
			goout.LogInfo(tcp.RemoteAddr().String() + "-" + tcp.LocalAddr().String())
			_, err := goout.WriteHttpResponseWithCt(tcp, []byte("Hello,GFW"), "text/plain; charset=utf-8")
			if err != nil {
				tcpWithTarget.Close()
				return
			}
			return
		}
	}
}

func startServer() {
	ta, _ := net.ResolveTCPAddr("tcp4", addr)
	tc, err := net.ListenTCP("tcp4", ta)
	if err != nil {
		goout.LogError(err)
		panic(err)
	}
	for {
		client, err := tc.AcceptTCP()
		if err == nil && client != nil {
			go handleTCP(client)
		} else if client != nil {
			client.Close()
		}
	}
}
func main() {
	flag.StringVar(&addr, "addr", ":80", "server bind address")
	flag.Parse()
	startServer()
}
