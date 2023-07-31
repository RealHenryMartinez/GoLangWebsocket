// Run the server
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Create a root ctx and a CancelFunc which can be used to cancel retentionMap goroutine
	rootCtx := context.Background()
	ctx, cancel := context.WithCancel(rootCtx)

	// Run at the end of the retentionMap goroutine
	defer cancel()

	setUpAPI(ctx)

	// Serve on port :8080, fudge yeah hardcoded port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// HTTP route
func setUpAPI(ctx context.Context) {
	// Handle Web socket connections with the manager
	manager := NewManager(ctx)

	// Give the frontend a route to the server
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	// At the route ws, give the manager a route to the server for the websocket
	http.HandleFunc("/ws", manager.serveWS)
	http.HandleFunc("/login", manager.loginHandler) // Handle login logic

	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, len(manager.clients))
	})
}
