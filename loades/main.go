package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

type StatusResponse struct {
	ID        string `json:"id"`
	Uptime    string `json:"uptime"`
	TaskCount int    `json:"task_count"`
}

type RequestLog struct {
	Endpoint string `json:"endpoint"`
	Action   string `json:"action"`
	Time     string `json:"time"`
}

var (
	startTime = time.Now()
	taskCount = 0
	mutex     = &sync.Mutex{}
	server    *http.Server
)

func logRequest(endpoint, action string) {
	// Открываем файл для чтения и записи (если файла нет, он будет создан)
	file, err := os.OpenFile("service_requests_log.json", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Чтение текущих данных из файла
	var logs []RequestLog
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&logs); err != nil && err.Error() != "EOF" {
		// Если ошибка при декодировании не EOF, выводим ошибку
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

	// Открываем файл для перезаписи с новыми логами
	file, err = os.OpenFile("service_requests_log.json", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Перезаписываем файл с новыми логами
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Форматируем вывод в JSON
	if err := encoder.Encode(logs); err != nil {
		fmt.Println("Error encoding JSON:", err)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	logRequest("/status", "check") // Логируем запрос

	uptime := time.Since(startTime).String()
	response := StatusResponse{
		ID:        "replica-1",
		Uptime:    uptime,
		TaskCount: taskCount,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func simulateLoad() {
	for {
		time.Sleep(time.Second * 1)
		mutex.Lock()
		taskCount += rand.Intn(10)
		mutex.Unlock()
	}
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	logRequest("/stop", "stop") // Логируем запрос

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Server stopping...\n")

	go func() {
		if err := server.Close(); err != nil {
			fmt.Println("Error stopping server:", err)
		} else {
			fmt.Println("Server stopped successfully.")
		}
	}()
}

func startServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/stop", handleStop)

	server = &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Error starting server:", err)
		}
	}()
	fmt.Println("SPP Service is running on http://localhost:8081")
}

func main() {
	startServer()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt)

	<-stopCh
	fmt.Println("Shutting down gracefully...")

	if err := server.Close(); err != nil {
		fmt.Println("Error closing server:", err)
	}
	fmt.Println("Server shutdown complete.")
}
