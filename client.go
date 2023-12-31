package main

import (
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
    hub         *Hub
    id		string
    conn        *websocket.Conn
    egress      chan[]byte
}

func (c *Client) sendErrorResponse(error string) {
    response := ClientResponse {
	Type: "error",
	Error: error,
    }

    c.conn.WriteJSON(response)
}
func (c *Client) close() {
    c.conn.WriteMessage(websocket.CloseAbnormalClosure, []byte(""))
    c.conn.Close()
    
    //c.hub.clients[c.] = nil
}

func (c *Client) sendMessage(msgType string) {
    response := ClientResponse {
	Type: msgType,
    }

    c.conn.WriteJSON(response)
}

func (c *Client) sendReceiverResponse(receivers *[]string) {
    response := ClientResponse {
	Type: "receivers",
	Receivers: receivers,
    }

    // May want to switch back to sending serialized message to client egress
    c.conn.WriteJSON(response)
}

func (c *Client) sendFunctionResponse(name string, functions *[]MacronFunction) {
    response := ClientResponse {
	Type: "functions",
	ReceiverName: name,
	Functions: functions,
    }
    c.conn.WriteJSON(response)
}

func (c *Client) readPump() {
    defer func() {
	c.close()
    }()

    for {
	var message ClientInbound
	err := c.conn.ReadJSON(&message)
	if err != nil {
	    log.Printf("error: %v", err)
	    c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	    c.conn.Close()
	    break
	}
	log.Printf("Client message: %v", message)
	switch message.Type {
	case "receivers":
	    log.Printf("Client requesting receivers...")
	    receivers := c.hub.GetReceivers()
	    c.sendReceiverResponse(&receivers)
	    log.Println("Sending list of receivers")
	case "functions":
	    log.Printf("Client requesting functions from: %s", message.ReceiverName)
	    err := c.hub.GetFunctions(message.ReceiverName, c.id)
	    if err != nil {
		log.Printf("Error Getting Functions: %v", err.Error())
		c.sendErrorResponse(err.Error())
	    }
	case "exec":
	    c.hub.ExecFunction(message.ReceiverName, *message.FunctionId)

	}
    }
}

func (c *Client) writePump() {
    defer func() {
	c.close()
    }()
    for {
	select {
	case message, ok := <- c.egress:
	    log.Println("Sending client message")
	    if !ok {
		c.conn.WriteMessage(websocket.CloseMessage, []byte{})
		c.conn.Close()
		log.Println("error sending client message")
		return
	    }
	    err := c.conn.WriteMessage(1, message)
	    if err != nil {
		return
	    }
	}
    }
}
