package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/yourusername/message-processor/internal/handler"
	"github.com/yourusername/message-processor/internal/repository"
	"github.com/yourusername/message-processor/internal/service"
)

func main() {
	// Настройка логгера
	logger, err := zap.NewProduction()
	if err != nil {
		// Логируем ошибку при создании логгера и завершаем выполнение
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("Starting the application")

	// Подключение к базе данных
	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Fatal("Failed to connect to the database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close the database connection", zap.Error(err))
		}
	}()

	// Создание Kafka продюсера
	kafkaProducer, err := sarama.NewAsyncProducer([]string{os.Getenv("KAFKA_BROKER")}, nil)
	if err != nil {
		logger.Fatal("Failed to create Kafka producer", zap.Error(err))
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			logger.Error("Failed to close Kafka producer", zap.Error(err))
		}
	}()

	repo := repository.NewRepository(db, logger)
	svc := service.NewService(repo, kafkaProducer, logger)
	h := handler.NewHandler(svc, logger)

	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", http.FileServer(http.Dir("./web")))

	server := &http.Server{
		Addr:    ":8080",
		Handler: h.InitRoutes(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server ListenAndServe", zap.Error(err))
		}
	}()

	logger.Info("HTTP server started on :8080")

	// Обработка сигналов завершения работы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down the server...")

	// Завершение работы сервера
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exiting")
}
