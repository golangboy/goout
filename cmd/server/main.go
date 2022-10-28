package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"goout"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

var addr string
var trafficLog *os.File

const (
	SEND  = 1
	RECV  = 2
	CONN  = 3
	CLOSE = 4
)

type traffic struct {
	TotalSend   int64
	TotalRecv   int64
	Host        string
	IpAddr      string
	LastConn    string
	Description string
}
type goOutClient struct {
	Addr        string
	Description string
	Traffic     map[string]*traffic
}
type summaryTraffic struct {
	sync.Mutex
	TotalSend int64
	TotalRecv int64
	TotalConn int64
	Detail    map[string]*goOutClient
}

var summary summaryTraffic
var hostRecord = make(map[string]string)
var goOutClientList []reflect.Value
var updateIdx int

// recordTraffic is a function to record traffic
func recordTraffic(targetAddr string, goOutClientAddr string, dataLength int, trafficType int) {
	summary.Lock()
	defer summary.Unlock()

	//remote port
	goOutClientAddr = goOutClientAddr[:strings.LastIndex(goOutClientAddr, ":")]
	//target port
	targetPort := targetAddr[strings.LastIndex(targetAddr, ":"):]
	// lookup
	targetIpAddr, _ := net.LookupHost(targetAddr[:strings.LastIndex(targetAddr, ":")])

	targetDomain := targetAddr[:strings.LastIndex(targetAddr, ":")]

	byHost := false
	// record hostname
	if net.ParseIP(targetAddr[:strings.LastIndex(targetAddr, ":")]) == nil {
		byHost = true
		hostRecord[targetIpAddr[0]] = targetAddr[:strings.LastIndex(targetAddr, ":")]
	} else {
		// www.google.com
		if hostRecord[targetIpAddr[0]] != "" {
			targetDomain = hostRecord[targetIpAddr[0]]
		}
	}

	if summary.Detail == nil {
		summary.Detail = make(map[string]*goOutClient)
	}
	if summary.Detail[goOutClientAddr] == nil {
		summary.Detail[goOutClientAddr] = &goOutClient{
			Addr:        goOutClientAddr,
			Description: goout.QueryIp(goOutClientAddr),
			Traffic:     make(map[string]*traffic),
		}
	}
	if updateIdx == len(goOutClientList) {
		goOutClientList = reflect.ValueOf(summary.Detail).MapKeys()
		updateIdx = 0
	}
	if summary.Detail[goOutClientAddr].Traffic[targetDomain+targetPort] == nil {
		summary.Detail[goOutClientAddr].Traffic[targetDomain+targetPort] = &traffic{}
	}
	summary.Detail[goOutClientAddr].Traffic[targetDomain+targetPort].Host = targetDomain
	summary.Detail[goOutClientAddr].Traffic[targetDomain+targetPort].IpAddr = targetIpAddr[0]
	summary.Detail[goOutClientAddr].Traffic[targetDomain+targetPort].Description = goout.QueryIp(targetIpAddr[0])
	switch trafficType {
	case SEND:
		summary.Detail[goOutClientAddr].Traffic[targetDomain+targetPort].TotalSend += int64(dataLength)
		summary.TotalSend += int64(dataLength)
	case RECV:
		summary.Detail[goOutClientAddr].Traffic[targetDomain+targetPort].TotalRecv += int64(dataLength)
		summary.TotalRecv += int64(dataLength)
	case CONN:
		summary.Detail[goOutClientAddr].Traffic[targetDomain+targetPort].LastConn = time.Now().Format("2006-01-02 15:04:05")
		summary.TotalConn++
	case CLOSE:
		summary.TotalConn--
	}
	updateGooutAddr := goOutClientList[updateIdx].String()
	if summary.Detail[updateGooutAddr] != nil {
		if byHost && nil != summary.Detail[updateGooutAddr].Traffic[targetIpAddr[0]+targetPort] {
			summary.Detail[updateGooutAddr].Traffic[targetAddr].IpAddr = targetIpAddr[0]
			summary.Detail[updateGooutAddr].Traffic[targetAddr].Host = targetDomain
			summary.Detail[updateGooutAddr].Traffic[targetAddr].LastConn = time.Now().Format("2006-01-02 15:04:05")
			summary.Detail[updateGooutAddr].Traffic[targetAddr].TotalSend += summary.Detail[updateGooutAddr].Traffic[targetIpAddr[0]].TotalSend
			summary.Detail[updateGooutAddr].Traffic[targetAddr].TotalRecv += summary.Detail[updateGooutAddr].Traffic[targetIpAddr[0]].TotalRecv
			delete(summary.Detail[updateGooutAddr].Traffic, targetIpAddr[0]+targetPort)
			updateIdx++
		}
	}
}

func handleTCP(tcp *net.TCPConn) {
	var ioBuffer bytes.Buffer
	var tcpWithTarget *net.TCPConn
	var targetHost string
	defer func() {
		if err := recover(); err != nil {
			goout.LogError(err)
			goout.LogError(targetHost)
		}
	}()
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
			targetHost = string(req.Body)
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
						recordTraffic(target.RemoteAddr().String(), tcp.RemoteAddr().String(), 0, CLOSE)
						target.Close()
						proxyClient.Close()
						return
					}
					n, err = goout.WriteHttpResponse(proxyClient, buff[:n])
					if err != nil {
						recordTraffic(target.RemoteAddr().String(), tcp.RemoteAddr().String(), 0, CLOSE)
						target.Close()
						proxyClient.Close()
						return
					}
					recordTraffic(targetHost, proxyClient.RemoteAddr().String(), n, RECV)
				}
			}(tcpWithTarget, tcp)
		} else if path == "/send" {
			n, err := tcpWithTarget.Write(req.Body)
			recordTraffic(targetHost, tcp.RemoteAddr().String(), n, SEND)
			if err != nil {
				tcpWithTarget.Close()
				tcp.Close()
				return
			}
		} else if path == "/" {
			remoteAddr := tcp.RemoteAddr().String()
			goout.LogInfo(remoteAddr + "(" + goout.QueryIp(remoteAddr[:strings.LastIndex(remoteAddr, ":")]) + ")" + "-" + tcp.LocalAddr().String())
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
func startLog() {
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
}
func main() {
	flag.StringVar(&addr, "addr", ":80", "server bind address")
	flag.StringVar(&webAddr, "web", ":8080", "web server bind address")
	flag.Parse()
	go startWebServer()
	go startServer()
	go startLog()
	select {}
}
