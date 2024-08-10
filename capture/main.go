package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

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

var (
	filename       = "htop_data.json"
	captureRunning = false
	mu             sync.Mutex
)

func captureSystemMetrics() (*SystemMetrics, error) {
	cpuPercents, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}

	memStats, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	diskUsage, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var procInfos []ProcessInfo
	for _, proc := range processes {
		name, _ := proc.Name()
		cpuPercent, _ := proc.CPUPercent()
		memPercent, _ := proc.MemoryPercent()

		procInfos = append(procInfos, ProcessInfo{
			PID:    proc.Pid,
			Name:   name,
			CPU:    cpuPercent,
			Memory: memPercent,
		})
	}

	return &SystemMetrics{
		Timestamp: time.Now(),
		CPU:       cpuPercents,
		Memory:    memStats,
		Disk:      diskUsage,
		Processes: procInfos,
	}, nil
}

func serializeMetrics(metrics *SystemMetrics, filename string) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = file.Write(data); err != nil {
		return err
	}

	if _, err = file.WriteString("\n"); err != nil {
		return err
	}

	return nil
}

func deserializeMetrics(filename string) ([]SystemMetrics, error) {
	var metrics []SystemMetrics

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var m SystemMetrics
		err := json.Unmarshal(line, &m)
		if err != nil {
			return nil, err
		}

		metrics = append(metrics, m)
	}

	return metrics, nil
}

func viewMetrics(filename string, duration time.Duration) []SystemMetrics {
	metrics, err := deserializeMetrics(filename)
	if err != nil {
		log.Printf("Error deserializing metrics: %v", err)
		return nil
	}

	threshold := time.Now().Add(-duration)
	var filteredMetrics []SystemMetrics

	for _, m := range metrics {
		if m.Timestamp.After(threshold) {
			filteredMetrics = append(filteredMetrics, m)
		}
	}

	return filteredMetrics
}

func startCapture() {
	mu.Lock()
	if captureRunning {
		mu.Unlock()
		return
	}
	captureRunning = true
	mu.Unlock()

	go func() {
		for {
			mu.Lock()
			if !captureRunning {
				mu.Unlock()
				break
			}
			mu.Unlock()

			metrics, err := captureSystemMetrics()
			if err != nil {
				log.Printf("Error capturing metrics: %v", err)
				continue
			}

			err = serializeMetrics(metrics, filename)
			if err != nil {
				log.Printf("Error serializing metrics: %v", err)
			}

			fmt.Printf("Captured metrics at %v\n", metrics.Timestamp)
			time.Sleep(10 * time.Second) // Adjust the interval as needed
		}
	}()
}

func stopCapture() {
	mu.Lock()
	captureRunning = false
	mu.Unlock()
}

func captureHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		startCapture()
		fmt.Fprintf(w, "Started capturing metrics.\n")
	case "DELETE":
		stopCapture()
		fmt.Fprintf(w, "Stopped capturing metrics.\n")
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	durationStr := r.URL.Query().Get("duration")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		http.Error(w, "Invalid duration format", http.StatusBadRequest)
		return
	}

	metrics := viewMetrics(filename, duration)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func runServer() {
	http.HandleFunc("/capture", captureHandler)
	http.HandleFunc("/view", viewHandler)

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	// Define command-line flags
	captureFlag := flag.Bool("capture", false, "Capture metrics continuously")
	viewFlag := flag.Bool("view", false, "View the captured metrics")
	durationFlag := flag.Duration("duration", 2*time.Hour, "Duration of the past metrics to view (e.g., 1h, 30m)")

	flag.Parse()

	// Start the server by default
	go runServer()

	// Handle view command
	if *viewFlag {
		metrics := viewMetrics(filename, *durationFlag)
		for _, m := range metrics {
			fmt.Printf("Timestamp: %v, CPU: %v, Memory Used: %v%%, Disk Used: %v%%\n",
				m.Timestamp, m.CPU, m.Memory.UsedPercent, m.Disk.UsedPercent)
		}
		return
	}

	// Handle capture command (default)
	if *captureFlag {
		startCapture()
		// Run capture indefinitely, as it's now the main task
		select {}
	} else {
		flag.Usage()
	}
}
