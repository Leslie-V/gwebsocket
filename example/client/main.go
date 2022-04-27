package main

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/websocket"
)

var origin = "http://127.0.0.1:8081/"
var url = "ws://127.0.0.1:8081/echo"

func main() {
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ok\n")
	for true {
		message := []byte("hello, world!你好")
		fmt.Printf("write\n")
		_, err = ws.Write(message)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Send: %s\n", message)

		var msg = make([]byte, 512)
		fmt.Printf("read\n")
		m, err := ws.Read(msg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Receive: %s\n", msg[:m])
		time.Sleep(time.Second * 10)
	}

	ws.Close() //关闭连接
}
