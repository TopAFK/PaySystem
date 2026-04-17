# ---------- BUILD STAGE ----------
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -o /toppay ./cmd/toppay

# ---------- RUNTIME STAGE ----------
FROM alpine:latest

# Устанавливаем Chromium для chromedp
RUN apk add --no-cache chromium

WORKDIR /app

COPY --from=builder /toppay /toppay

CMD ["/toppay"]