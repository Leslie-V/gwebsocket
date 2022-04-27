package server

import (
	"fmt"
	"log"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

const (
	MessageTypeSwitch = iota
	MessageTypeFree
	MessageTypeAdd
	MessageTypeUpgraded
)

// SwitchMessage ...
type Message struct {
	Type    int
	WsType  ws.OpCode
	ID      string
	Payload interface{}
}

// ConnManager ...
type ConnManager struct {
	EstablishedNum int64
	Connections    map[string]*GnetUpgraderConn
	MessageQueue   chan (*Message)
}

// NewConnManager ...
func NewConnManager() *ConnManager {
	return &ConnManager{
		EstablishedNum: 0,
		Connections:    make(map[string]*GnetUpgraderConn),
		MessageQueue:   make(chan *Message, 10000),
	}
}

func (cm *ConnManager) connManagerHandler() {
	for true {

		message := <-cm.MessageQueue
		switch message.Type {
		case MessageTypeSwitch:
			fmt.Printf("switch message to %s: %v\n", message.ID, message)
			conn, ok := cm.Connections[message.ID]
			if ok {
				wsutil.WriteServerMessage(conn, message.WsType, []byte(message.Payload.(string)))
			} else {
				log.Printf("switch connection(%v) error: not found\n", conn)
			}

		case MessageTypeFree:
			conn, ok := cm.Connections[message.ID]
			if ok {
				delete(cm.Connections, message.ID)
				cm.EstablishedNum--

				fmt.Printf("freeed connection(%v)\n", conn)
			} else {
				log.Printf("free connection(%v) error: not found\n", conn)
			}

		case MessageTypeAdd:
			_, ok := cm.Connections[message.ID]
			if ok {
				log.Printf("connection(%v) had be used\n", message)
			} else {
				cm.Connections[message.ID] = message.Payload.(*GnetUpgraderConn)
				cm.EstablishedNum++
				fmt.Printf("add connection: %s\n", message.Payload.(*GnetUpgraderConn).UniqId)
			}

		default:
			fmt.Printf("message not support(%v)\n", message)
		}
	}
}

// Start ...
func (cm *ConnManager) Start() {
	log.Printf("ConnManager Start\n")
	go cm.connManagerHandler()
}

// Add ...
func (cm *ConnManager) Add(id string, conn *GnetUpgraderConn) {
	cm.SendMessageTo(&Message{Type: MessageTypeAdd, ID: conn.UniqId, Payload: conn})
}

// Free ...
func (cm *ConnManager) Free(id string) {
	cm.SendMessageTo(&Message{Type: MessageTypeFree, ID: id})
}

// SendMessageTo ...
func (cm *ConnManager) SendMessageTo(message *Message) {
	cm.MessageQueue <- message
}
