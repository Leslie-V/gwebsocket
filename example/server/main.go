package main

import (
	"fmt"
	"gwebsocket/server"
	"log"
	"strconv"
	"time"

	"github.com/gobwas/ws/wsutil"
)

//简单的echo server
func EchoDataHandler(param *server.DataHandlerParam) {

	log.Printf("server received message: opcode:%x, %s\n", param.OpCode, string(param.Request))

	// response := fmt.Sprintf("response is :%s, 当前时间:%s\n", string(param.Request), time.Now().Format("2006-01-02 15:04:05"))
	response := "kkk"

	//param.Writer.Write([]byte(response))

	//ws.WriteFrame(param.Writer, ws.NewTextFrame([]byte(response)))

	wsutil.WriteServerMessage(param.Writer, param.OpCode, []byte(response))

	return
}

type A struct {
	ConnManager *server.ConnManager
}

func (a A) connCloseHandler(conn *server.GnetUpgraderConn) {
	log.Printf("connCloseHandler(%v) call\n", conn)
	a.ConnManager.Free(conn.UniqId)
}

func (a A) connEstablishedHandler(conn *server.GnetUpgraderConn) error {
	log.Printf("connEstablishedHandler(%v) call\n", conn)
	conn.UniqId = string(conn.Req.URI[1:])
	a.ConnManager.Add(conn.UniqId, conn)
	return nil
}

func main() {

	port := 8081
	s := server.NewServer(strconv.Itoa(port))

	a := &A{
		ConnManager: server.NewConnManager(),
	}

	a.ConnManager.Start()

	s.ConnCloseHandler = a.connCloseHandler
	s.ConnEstablishedHandler = a.connEstablishedHandler

	go func() {
		for {
			fmt.Printf("num :%d\n", a.ConnManager.EstablishedNum)
			time.Sleep(time.Second * 1)
		}

	}()

	s.Start(EchoDataHandler)
}
