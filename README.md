# System Metrics Monitoring Tool

This tool captures system metrics (CPU, memory, disk usage, and process information) and allows remote access via a REST API. It can also be used locally via command-line interface (CLI) commands.

## Features

- **Continuous Metrics Capture**: Capture system metrics at regular intervals.
- **Serialization**: Store captured metrics in a JSON file.
- **REST API**: Start and stop metric capture, and retrieve metrics remotely via HTTP requests.
- **CLI Interface**: Capture metrics and view them directly from the command line.

## Installation

1. **Clone the repository**:

   ```bash
   git clone https://github.com/sepiropht/ghtop.git
   cd ghtop
   ```

2. **Install dependencies**:

The tool uses gopsutil for capturing system metrics. Install the dependencies:

```bash```
go get github.com/tklauser/go-sysconf
go get github.com/shirou/gopsutil
go get golang.org/x/sys/unix
```

