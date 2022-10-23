package goout

import (
	"bytes"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

type GoOutCli struct {
	conn        *net.TCPConn
	readBuffer  bytes.Buffer
	writeBuffer bytes.Buffer
}

func (g *GoOutCli) Dial(gooutServer string, target string) error {
	var buff bytes.Buffer
	tcpAddr, _ := net.ResolveTCPAddr("tcp", gooutServer)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}
	g.conn = conn
	WriteHttpRequest(conn, "/conn", []byte(target))
	_, ok := ParseHttpResponse(conn, &buff)
	if false == ok {
		return errors.New("Dial LogError")
	}
	return nil
}
func (g *GoOutCli) Read(b []byte) (n int, err error) {
	r, ok := ParseHttpResponse(g.conn, &g.readBuffer)
	if false == ok {
		return 0, errors.New("ParseHttpResponse LogError")
	}
	return copy(b, r.Body), nil
}
func (g *GoOutCli) Write(b []byte) (n int, err error) {
	return WriteHttpRequest(g.conn, "/send", b)
}
func (g *GoOutCli) Close() error {
	return g.conn.Close()
}

type httpReq struct {
	Url  string
	Raw  string
	Body []byte
}
type httpResponse struct {
	Raw  string
	Body []byte
}

func ParseHttpResponse(reader net.Conn, ioBuffer *bytes.Buffer) (httpResponse, bool) {
	var ret httpResponse
	var buff bytes.Buffer
	var contentLength int
	var tmp [10485]byte
	timeOut := time.Duration(13 * time.Second)
	_ = timeOut
	for {
		var n int
		var err error
		if ioBuffer.Len() > 0 {
			n, err = ioBuffer.Read(tmp[:])
		} else {
			//reader.SetReadDeadline(time.Now().Add(timeOut))
			n, err = reader.Read(tmp[:])
			if err != nil {
				return ret, false
			}
		}

		buff.Write(tmp[:n])
		//Http Header End Flag
		str := buff.String()
		pos := strings.Index(str, "\r\n\r\n")
		if pos != -1 {
			ret.Raw = str[:pos]
			break
		}
	}
	//Get Content-Length
	str := buff.String()
	pos := strings.Index(str, "Content-Length")
	if pos == -1 {
		return ret, false
	}
	for j := pos; j < len(str); j++ {
		if str[j] == '\r' {
			break
		} else if str[j] >= '0' && str[j] <= '9' {
			contentLength = contentLength*10 + (int(str[j] - '0'))
		}
	}
	var needRs int
	curHttpTotal := len(ret.Raw) + 4 + contentLength
	if buff.Len() > curHttpTotal {
		ioBuffer.Write(buff.Bytes()[curHttpTotal:])
		ret.Body = buff.Bytes()[len(ret.Raw)+4 : curHttpTotal]
		return ret, true
	}
	needRs = (curHttpTotal) - buff.Len()
	for needRs > 0 {
		var minSize int
		if needRs > len(tmp) {
			minSize = len(tmp)
		} else {
			minSize = needRs
		}
		//reader.SetReadDeadline(time.Now().Add(timeOut))
		n, err := reader.Read(tmp[:minSize])

		if err != nil {
			return ret, false
		}
		buff.Write(tmp[:n])
		needRs = needRs - n
		if needRs == 0 {
			break
		}

	}
	ret.Body = buff.Bytes()[len(ret.Raw)+4:]
	return ret, true
}
func ParseHttpRequest(reader net.Conn, ioBuffer *bytes.Buffer) (httpReq, bool) {
	var ret httpReq
	var buff bytes.Buffer
	var contentLength int
	var tmp [10485]byte
	timeOut := time.Duration(13 * time.Second)
	_ = timeOut
	for {
		var n int
		var err error
		if ioBuffer.Len() > 0 {
			n, err = ioBuffer.Read(tmp[:])
		} else {
			//reader.SetReadDeadline(time.Now().Add(timeOut))
			n, err = reader.Read(tmp[:])
			if err != nil {
				return ret, false
			}
		}

		buff.Write(tmp[:n])
		//Http Header End Flag
		str := buff.String()
		pos := strings.Index(str, "\r\n\r\n")
		if pos != -1 {
			ret.Raw = str[:pos]

			spl := strings.Split(ret.Raw, "\r\n")
			spt := strings.Split(spl[0], " ")
			if len(spt) < 2 {
				return ret, false
			}
			ret.Url = spt[1]
			break
		}
	}
	//Get Content-Length
	str := buff.String()
	pos := strings.Index(str, "Content-Length")
	if pos == -1 {
		return ret, true
	}
	for j := pos; j < len(str); j++ {
		if str[j] == '\r' {
			break
		} else if str[j] >= '0' && str[j] <= '9' {
			contentLength = contentLength*10 + (int(str[j] - '0'))
		}
	}
	var needRs int
	curHttpTotal := len(ret.Raw) + 4 + contentLength
	if buff.Len() > curHttpTotal {
		ioBuffer.Write(buff.Bytes()[curHttpTotal:])
		ret.Body = buff.Bytes()[len(ret.Raw)+4 : curHttpTotal]
		return ret, true
	}
	needRs = (curHttpTotal) - buff.Len()
	for needRs > 0 {
		var minSize int
		if needRs > len(tmp) {
			minSize = len(tmp)
		} else {
			minSize = needRs
		}
		//reader.SetReadDeadline(time.Now().Add(timeOut))
		n, err := reader.Read(tmp[:minSize])

		if err != nil {
			return ret, false
		}
		buff.Write(tmp[:n])
		needRs = needRs - n
		if needRs == 0 {
			break
		}
	}
	ret.Body = buff.Bytes()[len(ret.Raw)+4:]
	return ret, true
}
func WriteHttpRequest(tcp *net.TCPConn, path string, data []byte) (int, error) {
	payload := "POST XXX HTTP/1.1\r\nConnection: keep-alive\r\nContent-Length: YYY\r\nContent-Type: application/octet-stream\r\n\r\n"
	payload = strings.ReplaceAll(payload, "XXX", path)
	payload = strings.ReplaceAll(payload, "YYY", strconv.Itoa(len(data)))
	tcp.Write([]byte(payload))
	return tcp.Write(data)
}
func WriteHttpResponse(tcp *net.TCPConn, data []byte) (int, error) {
	payload := "HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: xxx\r\n\r\n"
	payload = strings.ReplaceAll(payload, "xxx", strconv.Itoa(len(data)))
	tcp.Write([]byte(payload))
	return tcp.Write(data)
}
func WriteHttpResponseWithCt(tcp *net.TCPConn, data []byte, contentType string) (int, error) {
	payload := "HTTP/1.1 200 OK\r\nContent-Type: yyy\r\nContent-Length: xxx\r\n\r\n"
	payload = strings.ReplaceAll(payload, "xxx", strconv.Itoa(len(data)))
	payload = strings.ReplaceAll(payload, "yyy", contentType)
	tcp.Write([]byte(payload))
	return tcp.Write(data)
}
