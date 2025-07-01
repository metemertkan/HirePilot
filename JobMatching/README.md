# HirePilot JobMatching

This is a Golang CLI tool that logs into LinkedIn, searches for jobs, and processes each job using browser automation (chromedp).

## Features
- Modular CLI structure (Cobra)
- Config management (Viper)
- Logging
- LinkedIn login & job search (chromedp, placeholder)

## Getting Started

1. Clone the repo and enter the directory:
   ```bash
   git clone <your-repo-url>
   cd HirePilot/JobMatching
   ```
2. Fill in your LinkedIn credentials and search preferences in `config.yaml`.
3. Build and run:
   ```bash
   go mod tidy
   go run main.go search
   ```

## Run as Docker Container

1. Build the Docker image:
   ```bash
   docker build -t hirepilot-jobmatching .
   ```
2. Run the container:
   ```bash
   docker run --rm -v $(pwd)/config.yaml:/app/config.yaml hirepilot-jobmatching
   ```

You can schedule this container to run daily using cron or any scheduler.

## Project Structure
- `cmd/` - CLI entry (Cobra)
- `internal/config/` - Config loader
- `internal/logger/` - Logging setup
- `pkg/linkedin/` - LinkedIn automation logic (chromedp)

---
**Note:** LinkedIn automation is a placeholder. Implement your logic in `pkg/linkedin/linkedin.go`.
