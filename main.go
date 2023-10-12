package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	dotenv "github.com/joho/godotenv"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
    Server      ServerConfig
}
type ServerConfig struct {
    AuthType    string  `toml:"auth_type"`
    Password    string
}


func parseConfig(dir string) (*Config, error) {
    configBytes, err := os.ReadFile(dir)
    if err != nil {
        log.Printf("Error: %v", err)
        return nil, err
    }
    var cfg Config
    err = toml.Unmarshal(configBytes, &cfg)
    if err != nil {
        log.Printf("Error: %v", err)
        return nil, err
    }

    return &cfg, nil
}

func main() {
    args := os.Args[1:]
    

    err := dotenv.Load()
    if err != nil {
        log.Fatal("Could not load environment")
    }


    logDir := os.Getenv("MACRON_LOG")
    if logDir == "" {
        log.Fatal("Could not find log file directory in env: MACRON_LOG is empty")
    }

    logfile, err := os.Create(logDir)

    if err != nil {
        log.Fatal(err)
    }
    defer logfile.Close()

    log.SetOutput(logfile)


    switch len(args) {
    case 0:
        startServer()
    default:
        os.Exit(0)
    }
}

func (hub *Hub) HandlerInfo(w http.ResponseWriter, r *http.Request) {
    type InfoResponse struct {
        Type        string  `json:"type"`
        AuthType    string  `json:"auth_type"`
    }
    msg := InfoResponse {
        Type: "info",
        AuthType: hub.config.Server.AuthType,
    }
    bytes, err := json.Marshal(msg)
    if err != nil {
        w.WriteHeader(500)
        w.Write([]byte("Internal Error"))
        log.Printf("Error marshalling json: %v", err)
    }
    w.Header().Add("Content-Type", "application/json")
    w.WriteHeader(200)
    w.Write(bytes)
}

func setupRoutes(hub *Hub) chi.Router {
    router := chi.NewRouter()
    router.Use(middleware.RequestID)
    router.Use(middleware.RealIP)
    router.Use(middleware.Logger)
    router.Use(middleware.Recoverer)

    v1Router := chi.NewRouter()
    wsRouter := chi.NewRouter()
    
    if hub.config.Server.AuthType == "password" {
        wsRouter.Get("/client", hub.HandlerClientPassword)
        wsRouter.Get("/receiver", hub.HandlerReceiverPassword)
    } else {
        wsRouter.Get("/client", hub.HandlerClient)
        wsRouter.Get("/receiver", hub.HandlerReceiver)
    }

    v1Router.Mount("/ws", wsRouter)
    router.Mount("/v1", v1Router)
    router.Get("/info", hub.HandlerInfo)
    
    return router
}

func startServer() {
    println("Starting Macron Server...")
    log.Println("Starting Macron Server...")

    home := os.Getenv("HOME")
    cfgDir := filepath.Join(home, "/.config/macron-server/config.toml")

    config, err := parseConfig(cfgDir)

    switch config.Server.AuthType {
    case "password":
        if config.Server.Password == "" {
            println("AuthType is 'password' but password is not configured.")
            os.Exit(1)
        }
    }
    if err != nil {
        log.Printf("Error parsing config: %v", err)
    }
    portString := os.Getenv("PORT")
    if portString == "" {
        log.Fatal("PORT not found in the environment")
    }

    hub := Hub{
        nil,
        make(map[string]*Receiver),
        config,
    }

    router := setupRoutes(&hub)

    server := http.Server{
        Handler: router,
        Addr: ":" + portString,
    }

    err = server.ListenAndServe()
    if err != nil {
        log.Fatal(err)
    }
}
