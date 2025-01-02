package utils

import (
	"encoding/base64"
	"fmt"
	"log"
	"messaging-service/models"
	"messaging-service/repository"
	"net/http"
	"time"

	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
)

type WebSocketHandler struct {
	Connections map[string]*websocket.Conn
	Mutex       sync.Mutex
	AMQPChannel *amqp.Channel
	Repo        repository.MessageRepository
	Upgrader    websocket.Upgrader
}

func NewWebSocketHandler(repo repository.MessageRepository, amqpChannel *amqp.Channel) *WebSocketHandler {
	return &WebSocketHandler{
		Connections: make(map[string]*websocket.Conn),
		AMQPChannel: amqpChannel,
		Repo:        repo,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id in WebSocket request", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP request to WebSocket
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Register the WebSocket connection
	h.Mutex.Lock()
	h.Connections[userID] = conn
	h.Mutex.Unlock()
	log.Printf("WebSocket connection established for user: %s", userID)

	// Listen for incoming messages
	for {
		var message struct {
			Type        string `json:"type"`
			SenderID    string `json:"sender_id"`
			ReceiverID  string `json:"receiver_id"`
			Content     string `json:"content"`
			FileID      string `json:"file_id,omitempty"`
			FileName    string `json:"file_name,omitempty"`
			ChunkIndex  int    `json:"chunk_index,omitempty"`
			TotalChunks int    `json:"total_chunks,omitempty"`
			ChunkData   string `json:"chunk_data,omitempty"`
		}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			h.Mutex.Lock()
			delete(h.Connections, userID)
			h.Mutex.Unlock()
			return
		}

		log.Print("Message received ", message.Content, message.ReceiverID, message.SenderID, message.Type)
		// Handle new message
		switch message.Type {
		case "send_message":
			h.HandleNewMessage(message.SenderID, message.ReceiverID, message.Content)
		case "send_file_chunk":
			log.Print("CHUNKKKKK", message.Type)
			h.HandleChunkedFileMessage(message.SenderID, message.ReceiverID, message.FileID, message.FileName, message.ChunkIndex, message.TotalChunks, message.ChunkData)
		default:
			log.Printf("Unknown message type: %s", message.Type)
		}
	}
}

func (h *WebSocketHandler) HandleNewMessage(senderID, receiverID, content string) {
	// Create and save the message in the database
	message := models.Message{
		ID:         uuid.New(),
		SenderID:   uuid.MustParse(senderID),
		ReceiverID: uuid.MustParse(receiverID),
		Content:    content,
		CreatedAt:  time.Now(),
	}
	err := h.Repo.CreateNewMessage(&message)
	if err != nil {
		log.Printf("Failed to save message: %v", err)
		return
	}

	// Notify sender of successful message sent
	h.Mutex.Lock()
	senderConn, senderOnline := h.Connections[senderID]
	h.Mutex.Unlock()

	if senderOnline {
		if err := senderConn.WriteJSON(map[string]interface{}{
			"type": MESSAGE_SENT_ACK,
		}); err != nil {
			log.Printf("Failed to send acknowledgment to sender %s: %v", senderID, err)
		}
	}

	// Notify the receiver
	h.Mutex.Lock()
	receiverConn, receiverOnline := h.Connections[receiverID]
	h.Mutex.Unlock()

	if receiverOnline {
		if err := receiverConn.WriteJSON(map[string]interface{}{
			"type":    NEW_MESSAGE_RECEIVED,
			"message": message,
		}); err != nil {
			log.Printf("Failed to notify receiver %s: %v", receiverID, err)
		}
	} else {
		// Notify offline user via RabbitMQ
		err := h.AMQPChannel.Publish(
			"",
			NOTIFICATION_SERVICE,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(fmt.Sprintf(`{"type":"new_message", "user_id":"%s", "content":"%s"}`, receiverID, content)),
			},
		)
		if err != nil {
			log.Printf("Failed to send message notification for offline user %s: %v", receiverID, err)
		}
	}
}

type ChunkedFile struct {
	FileID      string
	SenderID    string
	ReceiverID  string
	FileName    string
	TotalChunks int
	Chunks      map[int][]byte
}

var chunkedFiles = make(map[string]*ChunkedFile)

func (h *WebSocketHandler) HandleChunkedFileMessage(senderID, receiverID, fileID, fileName string, chunkIndex, totalChunks int, chunkData string) {
	// Decode the chunk from Base64
	data, err := base64.StdEncoding.DecodeString(chunkData)
	if err != nil {
		log.Printf("Failed to decode file chunk: %v", err)
		return
	}

	h.Mutex.Lock()
	defer h.Mutex.Unlock() // Ensure mutex is unlocked exactly once

	// Check if this file ID is already being tracked
	chunkedFile, exists := chunkedFiles[fileID]
	if !exists {
		chunkedFile = &ChunkedFile{
			FileID:      fileID,
			SenderID:    senderID,
			ReceiverID:  receiverID,
			FileName:    fileName,
			TotalChunks: totalChunks,
			Chunks:      make(map[int][]byte),
		}
		chunkedFiles[fileID] = chunkedFile
	}

	// Store the chunk
	chunkedFile.Chunks[chunkIndex] = data

	// Calculate progress
	progress := float64(len(chunkedFile.Chunks)) / float64(totalChunks) * 100
	log.Printf("Progress: %.2f%% for fileID %s", progress, fileID)

	// Notify sender about upload progress
	go h.notifyFileProgress(senderID, fileID, progress)

	// Check if all chunks are received
	if len(chunkedFile.Chunks) == totalChunks {
		log.Println("All chunks received, processing file...")
		go h.handleCompleteFile(chunkedFile)
		delete(chunkedFiles, fileID)
	}
}

func (h *WebSocketHandler) notifyFileProgress(senderID, fileID string, progress float64) {
	log.Printf("Notifying progress %.2f%% for fileID %s to sender %s", progress, fileID, senderID)

	h.Mutex.Lock()
	defer h.Mutex.Unlock() // Mutex used for thread safety

	senderConn, senderOnline := h.Connections[senderID]
	if senderOnline {
		err := senderConn.WriteJSON(map[string]interface{}{
			"type":     FILE_UPLOAD_PROGRESS,
			"file_id":  fileID,
			"progress": progress,
		})
		if err != nil {
			log.Printf("Failed to notify sender %s about file upload progress: %v", senderID, err)
		} else {
			log.Printf("Progress notification sent successfully for fileID %s", fileID)
		}
	} else {
		log.Printf("Sender %s is not online, skipping progress notification", senderID)
	}
}

func (h *WebSocketHandler) handleCompleteFile(chunkedFile *ChunkedFile) {
	// Combine chunks into a single file
	completeFile := []byte{}
	for i := 0; i < chunkedFile.TotalChunks; i++ {
		if chunkedFile.Chunks[i] == nil {
			log.Printf("Missing chunk %d for fileID %s", i, chunkedFile.FileID)
			return
		}
		completeFile = append(completeFile, chunkedFile.Chunks[i]...)
	}

	// Save the file on the server
	uniqueFileName, originalFileName, err := SaveFile(chunkedFile.FileName, completeFile)
	if err != nil {
		log.Printf("Failed to save file: %v", err)
		return
	}

	// Save metadata to the database
	message := models.Message{
		ID:           uuid.New(),
		SenderID:     uuid.MustParse(chunkedFile.SenderID),
		ReceiverID:   uuid.MustParse(chunkedFile.ReceiverID),
		MessageType:  "file",
		FilePath:     uniqueFileName,
		FileName:     originalFileName,
		FileMimeType: http.DetectContentType(completeFile),
		CreatedAt:    time.Now(),
	}

	err = h.Repo.CreateNewMessage(&message)
	if err != nil {
		log.Printf("Failed to save file message metadata: %v", err)
		return
	}

	h.notifyFileReceived(chunkedFile.SenderID, chunkedFile.ReceiverID, message)
}

func (h *WebSocketHandler) notifyFileReceived(senderID, receiverID string, message models.Message) {
	// Notify the sender
	h.Mutex.Lock()
	senderConn, senderOnline := h.Connections[senderID]
	h.Mutex.Unlock()

	if senderOnline {
		log.Print("Sender is online", senderOnline)
		if err := senderConn.WriteJSON(map[string]interface{}{
			"type":    FILE_SENT_ACK,
			"message": message,
		}); err != nil {
			log.Printf("Failed to notify sender %s: %v", senderID, err)
		}
	}

	// Notify the receiver
	h.Mutex.Lock()
	receiverConn, receiverOnline := h.Connections[receiverID]
	h.Mutex.Unlock()

	if receiverOnline {
		if err := receiverConn.WriteJSON(map[string]interface{}{
			"type":    NEW_FILE_RECEIVED,
			"message": message,
		}); err != nil {
			log.Printf("Failed to notify receiver %s: %v", receiverID, err)
		}
	}
}
