package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ITelegramSender interface {
	SendTelegramMessage(message string) ([]byte, error)
}

type TelegramSender struct{}

func (t *TelegramSender) SendTelegramMessage(message string) ([]byte, error) {
	token := getEnv("BOT_TOKEN", "")
	chatId := getEnv("CHAT_ID", "")

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?parse_mode=html", token)
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(Message{chatId, message}); err != nil {
		return nil, err
	}
	resp, err := http.Post(url, "application/json; charset=utf-8", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return text, fmt.Errorf("telegram API %d: %s", resp.StatusCode, string(text))
	}
	return text, nil
}
