package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type sendData struct {
	Name  string
	Price float32
}

func main() {

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Disable CORS for testing
		},
	}

	// Start the server and keep track of the channel that it receives
	// new clients on:
	serverChan := make(chan chan string, 4)
	go stockServer(serverChan)

	// Defination of a HTTP handler function for the /status endpoint, that can receive
	// WebSocket-connections only... so note that browsing it with your browser will fail.
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		// Upgrade this HTTP connection to a WS connection:
		ws, _ := upgrader.Upgrade(w, r, nil)
		// And register a client for this connection with the uptimeServer:
		client := make(chan string, 1)
		serverChan <- client
		// And now check for stocks written to the client indefinitely.

		for {
			select {
			case text, _ := <-client:
				writer, _ := ws.NextWriter(websocket.TextMessage)
				writer.Write([]byte(text))
				writer.Close()
			}
		}
	})
	http.HandleFunc("/stock", homeHandler)

	log.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("view/index.html"))
}

func server(serverChan chan chan string) {
	var clients []chan string
	for {
		select {
		case client, _ := <-serverChan:
			clients = append(clients, client)
			// Broadcast the number of clients to all clients:
			for _, c := range clients {
				c <- fmt.Sprintf("%d client(s) connected.", len(clients))
			}
		}
	}
}

func client(clientName string, clientChan chan string) {
	for {
		text, _ := <-clientChan
		fmt.Printf("%s: %s\n", clientName, text)
	}
}

func stockServer(serverChan chan chan string) {
	var clients []chan string
	stockChan := make(chan []sendData, 1)
	// This goroutine will send data in background
	// updates to stockChan:
	go func(target chan []sendData) {
		i := 0
		for {
			time.Sleep(time.Second * 5)
			i++
			target <- []sendData{
				{Name: "Apple", Price: (rand.Float32() * 5) + 5},
				{Name: "Microsoft", Price: 300},
				{Name: "Google", Price: (rand.Float32() * 300) + 5},
				{Name: "Salesforce", Price: 500},
				{Name: "LinkedIn", Price: (rand.Float32() * 10) + 5},
				{Name: "Yahoo", Price: 700},
				{Name: "HP", Price: (rand.Float32() * 50) + 5},
				{Name: "Dell", Price: 900},
				{Name: "Levis", Price: (rand.Float32() * 13) + 5},
				{Name: "Bata", Price: 330},
			}
		}
	}(stockChan)
	// And now we listen to new clients and new uptime messages:
	for {
		select {
		case client, _ := <-serverChan:
			clients = append(clients, client)
		case uptime, _ := <-stockChan:
			// Send the uptime to all connected clients:
			for _, c := range clients {
				bytes, err := json.Marshal(uptime)
				if err != nil {
					fmt.Println("Can't serislize", bytes)
				}
				c <- fmt.Sprintf(string(bytes))
			}
		}
	}
}
