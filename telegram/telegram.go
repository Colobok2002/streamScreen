package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type InlineKeyboardButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

func SendTelegramMessageWithButton(buttonURL string) error {
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
