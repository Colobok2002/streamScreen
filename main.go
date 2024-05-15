package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"html/template"
	"image"
	"image/png"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
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

func sendVideoStream() {
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

func wsHandler(w http.ResponseWriter, r *http.Request) {
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

func run(ctx context.Context) error {
	listener, err := ngrok.Listen(ctx,
		config.HTTPEndpoint(config.WithForwardsTo("localhost:8081")),
		ngrok.WithAuthtoken("2a1spZ5Tu8L7gSQAJaafnsG4bJc_2h9nDWzFKoZ3Z1qu7BLLu"),
	)
	if err != nil {
		log.Println("Error starting ngrok:", err)
		return err
	}

	log.Println("App URL", listener.URL()+"/stream")
	return http.Serve(listener, nil)
}

func main() {
	tmpl, err := template.ParseFiles("stream_template.html")
	if err != nil {
		log.Fatal("Error parsing template:", err)
	}

	http.HandleFunc("/ws", wsHandler)

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		wsURL := ""
		if strings.Contains(r.Host, ":") {
			wsURL = "ws://" + r.Host + "/ws"
		} else {
			wsURL = "wss://" + r.Host + "/ws"
		}
		if err := tmpl.Execute(w, struct{ WebSocketURL string }{WebSocketURL: wsURL}); err != nil {
			http.Error(w, "Error executing template", http.StatusInternalServerError)
		}
	})

	go func() {
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	go sendVideoStream()

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
