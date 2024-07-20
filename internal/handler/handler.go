package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourusername/message-processor/internal/service"
	"go.uber.org/zap"
)

type Handler struct {
	svc    *service.Service
	logger *zap.Logger
}

func NewHandler(svc *service.Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) InitRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/messages", h.CreateMessage).Methods("POST")
	r.HandleFunc("/messages/{id:[0-9]+}/process", h.ProcessMessage).Methods("POST")
	r.HandleFunc("/statistics", h.GetStatistics).Methods("GET")
	r.Handle("/metrics", promhttp.Handler())
	return r
}

func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	content, ok := req["content"]
	if !ok {
		http.Error(w, "Missing content field", http.StatusBadRequest)
		return
	}

	if err := h.svc.ProcessMessage(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Increment message creation metric
	messageCreationCounter.Inc()

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ProcessMessage(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.svc.MarkMessageAsProcessed(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Increment message processing metric
	messageProcessingCounter.Inc()

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.GetProcessedMessagesCount()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]int{"processed_messages_count": count}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Define Prometheus metrics
var (
	messageCreationCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "message_creation_total",
		Help: "Total number of messages created",
	})
	messageProcessingCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "message_processing_total",
		Help: "Total number of messages processed",
	})
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(messageCreationCounter)
	prometheus.MustRegister(messageProcessingCounter)
}
