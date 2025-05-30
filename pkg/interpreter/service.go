package interpreter

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// Manage websocket connection and call handleMeterReading for each reading
func StartListener(host string, funcToCall func(reading *RawMeterReading)) {
	const (
		maxRetries     = 10
		baseRetryDelay = 2 * time.Second
		maxRetryDelay  = 60 * time.Second
	)

	// WebSocket server URL
	u := url.URL{Scheme: "ws", Host: host, Path: "/ws"}

	// Channel to handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	retryCount := 0

	for {
		select {
		case <-interrupt:
			log.Println("Interrupt received, shutting down...")
			return
		default:
			// Calculate retry delay with exponential backoff
			retryDelay := time.Duration(1<<retryCount) * baseRetryDelay
			if retryDelay > maxRetryDelay {
				retryDelay = maxRetryDelay
			}

			if retryCount > 0 {
				log.Printf("Retrying connection in %v... (attempt %d/%d)", retryDelay, retryCount+1, maxRetries)
				select {
				case <-time.After(retryDelay):
				case <-interrupt:
					log.Println("Interrupt received during retry wait, shutting down...")
					return
				}
			}

			log.Printf("Connecting to %s", u.String())

			// Create a simple dialer with timeout
			dialer := websocket.DefaultDialer
			dialer.HandshakeTimeout = 10 * time.Second
			c, _, err := dialer.Dial(u.String(), nil)
			if err != nil {
				log.Printf("Connection failed: %v", err)
				retryCount++
				if retryCount >= maxRetries {
					log.Printf("Max retries (%d) reached. Giving up.", maxRetries)
					return
				}
				continue
			}

			log.Println("Connected! Accepting meter readings.")

			// Reset retry count on successful connection
			retryCount = 0

			// Handle the connection until it breaks or we're interrupted
			connectionBroken := handleConnection(c, interrupt, funcToCall)

			c.Close()

			if !connectionBroken {
				// Clean shutdown requested
				return
			}

			log.Println("Connection lost, will retry...")
		}
	}
}

func handleConnection(
	c *websocket.Conn,
	interrupt chan os.Signal,
	funcToCall func(reading *RawMeterReading),
) bool {
	done := make(chan struct{})

	// Set read deadline to detect dead connections
	// Expect message every second
	c.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Goroutine to read messages
	go func() {
		defer close(done)
		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				} else {
					log.Printf("Connection closed: %v", err)
				}
				return
			}

			// Reset read deadline on successful message
			c.SetReadDeadline(time.Now().Add(10 * time.Second))

			// We only expect RawMeterReading messages
			if messageType == websocket.TextMessage {
				if meterReading := MeterReadingFromJsonBytes(message); meterReading != nil {
					funcToCall(meterReading)
				} else {
					log.Printf("Failed to parse meter reading: %s", string(message))
				}
			} else if messageType == websocket.PingMessage {
				// Handle ping messages (should be handled by SetPingHandler but just in case)
				log.Println("Received ping message")
			} else {
				log.Printf("Received unexpected message type: %d", messageType)
			}
		}
	}()

	// Goroutine to send periodic pings to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := c.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					log.Printf("Failed to send ping: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Wait for connection to break or interrupt signal
	select {
	case <-done:
		// Connection broke
		return true
	case <-interrupt:
		log.Println("Interrupt received, closing connection...")

		// Send close message
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("Error sending close message:", err)
		}

		// Wait for close confirmation or timeout
		select {
		case <-done:
		case <-time.After(time.Second):
		}

		// Clean shutdown
		return false
	}
}
