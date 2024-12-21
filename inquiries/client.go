package main

import (
	"bytes"
	"fmt"
	"net/http"
)

func main() {

	// JSON данные для запроса
	jsonData := []byte(`{"action":"stop"}`)

	// Отправляем POST запрос на контроллер (порт 8080)
	resp, err := http.Post("http://localhost:8080/stop-agent", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	// Выводим статус ответа
	fmt.Println("Response status:", resp.Status)
}
