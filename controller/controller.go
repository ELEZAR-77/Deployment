package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Command struct {
	Action string `json:"action"`
}

type Replica struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ClusterStatus struct {
	Replicas []Replica `json:"replicas"`
}

type RequestLog struct {
	Endpoint string `json:"endpoint"`
	Action   string `json:"action"`
	Time     string `json:"time"`
}

var clusterStatus = ClusterStatus{
	Replicas: []Replica{
		{ID: "replica-1", Status: "running"},
		{ID: "replica-2", Status: "running"},
	},
}

func logRequest(endpoint, action string) {
	// Шаг 1: Чтение существующих логов
	var logs []RequestLog
	data, err := os.ReadFile("controller_requests_log.json") // Чтение файла полностью
	if err != nil && !os.IsNotExist(err) {                   // Игнорируем ошибку, если файл не существует
		fmt.Println("Error reading file:", err)
		return
	}

	if len(data) > 0 { // Декодируем только если файл не пустой
		if err := json.Unmarshal(data, &logs); err != nil {
			fmt.Println("Error decoding JSON:", err)
			return
		}
	}

	// Шаг 2: Добавляем новый лог
	log := RequestLog{
		Endpoint: endpoint,
		Action:   action,
		Time:     time.Now().Format(time.RFC3339),
	}
	logs = append(logs, log)

	// Шаг 3: Записываем обновленные логи в файл
	file, err := os.OpenFile("controller_requests_log.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error opening file for writing:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Форматируем JSON
	if err := encoder.Encode(logs); err != nil {
		fmt.Println("Error encoding JSON:", err)
	}
}

func handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	logRequest("/cluster-status", "status") // Логируем запрос

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clusterStatus)
}

func handleStartAgent(w http.ResponseWriter, r *http.Request) {
	logRequest("/start-agent", "start") // Логируем запрос

	// Отправляем запрос на запуск агента
	cmd := Command{Action: "start"}
	jsonData, _ := json.Marshal(cmd)
	_, err := http.Post("http://localhost:8082/command", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to start agent", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "Agent started")
}

func handleStopAgent(w http.ResponseWriter, r *http.Request) {
	logRequest("/stop-agent", "stop") // Логируем запрос

	// Отправляем запрос на остановку агента
	cmd := Command{Action: "stop"}
	jsonData, _ := json.Marshal(cmd)
	_, err := http.Post("http://localhost:8082/command", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to stop agent", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "Agent stopped")
}

func main() {
	http.HandleFunc("/cluster-status", handleClusterStatus)
	http.HandleFunc("/start-agent", handleStartAgent)
	http.HandleFunc("/stop-agent", handleStopAgent)

	// Логируем каждый запрос
	logRequest("/start-agent", "start") // Пример логирования

	fmt.Println("Server start in port: http://localhost:8080")

	http.ListenAndServe(":8080", nil)

}
