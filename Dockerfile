# Build stage
FROM golang:1.20 AS builder

# Установка рабочего каталога
WORKDIR /app

# Копирование исходных кодов
COPY . .

# Скачивание зависимостей
RUN go mod download

# Сборка приложения
RUN go build -o /app/main ./cmd

# Run stage
FROM alpine:latest

# Установка рабочего каталога
WORKDIR /app

# Копирование собранного бинарного файла и веб-ресурсов
COPY --from=builder /app/main .
COPY web /app/web
COPY prometheus.yml /etc/prometheus/prometheus.yml

# Запуск приложения
ENTRYPOINT ["/app/main"]
