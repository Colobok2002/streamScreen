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
var workTunnel bool

func runTunnel() error {
	tunnel, err := localtunnel.New(8081, "localhost", localtunnel.Options{})
	if err != nil {
		log.Println("Error starting localtunnel:", err)
		return err
	}
	defer tunnel.Close()

	appURL := tunnel.URL() 
	log.Println("App URL:", appURL)

	if err := telegram.SendTelegramMessageWithButton(appURL+ "/stream",appURL+ "/restart",); err != nil {
		log.Println("Error sending message to Telegram:", err)
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for workTunnel {
		time.Sleep(1 * time.Microsecond)
	}
	workTunnel = true
	runTunnel()
	return nil
}

func RestartTunnel(w http.ResponseWriter, r *http.Request) {
	workTunnel = false
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Tunnel restarted successfully"))
}

func RestartTunnelWrapper() {
	workTunnel = false
}

func main() {

	go showConsol.CheckHotkey(RestartTunnelWrapper)

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

	http.HandleFunc("/restart", RestartTunnel)

	go func() {
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	go stream.SendVideoStream()
	workTunnel = true
	runTunnel()
}
