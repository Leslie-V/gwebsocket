package server

import (
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/gobwas/ws"
	"github.com/panjf2000/gnet"
)

// HttpRequest ...
type HttpRequest struct {
	URI    []byte
	Header map[string]string
}

// GnetUpgraderConn ...
// protocol upgrader, coon interface
type GnetUpgraderConn struct {
	GnetConn gnet.Conn

	Closed            bool
	UniqId            string //连接的全局唯一id ?
	LastActiveTs      int64
	IsSuccessUpgraded bool
	Upgrader          *ws.Upgrader
	Req               *HttpRequest
	Context           interface{}
}

// Read
func (u GnetUpgraderConn) Read(b []byte) (n int, err error) {

	targetLength := len(b)

	if targetLength < 1 {
		return 0, nil
	}

	if u.GnetConn.BufferLength() >= targetLength {
		//buffer中数据够
		curNum, realData := u.GnetConn.ReadN(targetLength)

		u.GnetConn.ShiftN(curNum)

		n = curNum

		copy(b, realData) //数据拷贝

	} else {
		//buffer 中数据不够
		allData := u.GnetConn.Read()
		u.GnetConn.ResetBuffer() //数据已全部读出来

		copy(b, allData) //数据拷贝

		n = len(allData)
	}

	return n, nil
}

// Write ...
func (u GnetUpgraderConn) Write(b []byte) (n int, err error) {
	if err := u.GnetConn.AsyncWrite(b); err != nil {
		return -1, err
	}

	return len(b), nil
}

// UpdateActiveTsToNow ...
func (u *GnetUpgraderConn) UpdateActiveTsToNow() {
	u.LastActiveTs = time.Now().Unix()
}

// OnHeader ...
func (g *GnetUpgraderConn) OnHeader(key, value []byte) error {
	// log.Printf("ws OnHeader, key:%s, value:%s\n", string(key), string(value))
	g.Req.Header[string(key)] = string(value)
	return nil
}

// OnRequest ...
func (g *GnetUpgraderConn) OnRequest(uri []byte) error {
	// log.Printf("ws OnRequest: data uri: %v\n", string(uri))
	g.Req.URI = uri
	return nil
}

// NewDefaultUpgrader ...
func NewDefaultUpgrader(conn gnet.Conn) *GnetUpgraderConn {
	return &GnetUpgraderConn{
		GnetConn:          conn,
		Upgrader:          &defaultUpgrader,
		IsSuccessUpgraded: false,
	}
}

// NewEmptyUpgrader ...
func NewEmptyUpgrader(conn gnet.Conn) *GnetUpgraderConn {
	return &GnetUpgraderConn{
		GnetConn: conn,

		Upgrader:          &emptyUpgrader,
		IsSuccessUpgraded: false,
	}
}

// NewExtUpgrader ...
func NewExtUpgrader(gconn gnet.Conn) *GnetUpgraderConn {
	conn := &GnetUpgraderConn{
		GnetConn: gconn,
		Req: &HttpRequest{
			Header: make(map[string]string),
		},
		IsSuccessUpgraded: false,
	}

	conn.Upgrader = &ws.Upgrader{
		OnRequest: conn.OnRequest,
		OnHeader:  conn.OnHeader,
	}

	return conn
}

// Prepare handshake header writer from http.Header mapping.
var header = ws.HandshakeHeaderHTTP(http.Header{
	"X-Go-Version-CCLehui": []string{runtime.Version()},
})

//空的协议升级类
var emptyUpgrader = ws.Upgrader{}

//默认的协议升级处理类
var defaultUpgrader = ws.Upgrader{
	OnHost: func(host []byte) error {
		if string(host) == "github.com" {
			return nil
		}

		log.Printf("ws OnHost:%s\n", string(host))

		return nil
	},
	OnHeader: func(key, value []byte) error {
		log.Printf("ws OnHeader, key:%s, value:%s\n", string(key), string(value))

		return nil
	},
	OnBeforeUpgrade: func() (ws.HandshakeHeader, error) {
		log.Printf("ws OnBeforeUpgrade\n")
		return header, nil
	},
	OnRequest: func(uri []byte) error {
		log.Printf("ws OnRequest: data uri: %v\n", string(uri))
		return nil
	},
}
