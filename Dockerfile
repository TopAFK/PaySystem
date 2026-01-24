FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -o /toppay ./cmd/toppay

# ВАЖНО: playwright runtime (Ubuntu-based) со всеми deps
FROM mcr.microsoft.com/playwright:v1.52.0-jammy
WORKDIR /
COPY --from=build /toppay /toppay
COPY --from=build /app/configs/.env /configs/.env

CMD ["/toppay"]