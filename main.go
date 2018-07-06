package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

// Message represents an user message
type Message struct {
	Address string
	Text    string
	Type    int
}

// ServerConfig represents the server configuration such as Host and Port
type ServerConfig struct {
	Host string
	Port string
}

var (
	// List of users
	users = make(map[*websocket.Conn]bool)

	// sync channel
	channel = make(chan Message)

	// upgrader is an object which takes a normal HTTP connection and upgrading it to a WebSocket
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// instance of server configuration
	configuration ServerConfig
)

func handleWebsocketConnections(w http.ResponseWriter, r *http.Request) {

	// try to "upgrade" the connection to a persistent two-way connection between the client and server.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Unable to promote connection to WebSocket protocol: \n%v\n", err)
		return
	}
	defer conn.Close()

	// register connected user here
	users[conn] = true

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Unable to read message from connection: \n%v\n", err)
			delete(users, conn)
			break
		}
		text := string(msg)
		message := Message{
			Address: conn.RemoteAddr().String(),
			Text:    text,
			Type:    msgType,
		}

		// Print the message to the console
		fmt.Printf("%s received message: %s\n", message.Address, message.Text)

		// Broadcast received message ....
		channel <- message
	}
}

func handleMessages() {

	for {
		// Get message from the channel and distribut it along connected users
		message := <-channel
		for user := range users {
			// connection has one concurent reader and one concurent writer so I don't see necessity to write in goroutine?
			err := user.WriteMessage(message.Type, []byte(message.Text))
			if err != nil {
				log.Printf("Unable to send a message: %v", err)
				user.Close()
				delete(users, user)
			}
		}
	}
}

func handleClient(w http.ResponseWriter, r *http.Request) {

	tmpl := template.Must(template.ParseFiles("templates/client.html"))
	tmpl.Execute(w, configuration)
}

func main() {

	// Collect required port
	port := os.Getenv("PORT") // $ export PORT=8080
	if len(port) == 0 {
		port = "8080" // default, if user didn't specify something
	}

	configuration = ServerConfig{
		Host: "127.0.0.1",
		Port: port,
	}

	http.HandleFunc("/", handleClient)
	http.HandleFunc("/ws", handleWebsocketConnections)

	// listen for incoming messages
	go handleMessages()

	log.Printf("Running HTTP server on http://%s:%s", configuration.Host, configuration.Port)
	addr := fmt.Sprintf(":%s", configuration.Port)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("Unable to run a server: ", err)
	}

}
