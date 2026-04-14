# ---------- BUILD STAGE ----------
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -o /toppay ./cmd/toppay

# ---------- RUNTIME STAGE ----------
FROM mcr.microsoft.com/playwright:v1.57.0

WORKDIR /app

COPY --from=builder /toppay /toppay

CMD ["/toppay"]