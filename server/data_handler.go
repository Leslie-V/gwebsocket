package server

import (
	"io"

	"github.com/gobwas/ws"
)

type DataHandlerParam struct {
	Request []byte

	OpCode ws.OpCode

	Writer io.Writer

	WSConn *GnetUpgraderConn //升级后的连接

	Server *WebSocketServer
}

type DataHandler func(param *DataHandlerParam)
