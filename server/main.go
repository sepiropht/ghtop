package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shirou/gopsutil/disk" // Importation ajoutée pour l'utilisation du disque
	"github.com/shirou/gopsutil/mem"
)

var db *sql.DB

type SystemMetrics struct {
	Timestamp time.Time              `json:"timestamp"`
	CPU       []float64              `json:"cpu"`
	Memory    *mem.VirtualMemoryStat `json:"memory"`
	Disk      *disk.UsageStat        `json:"disk"`
	Processes []ProcessInfo          `json:"processes"`
}

type ProcessInfo struct {
	PID    int32   `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	Memory float32 `json:"memory"`
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./metrics.db")
	if err != nil {
		log.Fatal(err)
	}

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS server (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME,
		pid INTEGER,
		name TEXT,
		cpu REAL,
		memory REAL
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
}

func fetchAndStoreData() {
	client := resty.New()
	resp, err := client.R().
		Get("http://localhost:8080/view?duration=1m") // Remplacez backend-url par l'URL réelle de votre backend

	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return
	}

	var metrics []SystemMetrics
	err = json.Unmarshal(resp.Body(), &metrics)
	if err != nil {
		log.Printf("Error unmarshalling data: %v", err)
		return
	}

	for _, metric := range metrics {
		for _, proc := range metric.Processes {
			_, err := db.Exec("INSERT INTO server (timestamp, pid, name, cpu, memory) VALUES (?, ?, ?, ?, ?)",
				metric.Timestamp, proc.PID, proc.Name, proc.CPU, proc.Memory)
			if err != nil {
				log.Printf("Error inserting into DB: %v", err)
			}
		}
	}
}

func startDataCollection() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			fetchAndStoreData()
		}
	}()
}

func getTopProcesses(w http.ResponseWriter, r *http.Request) {
	queryType := r.URL.Query().Get("type")
	durationStr := r.URL.Query().Get("duration")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		http.Error(w, "Invalid duration format", http.StatusBadRequest)
		return
	}

	threshold := time.Now().Add(-duration)
	var rows *sql.Rows

	if queryType == "cpu" {
		rows, err = db.Query("SELECT pid, name, cpu, memory FROM server WHERE timestamp >= ? ORDER BY cpu DESC LIMIT 10", threshold)
	} else if queryType == "memory" {
		rows, err = db.Query("SELECT pid, name, cpu, memory FROM server WHERE timestamp >= ? ORDER BY memory DESC LIMIT 10", threshold)
	} else {
		http.Error(w, "Invalid type. Must be 'cpu' or 'memory'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var processes []ProcessInfo
	for rows.Next() {
		var proc ProcessInfo
		if err := rows.Scan(&proc.PID, &proc.Name, &proc.CPU, &proc.Memory); err != nil {
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			return
		}
		processes = append(processes, proc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}

func serveHomePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func runServer() {
	http.HandleFunc("/", serveHomePage)
	http.HandleFunc("/top", getTopProcesses)

	fmt.Println("Starting server on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func main() {
	initDB()
	startDataCollection()
	runServer()
}
