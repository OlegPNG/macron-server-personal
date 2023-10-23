package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Receiver struct {
    name	string
    conn	*websocket.Conn
    hub		*Hub
    egress	chan[]byte
}

type MacronFunction struct {
    Id		*int	`json:"id"`
    Name	string  `json:"name"`
    Description	string	`json:"description"`
}
func (r *Receiver) close() {
    r.conn.WriteMessage(websocket.CloseAbnormalClosure, []byte(""))
    r.conn.Close()

    r.hub.RemoveReceiver(r.name)
}

func (r *Receiver) sendErrorResponse(error string) {
    response := ReceiverResponse {
	Type: "error",
	Error: error,
    }

    bytes, _ := json.Marshal(&response)

    r.egress <- bytes
}

func (r *Receiver) sendMessage(msgType string) {
    response := ReceiverResponse {
	Type: msgType,
    }

    bytes, _ := json.Marshal(&response)

    r.egress <- bytes
    //r.conn.WriteJSON(response)
}

func (r *Receiver) execFunction(id int) {
    response := ReceiverResponse {
	Type: "exec",
	Id: &id,
    }

    bytes, _ := json.Marshal(&response)

    r.egress <- bytes
}

func (r *Receiver) getFunctions() {
    r.sendMessage("functions")
}

func (r *Receiver) readPump() {
    defer func() {
	r.hub.RemoveReceiver(r.name)
	r.conn.Close()
    }()

    for {
	var message ReceiverInbound
	err := r.conn.ReadJSON(&message)
	if err != nil {
	    log.Printf("error: %v", err)
	    break
	}
	log.Printf("Receiver message: %v", message)
	switch message.Type {
	case "functions":
	    log.Println("Receiver sending functions...")
	    clientId := message.ClientId
	    if err != nil {
		log.Printf("error: %v", err)
	    } else {
		r.hub.SendFunctions(clientId, message.Functions)
	    }
	}
    }
}

func (r *Receiver) writePump() {
    defer func() {
	r.hub.RemoveReceiver(r.name)
	r.conn.Close()
    }()

    for {
	select {
	case message, ok := <- r.egress:
	    if !ok {
		r.conn.WriteMessage(websocket.CloseMessage, []byte{})
		log.Println("Receiver egress error")
	    }
	    log.Println("Sending receiver message: ", string(message))
	    //sendJsonWs(r.conn, message)
	    err := r.conn.WriteMessage(1, message)
	    if err != nil {
		return
	    }
	}
    }
}
