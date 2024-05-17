package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type InlineKeyboardButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

func getTunnelPassword() (string, error) {
	url := "https://loca.lt/mytunnelpassword"

	// Выполняем HTTP GET запрос к URL
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Считываем содержимое ответа
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Возвращаем пароль в виде строки
	return string(body), nil
}

func SendTelegramMessageWithButton(buttonURL string, restatrUrl string) error {
	botToken := "7139011613:AAFU7H6YKKPUZFBtvprwknH8VJ5LvDF4Ukw"
	chatID := "-1002111741114"
	password, _ := getTunnelPassword()
	message := "Началась новая трансялция\nPassword " + password
	buttonText := "Открыть в барузере"
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: buttonText, URL: buttonURL},
			},
			{
				{Text: "Перезапустить сервер", URL: restatrUrl},
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
