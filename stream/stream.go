package stream

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	frameDelay = 400 * time.Millisecond
)

var (
	mu      sync.Mutex
	clients = make(map[*websocket.Conn]bool)
)

func capturePrimaryDisplay() image.Image {
	numDisplays := screenshot.NumActiveDisplays()
	if numDisplays == 0 {
		return nil
	}
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		log.Println("Error capturing screen:", err)
		return nil
	}
	return img
}

func SendVideoStream() {
	for {
		img := capturePrimaryDisplay()
		if img == nil {
			time.Sleep(frameDelay)
			continue
		}

		var imgBuf bytes.Buffer
		err := png.Encode(&imgBuf, img)
		if err != nil {
			log.Println("Error encoding image:", err)
			time.Sleep(frameDelay)
			continue
		}
		imgBase64 := base64.StdEncoding.EncodeToString(imgBuf.Bytes())

		mu.Lock()
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, []byte(imgBase64)); err != nil {
				log.Printf("Error sending image to client: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
		mu.Unlock()

		time.Sleep(frameDelay)
	}
}

func WsStreamHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open WebSocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			break
		}
	}
}
