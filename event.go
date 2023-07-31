// Purpose of file: Contain all the logic to handle events from the client

package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event if the messages that are sent to the websocket
// Helps differentiate between different actions besides messages, anything that is sent, new_messages, etc
type Event struct {
	// Type is the type of event sent (message)
	Type string `json:"type"`

	// The data sent based on the type of event
	Payload json.RawMessage `json:"payload"`
}

// Event handler is a function signature to affect messages on the websocket from their type
// Checks the type of event received and returns an error if any was found
type EventHandler func(event Event, c *Client) error

// The event name for new chat messages sent
const EventSentMessage = "send_message"
const EventNewMessage = "new_message"

// Payload sent to the event sentMessage
type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
}

// NewMessageEvent is returned when responding to send_message
type NewMessageEvent struct {
	SendMessageEvent
	Sent time.Time `json:"sent"`
}

func SendMessageHandler(event Event, c *Client) error {
	// Get the message
	var chatEvent SendMessageEvent

	fmt.Println("chat event: ", chatEvent)
	fmt.Println("chat event: ", event)
	// Store the event payload into the event structure
	// check if the message was parsed correctly
	if err := json.Unmarshal(event.Payload, &chatEvent); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	// Preparing the message to send out
	var broadMessage NewMessageEvent

	broadMessage.Sent = time.Now()
	broadMessage.Message = chatEvent.Message
	broadMessage.From = chatEvent.From

	// Parse the message
	data, err := json.Marshal(broadMessage)
	// Check for any errors with parsing the message
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %v", err)
	}

	// Place the payload to the event that will be used to send the message back to the clients
	var outputMessage Event
	outputMessage.Payload = data
	outputMessage.Type = EventNewMessage

	// Let all the clients see the message
	for client := range c.manager.clients {

		// Send the message to the channel which the client would receive from the goroutine
		client.egress <- outputMessage
	}

	return nil
}
