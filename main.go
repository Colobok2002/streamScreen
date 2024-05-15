package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	frameDelay = 500 * time.Millisecond
)

func capturePrimaryDisplay() image.Image {
	numDisplays := screenshot.NumActiveDisplays()
	if numDisplays == 0 {
		return nil
	}
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil
	}
	return img
}

func sendVideoStream(conn *websocket.Conn) {
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

		if err := conn.WriteMessage(websocket.TextMessage, []byte(imgBase64)); err != nil {
			log.Println("Error sending image over WebSocket:", err)
			time.Sleep(frameDelay)
			continue
		}

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

	sendVideoStream(conn)
}

func main() {
	http.HandleFunc("/ws", wsHandler)

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "stream.html")
	})

	log.Fatal(http.ListenAndServe(":8081", nil))
}
