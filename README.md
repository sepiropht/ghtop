# GHTOP

A dumb metrics capture

1. **Metrics Capture Module**: A Go module that continuously captures system metrics (CPU, memory, disk usage, and process information)
2. **Web Server Module**: A Go-based HTTP server that retrieves the captured metrics via an API store them in sqlite db, and presents the data in a web interface.

## Why I Did This

I wanted a simple solution to monitor my server's CPU, RAM usage, and running processes through a web interface, with a history feature to help identify what might have caused crashes. I found existing tools like Grafana, Prometheus, etc., to be overly complex and overwhelming, often providing too much data that I didn’t need. So, I decided to build a straightforward, custom tool that gives me exactly what I need—a remote "htop" with historical data—without the unnecessary complexity.

## Usage

Clone the repo on the server that want data form, there you will start sonde, and on the server that monitore you will launch
the server

### Launch the sonde

In the sonde directory

```bash
$ docker-compose up --build
```

### Launch the web app

In the server directory

```bash
$ docker-compose up --build
```
