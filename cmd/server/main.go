package main

import (
	"bytes"
	"flag"
	"github.com/goccy/go-json"
	"goout"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var addr string
var trafficLog *os.File

const (
	SEND = 1
	RECV = 2
	CONN = 3
)

type traffic struct {
	TotalSend int64
	TotalRecv int64
}
type summaryTraffic struct {
	sync.Mutex
	TotalSend int64
	TotalRecv int64
	TotalConn int64
	Detail    map[string]map[string]*traffic
}

var summary summaryTraffic

// recordTraffic is a function to record traffic
func recordTraffic(targetAddr string, goOutClientAddr string, dataLength int, trafficType int) {
	//remote port
	goOutClientAddr = goOutClientAddr[:strings.LastIndex(goOutClientAddr, ":")]
	summary.Lock()
	defer summary.Unlock()
	if summary.Detail == nil {
		summary.Detail = make(map[string]map[string]*traffic)
	}
	if summary.Detail[goOutClientAddr] == nil {
		summary.Detail[goOutClientAddr] = make(map[string]*traffic)
	}
	if summary.Detail[goOutClientAddr][targetAddr] == nil {
		summary.Detail[goOutClientAddr][targetAddr] = &traffic{}
	}
	switch trafficType {
	case SEND:
		summary.Detail[goOutClientAddr][targetAddr].TotalSend += int64(dataLength)
		summary.TotalSend += int64(dataLength)
	case RECV:
		summary.Detail[goOutClientAddr][targetAddr].TotalRecv += int64(dataLength)
		summary.TotalRecv += int64(dataLength)
	}
}

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
			recordTraffic(targetHost, tcp.RemoteAddr().String(), 0, CONN)
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
					recordTraffic(target.RemoteAddr().String(), proxyClient.RemoteAddr().String(), n, RECV)
				}
			}(tcpWithTarget, tcp)
		} else if path == "/send" {
			n, err := tcpWithTarget.Write(req.Body)
			recordTraffic(tcpWithTarget.RemoteAddr().String(), tcp.RemoteAddr().String(), n, SEND)
			if err != nil {
				tcpWithTarget.Close()
				tcp.Close()
				return
			}
		} else if path == "/" {
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
	flag.StringVar(&webAddr, "web", ":8080", "web server bind address")
	flag.Parse()
	go startWebServer()
	go startServer()
	go func() {
		fileName := time.Now().Format("2006-01-02-15-04-05") + ".json"
		// 1秒钟写一次
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				trafficLog, err := os.Create(fileName)
				if err != nil {
					return
				}
				summary.Lock()
				marshal, _ := json.Marshal(summary)
				// 格式化输出
				var out bytes.Buffer
				json.Indent(&out, marshal, "", "\t")
				trafficLog.Write(out.Bytes())
				//trafficLog.Write(marshal)
				summary.Unlock()
				trafficLog.Close()
			}
		}
	}()
	select {}
}
