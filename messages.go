package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type ClientInbound struct {
    Type	    string  `json:"type"`
    Password	    string  `json:"password"`
    ReceiverName    string  `json:"receiver_name,omitempty"`
    FunctionId	    *int    `json:"function_id,omitempty"`
}

type ClientResponse struct {
    Type	    string		`json:"type"`
    Error	    string		`json:"error,omitempty"`
    ReceiverName    string		`json:"receiver_name,omitempty"`
    Receivers	    *[]string		`json:"receivers,omitempty"`
    Functions	    *[]MacronFunction   `json:"functions,omitempty"`
}

type ReceiverInbound struct {
    Type	    string		`json:"type"`
    ClientId	    string		`json:"client_id,omitempty"`
    Password	    string  		`json:"password,omitempty"`
    ReceiverName    string  		`json:"receiver_name"`
    Functions	    *[]MacronFunction	`json:"functions,omitempty"`
}

type ReceiverResponse struct {
    Type	string	`json:"type"`
    ClientId	string	`json:"client_id,omitempty"`
    Id		*int	`json:"id,omitempty"`
    Error	string	`json:"error,omitempty"`
}


func sendJsonWs(ws *websocket.Conn, payload interface{}) {
    err := ws.WriteJSON(payload)
    if err != nil {
	log.Printf("Failed to marshal JSON response: %v", payload)
	ws.Close()
	return
    }
}

func sendErrorReceiverWs(ws *websocket.Conn, error string) {
    msg := ReceiverResponse {
	Type: "error",
	Error: error,
    }
    err := ws.WriteJSON(msg)
    if err != nil {
	log.Printf("Failed to marshal Error response: %v", msg)
    }
}

func writeClientResponse(w http.ResponseWriter, code int, msgType string, msg string) {
    w.WriteHeader(code)
    response := ClientResponse {
	Type: msgType,
	Error: msg,
    }
    bytes, _ := json.Marshal(response)
    w.WriteHeader(code)
    w.Write(bytes)
}

func (hub *Hub) wsWriteClientResponse(ws *websocket.Conn, msgType string, receivers *[]string, error string) {
    if error != "" {
	response := ClientResponse {
	    Type: msgType,
	    Error: error,
	}
	ws.WriteJSON(response)
	ws.WriteMessage(websocket.CloseAbnormalClosure, []byte(""))
	ws.Close()
    } else {
	response := ClientResponse {
	    Type: msgType,
	    Receivers: receivers,
	}
	ws.WriteJSON(response)
    }
}

func (hub *Hub) wsWriteReceiverResponse(ws *websocket.Conn, msgType string, error string) {
    if error != "" {
	response := ReceiverResponse {
	    Type: msgType,
	    Error: error,
	}
	ws.WriteJSON(response)
	ws.WriteMessage(websocket.CloseAbnormalClosure, []byte(""))
	ws.Close()
    } else {
	response := ReceiverResponse {
	    Type: msgType,
	}
	ws.WriteJSON(response)
    }
}
