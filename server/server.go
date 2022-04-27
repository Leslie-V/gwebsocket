package server

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/ants"
	"github.com/panjf2000/gnet"
)

type ConnCloseHandleFunc func(wsConn *GnetUpgraderConn)
type ConnEstablishedHandleFunc func(wsConn *GnetUpgraderConn) error

// WebSocketServer ...
type WebSocketServer struct {
	*gnet.EventServer

	Addr    string
	Handler DataHandler

	WorkerPool    *ants.Pool
	connTimeWheel *TimeWheel

	ConnCloseHandler       ConnCloseHandleFunc
	ConnEstablishedHandler ConnEstablishedHandleFunc
}

// NewServer ...
func NewServer(addr string) *WebSocketServer {

	options := ants.Options{ExpiryDuration: time.Second * 10, Nonblocking: true}
	defaultAntsPool, _ := ants.NewPool(DefaultAntsPoolSize, ants.WithOptions(options))

	server := &WebSocketServer{}

	server.Addr = addr
	server.WorkerPool = defaultAntsPool

	server.connTimeWheel = NewTimeWheel(time.Second, int(ConnMaxIdleSeconds), timeWheelJob)

	return server
}

// WebsocoketFrameCodec ...
type WebsocketFrameCodec struct {
}

// Encode ...
func (wc *WebsocketFrameCodec) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	return buf, nil
}

// Decode ...
func (wc *WebsocketFrameCodec) Decode(c gnet.Conn) ([]byte, error) {
	buf := c.Read()
	if len(buf) == 0 {
		return nil, nil
	}
	return buf, nil
}

// Start ...
func (server *WebSocketServer) Start(handler DataHandler) {
	server.Handler = handler
	log.Fatal(gnet.Serve(server, fmt.Sprintf("tcp://:%s", server.Addr), gnet.WithMulticore(true), gnet.WithCodec(&WebsocketFrameCodec{})))
}

// OnInitComplete ...
func (server *WebSocketServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("websocket server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)

	server.connTimeWheel.Start()

	return
}

// OnOpened ...
func (server *WebSocketServer) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	return
}

// OnClosed ...
func (server *WebSocketServer) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Printf("client closed(%v)\n", err)

	if c.Context() != nil {
		conn := c.Context().(*GnetUpgraderConn)
		conn.Closed = true

		if server.ConnCloseHandler != nil {
			server.ConnCloseHandler(conn)
		}
	}

	return
}

// React ...
func (server *WebSocketServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {

	var ok bool
	var upgraderConn *GnetUpgraderConn

	// fmt.Printf("react, 当前全部数据, 1111, %v\n", c.Read())
	// fmt.Printf("react, 当前全部数据, 2222, %s\n", string(c.Read()))
	// fmt.Printf("react, frame: %s\n", string(frame))

	if upgraderConn, ok = c.Context().(*GnetUpgraderConn); !ok {
		// upgraderConn = NewDefaultUpgrader(c)
		// upgraderConn = NewEmptyUpgrader(c)
		upgraderConn = NewExtUpgrader(c)
		c.SetContext(upgraderConn)
	}

	if !upgraderConn.IsSuccessUpgraded {

		// Upgrade protocol
		_, err := upgraderConn.Upgrader.Upgrade(upgraderConn)
		if err != nil {
			log.Printf("react ws protocol upgrade exception， %v\n", err)
			return nil, gnet.Close
		} else {
			// log.Printf("react ws protocol upgrade success, %s\n", upgraderConn.GnetConn.RemoteAddr().String())

			upgraderConn.IsSuccessUpgraded = true
			server.updateConnActiveTs(upgraderConn)
		}

		if server.ConnEstablishedHandler != nil {
			if err := server.ConnEstablishedHandler(upgraderConn); err != nil {
				log.Printf("react ws ConnEstablished error， %v\n", err)
				return nil, gnet.Close
			}
		}

		return
	}

	// Data processing
	// 在 reactor 协程中做解码操作
	// msg, op, err := wsutil.ReadClientData(upgraderConn)
	// frame, err := ws.ReadFrame(upgraderConn)
	messages, err := wsutil.ReadClientMessage(upgraderConn, nil)
	if err != nil {
		log.Printf("message incomplete read next, message:%v,  error:%v\n", messages, err)
		return
	}

	// log.Printf("received message, op:%v,  msg:%v\n", op, msg)

	for _, message := range messages {

		switch message.OpCode {
		case ws.OpPing:
			log.Printf("ping, message:%v\n", message)
			wsutil.WriteServerMessage(upgraderConn, ws.OpPong, nil)
			server.updateConnActiveTs(upgraderConn)

		case ws.OpText:
			fallthrough
		case ws.OpBinary:
			server.WorkerPool.Submit(func() {
				// 具体业务在 worker pool中处理
				handlerParam := &DataHandlerParam{}
				handlerParam.OpCode = message.OpCode
				handlerParam.Request = message.Payload
				handlerParam.Writer = upgraderConn
				handlerParam.WSConn = upgraderConn
				handlerParam.Server = server

				server.Handler(handlerParam)
			})
			server.updateConnActiveTs(upgraderConn)

		case ws.OpClose:
			log.Printf("client close connection, Payload:%s,  error:%v\n", string(message.Payload), nil)
			server.closeConn(upgraderConn)
			upgraderConn.Closed = true
			return nil, gnet.Close

		default:
			log.Printf("not support oprate: %d, message:%v\n", message.OpCode, message)
		}
	}

	return
}

// send message passive
func (server *WebSocketServer) SendDownStreamMsg(wsConn *GnetUpgraderConn, opcode ws.OpCode, msg []byte) error {
	if wsConn == nil {
		return errors.New("SendDownStreamMsg wsConn is nil")
	}

	// server.updateConnActiveTs(wsConn)

	return wsutil.WriteServerMessage(*wsConn, opcode, msg)
}

// update connection active timestap
func (server *WebSocketServer) updateConnActiveTs(wsConn *GnetUpgraderConn) {

	now := time.Now().Unix()

	if now-wsConn.LastActiveTs > ConnMaxIdleSeconds {
		jobParam := &jobParam{server, wsConn}
		server.connTimeWheel.AddTimer(time.Second*time.Duration(ConnMaxIdleSeconds), nil, jobParam)
	}

	wsConn.LastActiveTs = now
}

// close connection
func (server *WebSocketServer) closeConn(wsConn *GnetUpgraderConn) {
	if wsConn == nil {
		return
	}

	if server.ConnCloseHandler != nil {
		server.ConnCloseHandler(wsConn)
		wsConn.Context = nil
		wsConn.Closed = true
	}

	ws.WriteFrame(wsConn, ws.NewCloseFrame(nil))
	wsConn.GnetConn.Close()
}

type jobParam struct {
	server *WebSocketServer
	wsConn *GnetUpgraderConn
}

// connection timeout job
func timeWheelJob(jParam interface{}) {
	param := jParam.(*jobParam)

	if param == nil || param.wsConn == nil || param.server == nil {
		return
	}

	diffNow := time.Now().Unix() - param.wsConn.LastActiveTs

	if diffNow > ConnMaxIdleSeconds {
		log.Printf("server关闭连接, 连接空闲%d秒, %v\n", ConnMaxIdleSeconds, param.wsConn)
		param.server.closeConn(param.wsConn)
	} else {
		param.server.connTimeWheel.AddTimer(time.Second*time.Duration((ConnMaxIdleSeconds-diffNow)), nil, param)
	}
}
