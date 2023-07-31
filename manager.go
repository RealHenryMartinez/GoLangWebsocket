package main

// Purpose of file: Keep track of clients using the websocket

// Be able to upgrade the http to web socket 101
import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Turn the HTTP requests into a websocket request
var (
	websocketUpgrader = websocket.Upgrader{
		// Apply the Origin Checker
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

var (
	// Send an error message to the client
	ErrEventNotSupported = errors.New("this event type is not supported")
)

// Check for the origin of the request
func checkOrigin(r *http.Request) bool {
	// Grab the request origin by the Header struct and then the Get method
	origin := r.Header.Get("Origin")

	// Switch statement to check for the origin if it is correct
	switch origin {
	case "http://localhost:8080":
		return true
	default:
		return false
	}
}

// Put all Clients in here
type Manager struct {

	// All of the clients in this websocket
	clients ClientList

	// The purpose of using a mutex is to provide safe concurrent access to shared resources, such as the clients list, in a multi-threaded environment.
	// The RWMutex allows multiple goroutines to read from the clients list concurrently, but only one goroutine can write to it at a time.
	sync.RWMutex
	// store handlers that are used to handle Events
	handlers map[string]EventHandler

	// A map of all the One time passwords allowed to access the web socket
	otps RetentionMap
}

// Constructor to initialize the values into the manager
func NewManager(ctx context.Context) *Manager {
	// Create a new Manager and initialize the values inside of the structure
	m := &Manager{
		clients:  make(ClientList),              // Make -> Initialize the clients into the client map
		handlers: make(map[string]EventHandler), // Handler are functions that handle events
		// Remove One time passwords older than 5 seconds
		otps: NewRetentionMap(ctx, 5*time.Second),
	}
	m.setupEventHandlers() // Default events and types and pointing to the manager via dot notation
	return m               // Allow the caller to access the new manager instance via the address
}

// Method receiver that points to the original manager instance it was called upon
// Sets the manager's event handlers
func (m *Manager) setupEventHandlers() {
	// sent_messages event

	// Make a handler by the event type as a key
	// e - sets a copy of the event
	// c - Reference to the client connection that uses this handler
	m.handlers[EventSentMessage] = SendMessageHandler
	// func(e Event, c *Client) error {
	// 	fmt.Println(e)
	// 	return nil
	// }
}

// Make sures the event is the correct event to the handler
// Make sures the event is the correct event handled by the websocket
func (m *Manager) routeEvent(event Event, c *Client) error {
	// Check if the event type is inside the handler list of events
	if handler, ok := m.handlers[event.Type]; ok {
		// Execute the Event Handler to find any errors returned while processing the event data inside of the handler list
		if err := handler(event, c); err != nil {
			return err
		}
		return nil

	} else {
		return ErrEventNotSupported
	}
}

// Allowing the HTTP client to connect to the websocket
// This is a receiver function meaning that it could be called from the manager struct
func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {

	// First we need to get the authentication credentials
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		// user not authorized
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Verify the OTP exists
	if !m.otps.VerifyOTP(otp) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Println("New websocket connection")

	// Upgrade the websocket by giving the web socket the HTTP request connect to the socket
	connection, err := websocketUpgrader.Upgrade(w, r, nil) // Reference the Upgrade function from the web socket struct and change the HTTP connection to websocket

	// Error occured while connecting to the websocket
	if err != nil {
		log.Println(err)
		return
	}

	// Create a new client by the returned instance
	client := NewClient(connection, m)

	// Add the client to the manager's list of clients
	m.addClient(client)
	// Close the connection
	//connection.Close()

	// Be able to allow the client to read and write simultaneously
	// Go Routines to allow the clients to read and write at the same time
	go client.readMessages()
	go client.writeMessages()

}

func (m *Manager) addClient(client *Client) {
	// sync.Mutex allows us to lock the manager by accessing the manager's state one at a time without other goroutine interfering potentially causing multiple clients
	m.Lock()

	// Calling until the end, allowing the sync.Mutex to unlock the state and
	defer m.Unlock()

	// Add the client and set it to true on its status
	m.clients[client] = true
}

// Remove the client to clean up the connection
func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	// Check if the status is true of false is in the list, then delete it
	if _, ok := m.clients[client]; ok {
		// Close the client's connection
		client.connection.Close()

		// Remove the client from the list using go's built in delete method
		delete(m.clients, client)
	}
}

func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req userLoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authenticate user if credentials are matched
	if req.Username == "HenryBenry" && req.Password == "henry" {
		// Formatting the response we want to send back to frontend
		type response struct {
			OTP string `json:"otp"`
		}

		// adding the new OTP to the retention map and fetching it
		otp := m.otps.NewOTP()

		// Create out response object containing the OTP data
		resp := response{
			OTP: otp.Key,
		}

		// Parse the response
		data, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			return // Stop the function
		}

		// Return the response to the caller
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// User failed to authenticate
	w.WriteHeader(http.StatusUnauthorized)
}
