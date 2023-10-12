package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Credentials struct {
    DeviceName	string `json:"device_name"`
    Password	string `json:"password"`
}

type Session struct {
    deviceName	string
    expiry	time.Time
}

type HttpHub struct {
    sessions	map[string]Session
    config	Config
}

func(hub *HttpHub) setupRoutes() chi.Router {
    router := chi.NewRouter()

    return router
}

func(hub *HttpHub) clientLoginHandler(w http.ResponseWriter, r *http.Request) {
    var creds Credentials
    err := json.NewDecoder(r.Body).Decode(&creds)
    if err != nil {
	w.WriteHeader(http.StatusBadRequest)
	return
    }

    if creds.DeviceName == "" {
	w.WriteHeader(http.StatusBadRequest)
	return
    }

    if hub.config.Server.Password != creds.Password {
	w.WriteHeader(http.StatusUnauthorized)
	return
    }

    sessionToken := uuid.NewString()
    expiresAt := time.Now().Add(120 * time.Second)

    hub.sessions[sessionToken] = Session{
	deviceName: creds.DeviceName,
	expiry: expiresAt,
    }

    http.SetCookie(w, &http.Cookie{
	Name: "session_token",
	Value: sessionToken,
	Expires: expiresAt,
    })

}
