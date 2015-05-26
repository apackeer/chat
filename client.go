package main

import (
	"github.com/gorilla/websocket"
)

// client represents a single chatting user.
type client struct {
	// socket is the websocket for this client.
	socket *websocket.Conn

	// send is the channel on which messages are sent to the client
	send chan []byte

	// room is the room this client is chatting in.
	room *room
}

// The read method allows our client to read from the socket via the
// ReadMessage method, continually sending any received messages to the forward
// channel on the room type.
func (c *client) read() {
	for {
		// Read a message from the websocket and put it in the room this client
		// is chatting in's forwarding channel.
		if _, msg, err := c.socket.ReadMessage(); err == nil {
			c.room.forward <- msg
		} else {
			break
		}
	}
	c.socket.Close()
}

// The write method continually accepts messages from the send channel writing
// everything out of the socket via the WriteMessage method. If writing to the
// socket fails, the for loop is broken and the socket is closed.
func (c *client) write() {
	// Get all the messages out of the send channel and send them back through
	// the websocket
	for msg := range c.send {
		if err := c.socket.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
	c.socket.Close()
}
