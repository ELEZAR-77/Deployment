package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Command struct {
	Action string `json:"action"`
}

type RequestLog struct {
	Endpoint string `json:"endpoint"`
	Action   string `json:"action"`
	Time     string `json:"time"`
}

var server *http.Server

func logRequest(endpoint, action string) {
	// Открываем файл для чтения и записи (если файла нет, он будет создан)
	file, err := os.OpenFile("agent_requests_log.json", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Чтение текущих данных из файла
	var logs []RequestLog
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&logs); err != nil && err.Error() != "EOF" {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// Добавляем новый лог
	log := RequestLog{
		Endpoint: endpoint,
		Action:   action,
		Time:     time.Now().Format(time.RFC3339),
	}
	logs = append(logs, log)

	// Позиция указателя на конец файла перед записью новых данных
	// Этот шаг можно опустить, если нужно просто добавлять новые записи в конец файла
	file.Seek(0, 0) // Сбрасываем указатель на начало файла

	// Перезаписываем файл с новыми логами
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Форматируем вывод в JSON
	if err := encoder.Encode(logs); err != nil {
		fmt.Println("Error encoding JSON:", err)
	}
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	var cmd Command
	if err := json.Unmarshal(body, &cmd); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	logRequest("/command", cmd.Action) // Логируем запрос

	if cmd.Action == "start" {
		fmt.Println("Starting the service...")
	} else if cmd.Action == "stop" {
		fmt.Println("Stopping the service on port 8081...")

		// Отправляем запрос на остановку внешнего сервиса
		resp, err := http.Post("http://localhost:8081/stop", "application/json", nil)
		if err != nil {
			fmt.Println("Error: Failed to send stop request to service:", err)
			http.Error(w, "Failed to stop the service", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Service stopped successfully")
		} else {
			fmt.Println("Error: Service responded with status", resp.StatusCode)
			http.Error(w, "Service failed to stop", http.StatusInternalServerError)
			return
		}

		// Ожидаем некоторое время перед остановкой самого агента
		fmt.Println("Waiting before stopping the agent...")
		time.Sleep(2 * time.Second) // Ожидание для завершения внешних процессов

		fmt.Println("Stopping agent...")
		go func() {
			// Останавливаем сервер агента
			if err := server.Shutdown(context.Background()); err != nil {
				fmt.Println("Error stopping agent server:", err)
			} else {
				fmt.Println("Agent server stopped successfully.")
			}
		}()
	} else {
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Command %s executed successfully", cmd.Action)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/command", handleCommand)

	server = &http.Server{
		Addr:    ":8082",
		Handler: mux,
	}

	idleConnsClosed := make(chan struct{})

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig

		fmt.Println("Received termination signal. Shutting down agent...")
		if err := server.Shutdown(context.Background()); err != nil {
			fmt.Println("Error during server shutdown:", err)
		}
		close(idleConnsClosed)
	}()

	fmt.Println("Agent listening on http://localhost:8082")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println("Error starting agent:", err)
	}

	<-idleConnsClosed
	fmt.Println("Agent has stopped.")
}
