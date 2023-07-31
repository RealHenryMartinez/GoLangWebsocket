package main

import (
	"encoding/json" // Be able to send JSON and not raw bytes
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// pongWait is how long we will await a pong response from client
	pongWait = 10 * time.Second
	// pingInterval has to be less than pongWait, We cant multiply by 0.9 to get 90% of time
	// Because that can make decimals, so instead *9 / 10 to get 90%
	// The reason why it has to be less than PingRequency is becuase otherwise it will send a new Ping before getting response
	pingInterval = (pongWait * 9) / 10
)

// A map of clients and check if they are connected to the web socket
type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn // WebSocket connection for the client
	manager    *Manager        // Reference to the client manager

	// The Type of data we are sending to the channel is Payload and Type based on the Event for data transmission which client can access
	egress chan Event // A channel for outgoing messages
}

// Create a new client with the given client's connection and reference to the manager
/*
	conn - The connection to the web socket connection server
	manager - The manager to connect to
	@returns - The reference to the new client by pointing to the new client
*/
func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	// Return the reference to the new client instance
	return &Client{
		connection: conn,
		manager:    manager,
		// Create a map list of bytes
		egress: make(chan Event), // Making a map list of channels that accept messages in []bytes
	}
}

// Help the client to read messages
// Ran as a goroutine
func (c *Client) readMessages() {
	defer func() {
		// Close the connection when a message is read
		c.manager.removeClient(c)
	}()

	// Max size of messages to read in bytes
	c.connection.SetReadLimit(512)

	// Wait for Pong response until it reaches the 10 second mark
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}
	// Set the next pong timeout
	c.connection.SetPongHandler(c.pongHandler)

	// Loop this process forever to read messages forever
	for {
		// Read the message that is next for the client in the connection
		messageType, payload, err := c.connection.ReadMessage()

		if err != nil {
			// Close the connection and return an error
			// Only logging the error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v\n", err)
			}
			break // Break the loop resulting in closing the connection and cleaning up the client

		}
		var request Event
		// Print or log the received JSON payload
		log.Println("Received JSON Payload:", payload)

		// Put the incoming data into json from the request reference
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("Error marshalling message: %v", err)
			//break // Breaking the connection
		}

		// Process the incoming payload
		if err := c.manager.routeEvent(request, c); err != nil {
			log.Println("Eror handeling message: ", err)
		}

		log.Println("Message type: ", messageType)
		// Stringify the message
		log.Println("Payload: ", string(payload))

		// Test that the function works by testing the message through every client
		// for wsclient := range c.manager.clients {
		// 	// Testing the message through the payload through the client
		// 	wsclient.egress <- payload
		// }
	}
}

// Listens for new messages from the connection to output to the client
// messagetype and payload to send to the client
func (c *Client) writeMessages() {
	log.Println("Writing message")
	// Trigger the ping to check if the client is still there
	ticker := time.NewTicker(pingInterval) // Function to periodically check if the client is still there from the ping interval

	// Cleanup function that will be called when no returns are called before the end of the function
	defer func() {
		ticker.Stop()             // stop the ticker timer when the client is not there anymore
		c.manager.removeClient(c) // If the function causes a closing to the connection
	}()

	for {
		// Waits for the goroutine to finish processing the message
		// <- Pause the queue and assign the current message connection _, ok := <-c.egress:
		message, ok := <-c.egress // Only receives the message bytes
		if !ok {
			// The websocket connection has been closed and seen from the error message
			if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
				// Log that the connection is closed and the reason
				log.Println("connection closed: ", err)
			}
			// Close the write message goroutine thread as the connection closed
			return
		}
		// Convert to JSON
		data, err := json.Marshal(message)
		log.Println(data)

		if err != nil {
			log.Println("connection closed: ", err)
			return // close the connection
		}
		// Send the message to the client
		/*
			websocket.TextMessage ->
		*/

		// Write a message to the clients by encoding the data into the web socket connection
		if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Println("ERROR IN WRITE MESSAGE: ", err)
		}
		log.Println("Sent Message")
		// Timer
		<-ticker.C // Wait for the next tick interval to be called
		log.Println("Ping")
		// Send the Ping
		if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			log.Println("writemsg: ", err)
			return // return to break this goroutine triggeing cleanup
		}
	}
}

// pongHandler is used to handle PongMessages for the Client
func (c *Client) pongHandler(appData string) error {
	// Current time + Pong Wait time
	log.Println("pong")

	// Set the new read deadline for the next Pong message
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
