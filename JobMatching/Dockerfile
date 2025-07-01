# syntax=docker/dockerfile:1
FROM golang:1.24 as builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o hirepilot main.go

FROM debian:bullseye-slim
RUN apt-get update && apt-get install -y \
    ca-certificates \
    fonts-liberation \
    libappindicator3-1 \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libcups2 \
    libdbus-1-3 \
    libgdk-pixbuf2.0-0 \
    libnspr4 \
    libnss3 \
    libx11-xcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    xdg-utils \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install Chromium for chromedp and add a symlink for compatibility
RUN apt-get update && apt-get install -y chromium && \
    ln -sf /usr/bin/chromium /usr/bin/chromium-browser && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/hirepilot .
COPY config.yaml .

ENV CHROMEDP_BROWSER_PATH=/usr/bin/chromium

ENTRYPOINT ["/app/hirepilot"]
