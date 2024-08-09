# GHTOP

A dumb metrics capture

1. **Metrics Capture Module**: A Go module that continuously captures system metrics (CPU, memory, disk usage, and process information) and stores them in a JSON file.
2. **Web Server Module**: A Go-based HTTP server that retrieves the captured metrics via an API, and presents the data in a web interface.

## Table of Contents

- [System Metrics Capture](#system-metrics-capture)
- [Web Server Module](#web-server-module)
- [Setup and Running the Project](#setup-and-running-the-project)
- [Endpoints](#endpoints)
- [License](#license)

## System Metrics Capture

This component is responsible for capturing system metrics at regular intervals. It uses the `gopsutil` library to gather information about CPU, memory, disk usage, and running processes.

### Features

- **Capture Metrics**: Captures CPU, memory, disk
- **Command-Line Interface**: Includes command-line flags to start/stop capturing metrics or view the captured data.

### Usage

- **Start Capturing**:
  ```bash
  go run main.go -capture
  ```

## Web Server Module
This component provides an HTTP API to interact with the metrics captured by the Metrics Capture Module. It also serves a simple web interface that allows users to view the top CPU and memory-consuming processes.

### Features

REST API: Provides endpoints to start/stop capturing metrics and view captured data.
Web Interface: Serves a web page displaying the top 10 CPU and memory-consuming processes, with the ability to select the time range for the displayed data.
Endpoints
/capture (POST, DELETE)

POST: Start capturing metrics.
DELETE: Stop capturing metrics.
/view (GET)

Query parameters:
duration: The duration of time for which to view the metrics (e.g., 1h, 30m).
Returns captured metrics in JSON format.
Web Interface
The web interface allows users to select a time duration (e.g., 1 minute, 5 minutes, 1 hour) and displays the top 10 CPU and memory-consuming processes in two separate tables.

### Why I Did This

I wanted a simple solution to monitor my server's CPU, RAM usage, and running processes through a web interface, with a history feature to help identify what might have caused crashes. I found existing tools like Grafana, Prometheus, etc., to be overly complex and overwhelming, often providing too much data that I didn’t need. So, I decided to build a straightforward, custom tool that gives me exactly what I need—a remote "htop" with historical data—without the unnecessary complexity.
