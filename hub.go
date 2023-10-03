package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool { return true },
}


type Receiver struct {
    deviceName	string
    conn	*websocket.Conn
    hub		*Hub
    egress	chan[]byte
}

type Hub struct {
    client	*Client
    receivers	map[string] *Receiver
    config	*Config
}

func (hub *Hub) GetReceivers() []string {
    r := make([]string, 0)
    for _, value := range hub.receivers {
	r = append(r, value.deviceName)
    }

    log.Printf("Number of receivers: %d", len(r))
    return r
}

func (hub *Hub) GetFunctions(name string) ([]string, error) {
    if name == "" {
	return nil, errors.New("Receiver Name Empty.")
    }
    if hub.receivers[name] == nil {
	return nil, fmt.Errorf("Receiver not found with name: %s", name)
    }
    functionList := []string{"Foobar", "Firefox", "Alacritty"}
    return functionList, nil
}

func (hub *Hub) RemoveReceiver(name string) {
    delete(hub.receivers, name)
}

func (hub *Hub) HandlerClientPassword(w http.ResponseWriter, r *http.Request) {
    if hub.client != nil {
	log.Println("Client already exists")
	log.Printf("Current Client: %v", hub.client)
	w.WriteHeader(400)
	return
    }
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
	log.Println(err)
	ws.Close()
	return
    }

    client := &Client{
	hub: hub,
	conn: ws,
	egress: make(chan []byte),
    }
    var authMsg ClientInbound
    err = ws.ReadJSON(&authMsg)
    if err != nil {
	log.Printf("Error unmarshalling request: %v", err)
	//hub.wsWriteClientResponse(ws, "error", nil, "Invalid JSON format") 
	client.sendErrorResponse("Invalid JSON format")
    }
    if authMsg.Password != hub.config.Server.Password {
	log.Printf("Client failed password authentication.")
	//hub.wsWriteClientResponse(ws, "error", nil, "Incorrect password.")
	client.sendErrorResponse("Incorrect password.")
	return
    }

    hub.client = client
    //hub.wsWriteClientResponse(ws, "auth_success", nil, "")
    client.sendMessage("auth_success")
    
    go client.readPump()
    go client.writePump()
}

func (hub *Hub) HandlerClient(w http.ResponseWriter, r *http.Request) {

    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
	log.Println(err)
	ws.Close()
	return
    }

    if hub.client != nil {
	client := &Client {
	    hub: hub,
	    conn: ws,
	    egress: make(chan []byte),
	}
	hub.client = client
    } else {
	log.Println("Client already exists")
	log.Printf("Current Client: %v", hub.client)
	ws.Close()
	return
    }

    confirmation := ClientResponse{
	Type: "auth_success",
	Receivers: nil,
    }
    sendJsonWs(ws, confirmation)

    go hub.client.readPump()
    go hub.client.writePump()
}

func (hub *Hub) HandlerReceiverPassword(w http.ResponseWriter, r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
	log.Println(err)
	ws.Close()
	return
    }
    log.Println("Receiver authenticating...")
    
    var authMsg ReceiverInbound
    err = ws.ReadJSON(&authMsg)
    if err != nil {
	log.Printf("Error parsing receiver auth message: %v", err)
	hub.wsWriteReceiverResponse(ws, "error", "Invalid JSON format.")
	return
    }
    if authMsg.Password != hub.config.Server.Password {
	log.Println("Receiver failed password authentication.")
	hub.wsWriteReceiverResponse(ws, "auth_failure", "Incorrect password.")
	return
    }
    receiver := &Receiver {
	deviceName: authMsg.ReceiverName,
	conn: ws,
	hub: hub,
	egress: make(chan []byte),
    }
    hub.receivers[authMsg.ReceiverName] = receiver
    hub.wsWriteReceiverResponse(ws, "auth_success", "")

    go receiver.readPump()
    go receiver.writePump()

}

func (hub *Hub) HandlerReceiver(w http.ResponseWriter, r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
	log.Println(err)
	ws.Close()
	return
    }
    log.Println("Receiver connecting...")
    _, p, err := ws.ReadMessage()
    if err != nil {
	log.Println("Error: ", err)
	ws.Close()
	return
    }

    log.Println("Receiver auth: ", string(p))

    var msg ReceiverInbound
    err = json.Unmarshal(p, &msg)
    if err != nil {
	sendErrorReceiverWs(ws, "Incorrect JSON format")
	ws.Close()
    }

    if msg.ReceiverName != "" {
	if hub.receivers[msg.ReceiverName] != nil {
	    sendErrorReceiverWs(ws, "Receiver with name already exists: " + msg.ReceiverName)
	    ws.Close()
	    return
	}
    }else {
	log.Println("Reciever name is nil")
	return
    }

    receiver := &Receiver{
	deviceName: msg.ReceiverName,
	conn: ws,
	hub: hub,
	egress: make(chan []byte),
    }

    hub.receivers[receiver.deviceName] = receiver
}


func (r *Receiver) readPump() {
    defer func() {
	r.hub.RemoveReceiver(r.deviceName)
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
    }
}

func (r *Receiver) writePump() {
    defer func() {
	r.hub.RemoveReceiver(r.deviceName)
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
	    sendJsonWs(r.conn, message)
	}
    }
}
