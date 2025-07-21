# HTTP Monitor

A Go CLI tool that monitors HTTP endpoints using custom DNS resolution.

## Configuration

The script has two hard-coded URLs that you can modify in `monitor.go`:

- `targetURL`: The URL to monitor (default: `https://example.com`)
- `dnsServer`: The DNS server to use for resolution (default: `8.8.8.8:53`)

## Usage

1. Build the binary:
   ```bash
   go build -o monitor monitor.go
   ```

2. Run the monitor:
   ```bash
   ./monitor
   ```

The script will:
- Make an HTTP GET request every 10 seconds
- Use a 5-second timeout for each request
- Print "o" for successful requests (2xx status codes)
- Print "x" for failed requests
- Stop gracefully when you press Ctrl+C

## Features

- Custom DNS resolution using specified DNS server
- Graceful shutdown on interrupt signals
- Real-time status feedback
- 5-second request timeout
- 10-second interval between requests
# longcon
