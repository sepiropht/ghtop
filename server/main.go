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
	"github.com/shirou/gopsutil/disk"
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

type Server struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
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
		memory REAL,
		server_id INTEGER
	);
	CREATE TABLE IF NOT EXISTS servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		url TEXT
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
}

func addServerToDB(name, url string) error {
	_, err := db.Exec("INSERT INTO servers (name, url) VALUES (?, ?)", name, url)
	return err
}

func getServersFromDB() ([]Server, error) {
	rows, err := db.Query("SELECT id, name, url FROM servers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var server Server
		if err := rows.Scan(&server.ID, &server.Name, &server.URL); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func fetchAndStoreData() {
	servers, err := getServersFromDB()
	if err != nil {
		log.Printf("Error fetching servers: %v", err)
		return
	}

	client := resty.New()

	for _, server := range servers {
		resp, err := client.R().
			Get(fmt.Sprintf("%s/view?duration=1m", server.URL))

		if err != nil {
			log.Printf("Error fetching data from server %s: %v", server.Name, err)
			continue
		}

		var metrics []SystemMetrics
		err = json.Unmarshal(resp.Body(), &metrics)
		if err != nil {
			log.Printf("Error unmarshalling data from server %s: %v", server.Name, err)
			continue
		}

		for _, metric := range metrics {
			for _, proc := range metric.Processes {
				_, err := db.Exec("INSERT INTO server (timestamp, pid, name, cpu, memory, server_id) VALUES (?, ?, ?, ?, ?, ?)",
					metric.Timestamp, proc.PID, proc.Name, proc.CPU, proc.Memory, server.ID)
				if err != nil {
					log.Printf("Error inserting into DB: %v", err)
				}
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

func getServersHandler(w http.ResponseWriter, r *http.Request) {
	servers, err := getServersFromDB()
	if err != nil {
		http.Error(w, "Error fetching servers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func getTopProcesses(w http.ResponseWriter, r *http.Request) {
	queryType := r.URL.Query().Get("type")
	durationStr := r.URL.Query().Get("duration")
	serverID := r.URL.Query().Get("serverId")

	// Parse the duration from the query string
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		http.Error(w, "Invalid duration format", http.StatusBadRequest)
		log.Printf("Invalid duration format: %v", durationStr)
		return
	}

	threshold := time.Now().Add(-duration)
	var rows *sql.Rows

	// Build the query string based on the query type
	query := `
		SELECT pid, name, cpu, memory FROM server
		WHERE timestamp >= ? AND server_id = ? 
		ORDER BY %s DESC LIMIT 10
	`
	if queryType == "cpu" {
		rows, err = db.Query(fmt.Sprintf(query, "cpu"), threshold, serverID)
		log.Printf("Executing CPU query for server ID %s with duration %s", serverID, durationStr)
	} else if queryType == "memory" {
		rows, err = db.Query(fmt.Sprintf(query, "memory"), threshold, serverID)
		log.Printf("Executing Memory query for server ID %s with duration %s", serverID, durationStr)
	} else {
		http.Error(w, "Invalid type. Must be 'cpu' or 'memory'", http.StatusBadRequest)
		log.Printf("Invalid query type: %v", queryType)
		return
	}

	// Check if the query execution resulted in an error
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		log.Printf("Database query failed: %v", err)
		return
	}
	defer rows.Close()

	// Collect the process information from the query results
	var processes []ProcessInfo
	for rows.Next() {
		var proc ProcessInfo
		log.Printf("Error scanning row: %v", rows)
		if err := rows.Scan(&proc.PID, &proc.Name, &proc.CPU, &proc.Memory); err != nil {
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			log.Printf("Error scanning row: %v", err)
			return
		}
		processes = append(processes, proc)
	}

	// Return the JSON response with the top processes
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(processes); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		log.Printf("Error encoding JSON: %v", err)
	}
}

func addServerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	url := r.FormValue("url")

	if name == "" || url == "" {
		http.Error(w, "Missing name or URL", http.StatusBadRequest)
		return
	}

	err := addServerToDB(name, url)
	if err != nil {
		http.Error(w, "Failed to add server", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
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
	http.HandleFunc("/add-server", addServerHandler)
	http.HandleFunc("/servers", getServersHandler)

	fmt.Println("Starting server on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func main() {
	initDB()
	startDataCollection()
	runServer()
}
