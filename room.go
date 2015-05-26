package main

import (
	"log"
	"net/http"

	"github.com/apackeer/trace"
	"github.com/gorilla/websocket"
)

type room struct {
	// forward is a channel that holds incoming messages
	// that should be forward to other clients.
	forward chan []byte

	// The join and leave channels exist simply to allow us to safely add and
	// remove clients from the clients map. If we were to access the map
	// directly, it is possible that two Go routines running concurrently might
	// try to modify the map at the same time resulting in corrupt memory or an
	// unpredictable state.

	// join is a channel for clients wishing to join the room.
	join chan *client

	// leave is a channel for clients wishing to leave the room.
	leave chan *client

	// clients holds all current clients in this room.
	clients map[*client]bool

	// tracer will recieve trace information of activity in the rrom.
	tracer trace.Tracer
}

// newRoom makes a new room that is ready to go.
func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
		tracer:	 trace.Off()
	}
}

// Keep watching the three channels inside our room: join, leave, and forward.
// If a message is received on any of those channels, the select statement
// will run the code for that particular case. It is important to remember
// that it will only run one block of case code at a time. This is how we are
// able to synchronize to ensure that our r.clients map is only ever modified
// by one thing at a time.
func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			// joining. If we receive a message on the join channel, we simply
			// update the r.clients map to keep a reference of the client that has
			// joined the room. Notice that we are setting the value to true. We are
			// using the map more like a slice, but do not have to worry about
			// shrinking the slice as clients come and go through time—setting the
			// value to true is just a handy, low-memory way of storing the
			// reference.
			r.clients[client] = true
			r.tracer.Trace("New client joined")
		case client := <-r.leave:
			// leaving. If we receive a message on the leave channel, we simply
			// delete the client type from the map, and close its send channel.
			// Closing a channel has special significance in Go, which becomes clear
			// when we look at our final select case.
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("Client left")
		case msg := <-r.forward:
			// forward message to all clients. If we receive a message on the forward
			// channel, we iterate over all the clients and send the message down
			// each client's send channel. Then, the write method of our client type
			// will pick it up and send it down the socket to the browser.
			for client := range r.clients {
				select {
				case client.send <- msg:
					// send the message by putting it in clients send queue
					r.tracer.Trace(" -- sent to client")
				default:
					// failed to send. ie there is no send channel on the client (closed)
					// If the send channel is closed, then we know the client is not
					// receiving any more messages, and this is where our second select
					// clause (specifically the default case) takes the action of
					// removing the client from the room and tidying things up.
					delete(r.clients, client)
					close(client.send)
					r.tracer.Trace(" -- failed to send, cleaned up client")
				}
			}
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize,
	WriteBufferSize: socketBufferSize}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}

	// All being well, we then create our client and pass it into the join
	// channel for the current room. We also defer the leaving operation for
	// when the client is finished, which will ensure everything is tidied up
	// after a user goes away.

	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}
	r.join <- client
	defer func() { r.leave <- client }()
	// The write method for the client is then called as a Go routine seperate
	// thread
	go client.write()

	// Finally, we call the read method in the main thread, which will block
	// operations (keeping the connection alive) until it's time to close it.
	client.read()
}
