package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool { return true },
}



type Hub struct {
    sessions	map[string] *Session
    clients	map[string] *Client
    receivers	map[string] *Receiver
    config	*Config
}

type Session struct {
    //deviceName	string
    expiry	time.Time
}
func(s *Session) isExpired() bool {
    return s.expiry.Before(time.Now())
}

func (hub *Hub) GetReceivers() []string {
    r := make([]string, 0)
    for _, value := range hub.receivers {
	r = append(r, value.name)
    }

    log.Printf("Number of receivers: %d", len(r))
    return r
}

func (hub *Hub) GetFunctions(name string, clientId string) error {
    log.Printf("Receiver name requested: %s", name)
    if name == "" {
	return errors.New("Receiver Name Empty.")
    }
    if hub.receivers[name] == nil {
	return fmt.Errorf("Receiver not found with name: %s", name)
    }
    hub.receivers[name].getFunctions(clientId)
    return nil
}

//func (hub *Hub) SendFunctions(functions *[]MacronFunction) {
//    log.Printf("Functions: %v", functions)
//    response := ClientResponse {
//	Type: "functions",
//	Functions: functions,
//    }
//    bytes, _ := json.Marshal(&response)
//    
//    hub.client.egress <- bytes
//}
func (hub *Hub) SendFunctions(id string, functions *[]MacronFunction) {
    log.Printf("Functions: %v", functions)
    response := ClientResponse {
	Type: "functions",
	Functions: functions,
    }
    bytes, _ := json.Marshal(&response)
    
    if client := hub.clients[id]; client != nil { 
	hub.clients[id].egress <- bytes
    } else {
	log.Println("Receiver Response: ClientID provided does not exist...")
    }
}

func (hub *Hub) RemoveReceiver(name string) {
    delete(hub.receivers, name)
}

func (hub *Hub) ExecFunction(name string, id int) error {
    if name == "" {
	return errors.New("Receiver Name Empty.")
    }
    receiver := hub.receivers[name]
    if receiver == nil {
	return fmt.Errorf("Receiver not found with name: %s", name)
    }
    
    receiver.execFunction(id)
    return nil
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
	name: msg.ReceiverName,
	conn: ws,
	hub: hub,
	egress: make(chan []byte),
    }

    hub.receivers[receiver.name] = receiver
}


