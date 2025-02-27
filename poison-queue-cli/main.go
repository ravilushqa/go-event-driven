package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/urfave/cli/v2"
)

const PoisonQueueTopic = "PoisonQueue"

// Message represents a message in the poison queue
type Message struct {
	ID            string
	Reason        string
	OriginalTopic string
	CreatedAt     time.Time
	Content       string
}

// Mock data store
var (
	mockMessages      = make(map[string]Message)
	mockMessagesMutex sync.RWMutex
	mockDataInterval  *time.Ticker
)

// ErrorReasons is a list of possible error reasons for poisoned messages
var ErrorReasons = []string{
	"Failed to process JSON payload",
	"Network connection timeout",
	"Invalid message format",
	"Authorization failed",
	"Database query error",
	"Rate limit exceeded",
	"Dependency service unavailable",
	"Message too large",
	"Invalid schema version",
	"Required field missing",
}

// OriginalTopics is a list of possible original topics
var OriginalTopics = []string{
	"user.events",
	"order.created",
	"payment.processed",
	"inventory.updated",
	"notification.email",
	"audit.log",
	"metrics.system",
}

// Handler uses GoChannel for pub/sub
type Handler struct {
	subscriber message.Subscriber
	publisher  message.Publisher
}

var (
	// Singleton pubsub for the entire application
	goPubSub *gochannel.GoChannel
)

// NewHandler creates a new handler with GoChannel pub/sub
func NewHandler() (*Handler, error) {
	// Create a GoChannel pubsub if it doesn't exist yet
	if goPubSub == nil {
		logger := watermill.NewStdLogger(false, false)

		// Create a gochannel pubsub with persistent flag for messages to be consumed multiple times
		goPubSub = gochannel.NewGoChannel(
			gochannel.Config{
				Persistent: true,
			},
			logger,
		)
	}

	return &Handler{
		subscriber: goPubSub,
		publisher:  goPubSub,
	}, nil
}

// StartMockDataGenerator begins sending mock poisoned messages
func StartMockDataGenerator(interval time.Duration) {
	if mockDataInterval != nil {
		mockDataInterval.Stop()
	}

	mockDataInterval = time.NewTicker(interval)

	go func() {
		for range mockDataInterval.C {
			// Add a new mock message with 50% probability
			if rand.Float32() < 0.5 {
				AddMockMessage()
			}
		}
	}()
}

// StopMockDataGenerator stops the mock data generator
func StopMockDataGenerator() {
	if mockDataInterval != nil {
		mockDataInterval.Stop()
		mockDataInterval = nil
	}
}

// AddMockMessage creates and adds a new mock message to the queue
func AddMockMessage() Message {
	// Generate random mock data
	id := uuid.New().String()
	reason := ErrorReasons[rand.Intn(len(ErrorReasons))]
	originalTopic := OriginalTopics[rand.Intn(len(OriginalTopics))]

	// Create a watermill message
	msg := message.NewMessage(id, []byte(fmt.Sprintf("Sample message content with error: %s", reason)))

	// Add metadata
	msg.Metadata.Set(middleware.ReasonForPoisonedKey, reason)
	msg.Metadata.Set(middleware.PoisonedTopicKey, originalTopic)

	// Initialize the handler if needed
	if goPubSub == nil {
		handler, _ := NewHandler()
		if handler == nil {
			log.Println("Failed to create handler for message generation")
			return Message{}
		}
	}

	// Publish to the poison queue topic
	err := goPubSub.Publish(PoisonQueueTopic, msg)
	if err != nil {
		log.Printf("Error publishing mock message: %v", err)
		return Message{}
	}

	// Create message object to return and add to cache
	newMsg := Message{
		ID:            id,
		Reason:        reason,
		OriginalTopic: originalTopic,
		CreatedAt:     time.Now(),
		Content:       fmt.Sprintf("Sample message content with error: %s", reason),
	}

	// Add to cache if it exists
	messageCache.Lock()
	if messageCache.loaded {
		messageCache.messages = append(messageCache.messages, newMsg)
	}
	messageCache.Unlock()

	log.Printf("Added mock message: %s (Reason: %s, Topic: %s)", id, reason, originalTopic)

	return newMsg
}

// messageCache stores a cached copy of all messages to ensure consistent results
var messageCache struct {
	sync.RWMutex
	messages []Message
	loaded   bool
}

// Preview returns all messages in the poison queue
func (h *Handler) Preview(ctx context.Context) ([]Message, error) {
	// If we already have a cached message list, return it immediately
	messageCache.RLock()
	if messageCache.loaded && len(messageCache.messages) > 0 {
		result := messageCache.messages
		messageCache.RUnlock()
		return result, nil
	}
	messageCache.RUnlock()

	// Otherwise, load messages from the queue
	messages, err := h.subscriber.Subscribe(ctx, PoisonQueueTopic)
	if err != nil {
		return nil, err
	}

	firstID := ""
	var result []Message

	// Create a timeout context with a longer timeout to ensure we get all messages
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	collectDone := make(chan struct{})

	// Start collecting messages in a goroutine
	go func() {
		for msg := range messages {
			if firstID == msg.UUID {
				// We've seen all messages once, break the loop
				close(collectDone)
				return
			}

			if firstID == "" {
				firstID = msg.UUID
			}

			// Create a message struct
			msgData := Message{
				ID:            msg.UUID,
				Reason:        msg.Metadata.Get(middleware.ReasonForPoisonedKey),
				OriginalTopic: msg.Metadata.Get(middleware.PoisonedTopicKey),
				CreatedAt:     time.Now(), // We don't have the actual creation time
				Content:       string(msg.Payload),
			}

			result = append(result, msgData)

			// Re-publish the message to keep it in the queue
			err := h.publisher.Publish(PoisonQueueTopic, msg)
			if err != nil {
				log.Printf("Error re-publishing message: %v", err)
			}

			msg.Ack()
		}
	}()

	// Wait for either all messages to be collected or a timeout
	select {
	case <-timeoutCtx.Done():
		// If we timeout, use what we have so far
	case <-collectDone:
		// We've seen all messages
	}

	// Update the cache with the collected messages
	if len(result) > 0 {
		messageCache.Lock()
		messageCache.messages = result
		messageCache.loaded = true
		messageCache.Unlock()
	}

	return result, nil
}

// Remove removes a message from the poison queue
func (h *Handler) Remove(ctx context.Context, messageID string) error {
	messages, err := h.subscriber.Subscribe(ctx, PoisonQueueTopic)
	if err != nil {
		return err
	}

	firstID := ""
	messageFound := false

	// Create a timeout context to avoid infinite waiting
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		return fmt.Errorf("timed out while searching for message")
	default:
		for msg := range messages {
			if firstID == msg.UUID {
				if !messageFound {
					return fmt.Errorf("message %s not found", messageID)
				}
				break // We've processed all messages once
			}

			if firstID == "" {
				firstID = msg.UUID
			}

			if msg.UUID == messageID {
				msg.Ack()
				messageFound = true
				log.Printf("Removed message: %s", messageID)

				// Update the cache to remove this message
				messageCache.Lock()
				if messageCache.loaded {
					// Create a new slice without the removed message
					newMessages := make([]Message, 0, len(messageCache.messages))
					for _, m := range messageCache.messages {
						if m.ID != messageID {
							newMessages = append(newMessages, m)
						}
					}
					messageCache.messages = newMessages
				}
				messageCache.Unlock()

				continue // Skip republishing this message
			}

			// Republish other messages
			err = h.publisher.Publish(PoisonQueueTopic, msg)
			if err != nil {
				return err
			}

			msg.Ack()
		}
	}

	if !messageFound {
		return fmt.Errorf("message %s not found", messageID)
	}

	return nil
}

// Requeue requeues a message to its original topic
func (h *Handler) Requeue(ctx context.Context, messageID string) error {
	messages, err := h.subscriber.Subscribe(ctx, PoisonQueueTopic)
	if err != nil {
		return err
	}

	firstID := ""
	messageFound := false

	// Create a timeout context to avoid infinite waiting
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		return fmt.Errorf("timed out while searching for message")
	default:
		for msg := range messages {
			if firstID == msg.UUID {
				if !messageFound {
					return fmt.Errorf("message %s not found", messageID)
				}
				break // We've processed all messages once
			}

			if firstID == "" {
				firstID = msg.UUID
			}

			if msg.UUID == messageID {
				topic := msg.Metadata.Get(middleware.PoisonedTopicKey)
				if topic == "" {
					// Republish the message to keep it in the queue
					err = h.publisher.Publish(PoisonQueueTopic, msg)
					if err != nil {
						return err
					}
					msg.Ack()
					return fmt.Errorf("failed to requeue message %s: no original topic", messageID)
				}

				// Publish to the original topic
				err = h.publisher.Publish(topic, msg)
				if err != nil {
					// If publishing to the original topic fails, put it back in the poison queue
					err2 := h.publisher.Publish(PoisonQueueTopic, msg)
					if err2 != nil {
						log.Printf("Failed to re-add message to poison queue: %v", err2)
					}
					msg.Ack()
					return fmt.Errorf("failed to publish to %s: %w", topic, err)
				}

				messageFound = true
				log.Printf("Requeued message %s to topic %s", messageID, topic)

				// Update the cache to remove this message
				messageCache.Lock()
				if messageCache.loaded {
					// Create a new slice without the requeued message
					newMessages := make([]Message, 0, len(messageCache.messages))
					for _, m := range messageCache.messages {
						if m.ID != messageID {
							newMessages = append(newMessages, m)
						}
					}
					messageCache.messages = newMessages
				}
				messageCache.Unlock()

				msg.Ack()
				continue // Skip republishing this message to the poison queue
			}

			// Republish other messages
			err = h.publisher.Publish(PoisonQueueTopic, msg)
			if err != nil {
				return err
			}

			msg.Ack()
		}
	}

	if !messageFound {
		return fmt.Errorf("message %s not found", messageID)
	}

	return nil
}

// WebServer is a struct that encapsulates all web handler functionality
type WebServer struct {
	templates *template.Template
	basePath  string
}

// NewWebServer creates a new web server instance
func NewWebServer(basePath string) (*WebServer, error) {
	templates, err := template.ParseGlob(filepath.Join(basePath, "templates/*.html"))
	if err != nil {
		return nil, err
	}

	return &WebServer{
		templates: templates,
		basePath:  basePath,
	}, nil
}

// renderTemplate renders a template with the given name and data
func (s *WebServer) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html")
	err := s.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// IndexHandler serves the main page
func (s *WebServer) IndexHandler(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate(w, "layout.html", nil)
}

// MessagesHandler lists all messages in the poison queue
func (s *WebServer) MessagesHandler(w http.ResponseWriter, r *http.Request) {
	h, err := NewHandler()
	if err != nil {
		http.Error(w, "Failed to create handler: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messages, err := h.Preview(ctx)
	if err != nil {
		http.Error(w, "Failed to preview messages: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Messages []Message
		Stats    struct {
			Total    int
			ByTopic  map[string]int
			ByReason map[string]int
		}
	}{
		Messages: messages,
	}

	// Calculate statistics
	data.Stats.Total = len(messages)
	data.Stats.ByTopic = make(map[string]int)
	data.Stats.ByReason = make(map[string]int)

	for _, msg := range messages {
		data.Stats.ByTopic[msg.OriginalTopic]++
		data.Stats.ByReason[msg.Reason]++
	}

	s.renderTemplate(w, "messages.html", data)
}

// RequeueHandler requeues a message to its original topic
func (s *WebServer) RequeueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	if messageID == "" {
		http.Error(w, "Message ID is required", http.StatusBadRequest)
		return
	}

	h, err := NewHandler()
	if err != nil {
		notif := struct {
			Type    string
			Message string
		}{
			Type:    "error",
			Message: "Failed to create handler: " + err.Error(),
		}
		s.renderTemplate(w, "notification.html", notif)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = h.Requeue(ctx, messageID)
	if err != nil {
		notif := struct {
			Type    string
			Message string
		}{
			Type:    "error",
			Message: "Failed to requeue message: " + err.Error(),
		}
		s.renderTemplate(w, "notification.html", notif)
		w.Header().Set("HX-Trigger", `{"refreshMessages": true}`)
		return
	}

	notif := struct {
		Type    string
		Message string
	}{
		Type:    "success",
		Message: "Message requeued successfully",
	}
	s.renderTemplate(w, "notification.html", notif)
	w.Header().Set("HX-Trigger", `{"refreshMessages": true}`)
}

// RemoveHandler removes a message from the poison queue
func (s *WebServer) RemoveHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	if messageID == "" {
		http.Error(w, "Message ID is required", http.StatusBadRequest)
		return
	}

	h, err := NewHandler()
	if err != nil {
		notif := struct {
			Type    string
			Message string
		}{
			Type:    "error",
			Message: "Failed to create handler: " + err.Error(),
		}
		s.renderTemplate(w, "notification.html", notif)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = h.Remove(ctx, messageID)
	if err != nil {
		notif := struct {
			Type    string
			Message string
		}{
			Type:    "error",
			Message: "Failed to remove message: " + err.Error(),
		}
		s.renderTemplate(w, "notification.html", notif)
		w.Header().Set("HX-Trigger", `{"refreshMessages": true}`)
		return
	}

	notif := struct {
		Type    string
		Message string
	}{
		Type:    "success",
		Message: "Message removed successfully",
	}
	s.renderTemplate(w, "notification.html", notif)
	w.Header().Set("HX-Trigger", `{"refreshMessages": true}`)
}

// GenerateHandler adds a mock message to the queue
func (s *WebServer) GenerateHandler(w http.ResponseWriter, r *http.Request) {
	msg := AddMockMessage()

	notif := struct {
		Type    string
		Message string
	}{
		Type:    "success",
		Message: fmt.Sprintf("Generated new message: %s", msg.ID),
	}
	s.renderTemplate(w, "notification.html", notif)
	w.Header().Set("HX-Trigger", `{"refreshMessages": true}`)
}

// GeneratorControlHandler controls the automatic mock message generator
func (s *WebServer) GeneratorControlHandler(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")

	if action == "start" {
		// Parse interval
		interval := 5 * time.Second
		if intervalStr := r.URL.Query().Get("interval"); intervalStr != "" {
			if parsedInterval, err := time.ParseDuration(intervalStr); err == nil {
				interval = parsedInterval
			}
		}

		StartMockDataGenerator(interval)

		notif := struct {
			Type    string
			Message string
		}{
			Type:    "success",
			Message: fmt.Sprintf("Started automatic message generator (interval: %s)", interval),
		}
		s.renderTemplate(w, "notification.html", notif)
	} else if action == "stop" {
		StopMockDataGenerator()

		notif := struct {
			Type    string
			Message string
		}{
			Type:    "success",
			Message: "Stopped automatic message generator",
		}
		s.renderTemplate(w, "notification.html", notif)
	} else if action == "reset-cache" {
		// Reset the message cache to force a fresh load
		messageCache.Lock()
		messageCache.messages = nil
		messageCache.loaded = false
		messageCache.Unlock()

		notif := struct {
			Type    string
			Message string
		}{
			Type:    "success",
			Message: "Message cache reset - data will be reloaded",
		}
		s.renderTemplate(w, "notification.html", notif)
	} else {
		notif := struct {
			Type    string
			Message string
		}{
			Type:    "error",
			Message: "Invalid action. Use 'start', 'stop', or 'reset-cache'",
		}
		s.renderTemplate(w, "notification.html", notif)
	}

	w.Header().Set("HX-Trigger", `{"refreshMessages": true}`)
}

// runWebServer starts the web server
func runWebServer(port string) error {
	basePath, err := os.Getwd()
	if err != nil {
		return err
	}

	// Initialize random number generator
	rand.Seed(time.Now().UnixNano())

	// Initialize GoChannel pubsub
	_, err = NewHandler()
	if err != nil {
		return fmt.Errorf("failed to initialize GoChannel pubsub: %w", err)
	}

	// Add some initial mock messages
	for i := 0; i < 5; i++ {
		AddMockMessage()
	}

	webServer, err := NewWebServer(basePath)
	if err != nil {
		return err
	}

	r := mux.NewRouter()

	// Routes
	r.HandleFunc("/", webServer.IndexHandler).Methods("GET")
	r.HandleFunc("/messages", webServer.MessagesHandler).Methods("GET")
	r.HandleFunc("/requeue/{id}", webServer.RequeueHandler).Methods("POST")
	r.HandleFunc("/remove/{id}", webServer.RemoveHandler).Methods("DELETE")
	r.HandleFunc("/generate", webServer.GenerateHandler).Methods("POST")
	r.HandleFunc("/generator", webServer.GeneratorControlHandler).Methods("POST")

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(basePath, "static")))))

	// Add HTMX HX-Trigger to refresh messages after actions
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trigger := r.Header.Get("HX-Trigger")
			if strings.EqualFold(trigger, "refreshMessages") {
				w.Header().Set("HX-Trigger", `{"refreshMessages": true}`)
			}
			next.ServeHTTP(w, r)
		})
	})

	fmt.Printf("Web server running at http://localhost:%s\n", port)
	fmt.Println("GoChannel mode: Using in-memory message bus instead of Kafka")
	return http.ListenAndServe(":"+port, r)
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	app := &cli.App{
		Name:  "poison-queue-cli",
		Usage: "Manage the Poison Queue",
		Commands: []*cli.Command{
			{
				Name:  "preview",
				Usage: "preview messages",
				Action: func(c *cli.Context) error {
					h, err := NewHandler()
					if err != nil {
						return err
					}

					messages, err := h.Preview(c.Context)
					if err != nil {
						return err
					}

					if len(messages) == 0 {
						fmt.Println("No messages in the poison queue")
						return nil
					}

					fmt.Println("Messages in poison queue:")
					fmt.Println("---------------------------")
					fmt.Printf("%-36s | %-25s | %s\n", "ID", "ORIGINAL TOPIC", "REASON")
					fmt.Println("-----------------------------------------------------------------------------------")
					for _, m := range messages {
						fmt.Printf("%-36s | %-25s | %s\n", m.ID, m.OriginalTopic, m.Reason)
					}

					return nil
				},
			},
			{
				Name:      "remove",
				ArgsUsage: "<message_id>",
				Usage:     "remove message",
				Action: func(c *cli.Context) error {
					h, err := NewHandler()
					if err != nil {
						return err
					}

					err = h.Remove(c.Context, c.Args().First())
					if err != nil {
						return err
					}

					fmt.Println("Message removed successfully")
					return nil
				},
			},
			{
				Name:      "requeue",
				ArgsUsage: "<message_id>",
				Usage:     "requeue message",
				Action: func(c *cli.Context) error {
					h, err := NewHandler()
					if err != nil {
						return err
					}

					err = h.Requeue(c.Context, c.Args().First())
					if err != nil {
						return err
					}

					fmt.Println("Message requeued successfully")
					return nil
				},
			},
			{
				Name:  "generate",
				Usage: "generate a mock message",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "count",
						Aliases: []string{"n"},
						Value:   1,
						Usage:   "number of mock messages to generate",
					},
				},
				Action: func(c *cli.Context) error {
					count := c.Int("count")
					if count < 1 {
						count = 1
					}

					for i := 0; i < count; i++ {
						msg := AddMockMessage()
						fmt.Printf("Generated message: %s (Reason: %s, Topic: %s)\n",
							msg.ID, msg.Reason, msg.OriginalTopic)
					}

					return nil
				},
			},
			{
				Name:  "web",
				Usage: "start web UI server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Value:   "8080",
						Usage:   "port to run the web server on",
					},
					&cli.IntFlag{
						Name:    "mock-data",
						Aliases: []string{"m"},
						Value:   5,
						Usage:   "number of initial mock messages to generate",
					},
					&cli.BoolFlag{
						Name:    "auto-generate",
						Aliases: []string{"a"},
						Value:   false,
						Usage:   "automatically generate new messages periodically",
					},
					&cli.StringFlag{
						Name:    "interval",
						Aliases: []string{"i"},
						Value:   "5s",
						Usage:   "interval for auto-generating messages (e.g. 5s, 1m)",
					},
				},
				Action: func(c *cli.Context) error {
					// Generate initial mock data
					mockCount := c.Int("mock-data")
					for i := 0; i < mockCount; i++ {
						AddMockMessage()
					}

					// Start auto-generator if requested
					if c.Bool("auto-generate") {
						interval, err := time.ParseDuration(c.String("interval"))
						if err != nil {
							interval = 5 * time.Second
						}
						StartMockDataGenerator(interval)
						fmt.Printf("Auto-generating mock messages every %s\n", interval)
					}

					return runWebServer(c.String("port"))
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
