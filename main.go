package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pubnub "github.com/pubnub/go/v7"
)

const (
	// Demo keys - replace with your own keys from PubNub Admin Portal
	PUBLISH_KEY   = "Change me: your-publish-key-here"
	SUBSCRIBE_KEY = "Change me: your-subscribe-key-here"
	CHAT_CHANNEL  = "chat-room"
)

type ChatMessage struct {
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	fmt.Println("🚀 PubNub Go Chat Application")
	fmt.Println("=============================")

	// Check if PubNub keys are properly configured
	if strings.Contains(PUBLISH_KEY, "your-publish-key-here") {
		fmt.Println("PLEASE DEFINE PUBNUB KEYS IN MAIN.GO")
		fmt.Println("See ReadMe for more information")
		os.Exit(1)
	}

	// Get username from user
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	if username == "" {
		username = "Anonymous"
	}

	fmt.Printf("Welcome %s! You're now entering the chat room.\n", username)
	fmt.Println("Type 'quit' to exit the chat.")
	fmt.Println("=============================")

	// Initialize PubNub
	config := pubnub.NewConfigWithUserId(pubnub.UserId(username))
	config.SubscribeKey = SUBSCRIBE_KEY
	config.PublishKey = PUBLISH_KEY

	pn := pubnub.NewPubNub(config)

	// Create listener for incoming messages
	listener := pubnub.NewListener()

	// Channel to signal when connected
	doneConnect := make(chan bool)

	// Start goroutine to handle incoming events
	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					fmt.Println("✅ Connected to PubNub! You can start chatting now.")
					doneConnect <- true
				case pubnub.PNDisconnectedCategory:
					fmt.Println("❌ Disconnected from PubNub")
				case pubnub.PNReconnectedCategory:
					fmt.Println("🔄 Reconnected to PubNub")
				}
			case message := <-listener.Message:
				// Handle incoming chat messages
				if msg, ok := message.Message.(map[string]interface{}); ok {
					msgUsername := ""
					msgText := ""
					msgTime := ""

					if u, exists := msg["username"]; exists {
						if uStr, ok := u.(string); ok {
							msgUsername = uStr
						}
					}

					if m, exists := msg["message"]; exists {
						if mStr, ok := m.(string); ok {
							msgText = mStr
						}
					}

					if t, exists := msg["timestamp"]; exists {
						if tStr, ok := t.(string); ok {
							msgTime = tStr
						}
					}

					// Don't display our own messages
					if msgUsername != username {
						fmt.Printf("\n💬 [%s] %s: %s\n", msgTime, msgUsername, msgText)
						fmt.Print("You: ")
					}
				}
			case presence := <-listener.Presence:
				// Handle presence events (users joining/leaving)
				if presence.Event == "join" && presence.UUID != username {
					fmt.Printf("\n🟢 %s joined the chat\n", presence.UUID)
					fmt.Print("You: ")
				} else if presence.Event == "leave" && presence.UUID != username {
					fmt.Printf("\n🔴 %s left the chat\n", presence.UUID)
					fmt.Print("You: ")
				}
			}
		}
	}()

	// Add listener to PubNub
	pn.AddListener(listener)

	// Subscribe to the chat channel with presence
	pn.Subscribe().
		Channels([]string{CHAT_CHANNEL}).
		WithPresence(true).
		Execute()

	// Wait for connection
	<-doneConnect

	// Main chat loop
	for {
		fmt.Print("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" || input == "exit" {
			fmt.Println("👋 Goodbye!")
			pn.UnsubscribeAll()
			time.Sleep(1 * time.Second) // Give time for unsubscribe
			break
		}

		if input != "" {
			// Create chat message
			chatMsg := ChatMessage{
				Username:  username,
				Message:   input,
				Timestamp: time.Now(),
			}

			// Publish message to channel
			_, status, err := pn.Publish().
				Channel(CHAT_CHANNEL).
				Message(map[string]interface{}{
					"username":  chatMsg.Username,
					"message":   chatMsg.Message,
					"timestamp": chatMsg.Timestamp.Format("15:04:05"),
				}).
				Execute()

			if err != nil {
				log.Printf("Failed to publish message: %v", err)
			} else if status.StatusCode != 200 {
				log.Printf("Publish failed with status: %d", status.StatusCode)
			}
		}
	}
}
