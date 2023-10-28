package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Credential struct {
    Email	string `json:"email"`
    Password	string `json:"password"`
}

type AuthenticationMessage struct {
    Type		string	`json:"type"`
    SessionToken	string	`json:"session_token"`
}

func NewAuthMessage(token string) AuthenticationMessage {
    return AuthenticationMessage{
	Type: "auth",
	SessionToken: token,
    }
}

func (hub *Hub) LoginHandler(w http.ResponseWriter, r *http.Request) {
    var creds Credential
    err := json.NewDecoder(r.Body).Decode(&creds)
    if err != nil {
	w.WriteHeader(http.StatusBadRequest)
	log.Println("Auth Failed: Couldn't marshal credentials")
	return
    }
    if hub.config.Server.AuthType == "full" {
	if creds.Email != hub.config.Server.Email {
	    w.WriteHeader(http.StatusUnauthorized)
	    log.Println("Incorrect email")
	    return
	}
    } else {
	if creds.Email == "" {
	    w.WriteHeader(http.StatusUnauthorized)
	    log.Println("Auth Failed: No email found")
	    return
	}
    }
    if creds.Password != hub.config.Server.Password {
	w.WriteHeader(http.StatusUnauthorized)
	log.Println("Auth Failed: Invalid password")
	return
    }


    session := &Session{
	expiry: time.Now().Add(120 * time.Second),
    }
    token := uuid.New().String()

    hub.sessions[token] = session

    _, ok := hub.sessions[token]
    
    log.Printf("Session with token %v exists: %v", token, ok)

    msg := NewAuthMessage(token)
    bytes, err := json.Marshal(msg)
    if err != nil {
	log.Println("Error marshalling authentication response.")
	w.WriteHeader(http.StatusInternalServerError)
	return
    }
    w.Write(bytes)
}

func (hub *Hub) TokenAuth(token string) error {
    //split := strings.Split(header, "Bearer: ")
    //log.Printf("Split first index: %v", split[0])
    //token := split[1]

    log.Printf("Received Token: %v", token)
    session, ok := hub.sessions[token]
    if !ok {
	log.Printf("Existing Token: %v", hub.sessions)
	return errors.New("Session does not Exist")
    }
    if session.isExpired() {
	delete(hub.sessions, token)
	log.Println("Token is expired")
	return errors.New("Session has expired")
    }

    return nil
}

func (hub *Hub) ClientHandler(w http.ResponseWriter, r *http.Request) {
    //auth := r.Header.Get("Authorization")
    auth := r.URL.Query().Get("session_token")
    err := hub.TokenAuth(auth)
    if err != nil {
	w.WriteHeader(http.StatusUnauthorized)
	return
    }
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
	log.Println(err)
	ws.Close()
	return
    }
    clientId := uuid.New().String()
    client := &Client{
	hub: hub,
	id: clientId,
	conn: ws,
	egress: make(chan []byte),
    }
    hub.clients[clientId] = client
    client.sendMessage("auth_success")

    go client.readPump()
    go client.writePump()
}

func (hub *Hub) ReceiverHandler(w http.ResponseWriter, r *http.Request) {
    auth := r.URL.Query().Get("session_token")
    err := hub.TokenAuth(auth)
    if err != nil {
	w.WriteHeader(http.StatusUnauthorized)
	return
    }
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
	log.Println(err)
	ws.Close()
	return
    }

    log.Println("Receiver initiating...")

    var authMsg ReceiverInbound
    err = ws.ReadJSON(&authMsg)
    if err != nil {
	log.Printf("Error parsing receiver auth message: %v", err)
	hub.wsWriteReceiverResponse(ws, "error", "Invalid JSON format.")
	return
    }
    if hub.receivers[authMsg.ReceiverName] != nil {
	hub.wsWriteReceiverResponse(ws, "error", "Receiver name already exists.")
	return
    }

    receiver := &Receiver {
	name: authMsg.ReceiverName,
	conn: ws,
	hub: hub,
	egress: make(chan []byte),
    }
    hub.receivers[authMsg.ReceiverName] = receiver
    hub.wsWriteReceiverResponse(ws, "auth_success", "")

    go receiver.readPump()
    go receiver.writePump()
}

func (hub *Hub) HandlerClientPassword(w http.ResponseWriter, r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
	log.Println(err)
	ws.Close()
	return
    }

    id := uuid.New().String()
    client := &Client{
	hub: hub,
	id: id,
	conn: ws,
	egress: make(chan []byte),
    }
    var authMsg ClientInbound
    err = ws.ReadJSON(&authMsg)
    if err != nil {
	log.Printf("Error unmarshalling request: %v", err)
	//hub.wsWriteClientResponse(ws, "error", nil, "Invalid JSON format") 
	client.sendErrorResponse("Invalid JSON format")
	client.close()
	return
    }
    if authMsg.Password != hub.config.Server.Password {
	log.Printf("Client failed password authentication.")
	//hub.wsWriteClientResponse(ws, "error", nil, "Incorrect password.")
	client.sendErrorResponse("Incorrect password.")
	client.close()
	return
    }

    hub.clients[id] = client
    //hub.client = client
    //hub.wsWriteClientResponse(ws, "auth_success", nil, "")
    client.sendMessage("auth_success")
    
    go client.readPump()
    go client.writePump()
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
	name: authMsg.ReceiverName,
	conn: ws,
	hub: hub,
	egress: make(chan []byte),
    }
    hub.receivers[authMsg.ReceiverName] = receiver
    hub.wsWriteReceiverResponse(ws, "auth_success", "")

    go receiver.readPump()
    go receiver.writePump()

}

func (hub *Hub) HandlerAuth(w http.ResponseWriter, r *http.Request) {

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    //http.ServeFile(w, r, "static/index.html")
    tmpl := template.Must(template.ParseFiles("templates/index.tmpl.html"))
    tmpl.Execute(w, "")
}
