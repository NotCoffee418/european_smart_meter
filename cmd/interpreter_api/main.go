package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/NotCoffee418/european_smart_meter/pkg/config"
	"github.com/NotCoffee418/european_smart_meter/pkg/interpreter"
	"github.com/NotCoffee418/european_smart_meter/pkg/port_reader"
	"github.com/gorilla/websocket"
)

var p1Reader *port_reader.P1Reader

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// ws clients for broadcasting live readings
var (
	wsClients                   = make(map[*websocket.Conn]bool)
	wsClientsMutex sync.RWMutex = sync.RWMutex{}
)

func main() {
	// Load config
	if err := config.LoadInterpreterAPIConfig(); err != nil {
		log.Fatalf("Failed to load interpreter API config: %v", err)
	}

	p1Reader = port_reader.NewP1Reader(
		config.ActiveInterpreterAPIConfig.SerialDevice,
		config.ActiveInterpreterAPIConfig.Baudrate,
	)

	// Start reading P1 port and handle signals/errors
	go p1Reader.StartReading(
		func(reading *interpreter.RawMeterReading) {
			BroadcastToWebSockets(reading)
		},
		func(err error) {
			if err != nil {
				log.Fatalf("Error reading P1 port: %v", err)
			}
		},
	)

	// Setup HTTP handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{
			"message": "Belgian Smart Meter API",
			"status":  "running",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		reading := p1Reader.GetLatestReading()
		if reading == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "No readings available yet",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reading)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		AddWebSocketClient(conn)

		// Send current reading immediately if available
		if reading := p1Reader.GetLatestReading(); reading != nil {
			conn.WriteMessage(websocket.TextMessage, reading.ToJsonBytes())
		}

		// Keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				RemoveWebSocketClient(conn)
				break
			}
		}
	})

	listener := fmt.Sprintf("%s:%d", config.ActiveInterpreterAPIConfig.ListenAddress, config.ActiveInterpreterAPIConfig.ListenPort)

	log.Printf("Starting European Smart Meter Interpreter API on %s", listener)
	log.Fatal(http.ListenAndServe(listener, nil))
}

func BroadcastToWebSockets(reading *interpreter.RawMeterReading) {
	wsClientsMutex.RLock()
	clients := make([]*websocket.Conn, 0, len(wsClients))
	for client := range wsClients {
		clients = append(clients, client)
	}
	wsClientsMutex.RUnlock()

	for _, client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, reading.ToJsonBytes()); err != nil {
			RemoveWebSocketClient(client)
		}
	}
}

func AddWebSocketClient(conn *websocket.Conn) {
	wsClientsMutex.Lock()
	wsClients[conn] = true
	wsClientsMutex.Unlock()
}

func RemoveWebSocketClient(conn *websocket.Conn) {
	wsClientsMutex.Lock()
	delete(wsClients, conn)
	wsClientsMutex.Unlock()
	conn.Close()
}
