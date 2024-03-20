package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Maximum message size allowed
	maxMessageSize = 512
	// time allowed to write a message
	waitWriteTimeout = 10 * time.Second
	// send to peer with this period
	pingPeriod = time.Minute
	pongWait   = pingPeriod + (10 * time.Second)
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Message struct {
	Nickname string `json:"nickname,omitempty"`
	Content  string `json:"content,omitempty"`
}

type Client struct {
	nickname     string
	hub          *Hub
	conn         *websocket.Conn
	queueMessage chan *Message
}

func (c *Client) readWS() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("error from SetReadDeadline %v", err)

	}

	c.conn.SetPongHandler(func(ping string) error {
		fmt.Println("Pong:", c.nickname, ping)

		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("error from SetReadDeadline %v", err)
		}
		return nil
	})

	for {
		msg := Message{}
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Println("cannot read the message ", err)
			}
			return
		}
		c.hub.broadcast <- msg
	}
}

func (c *Client) writeWS() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, isOpen := <-c.queueMessage:
			if !isOpen {
				return
			}

			err := c.conn.SetWriteDeadline(time.Now().Add(waitWriteTimeout))
			if err != nil {
				log.Printf("error from SetWriteDeadline %v", err)
			}
			if err := c.conn.WriteJSON(&msg); err != nil {
				log.Println("can't write message into ws", err)
				return
			}
		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(waitWriteTimeout))
			if err != nil {
				log.Printf("error from SetWriteDeadline %v", err)
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("Ping")); err != nil {
				log.Printf("it can't write the Ping message: %v", err)
				return
			}
		}
	}
}

func HandleWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	nickname := r.URL.Query()["nickname"]

	if len(nickname) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("cannot get the websocket connection ", err)
		return
	}

	client := &Client{
		nickname:     nickname[0],
		hub:          hub,
		conn:         conn,
		queueMessage: make(chan *Message, 2),
	}

	client.hub.register <- client
	go client.writeWS()
	go client.readWS()
}
