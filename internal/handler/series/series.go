package series

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/prosergeant/seriesChecker/internal/resolver"
	"github.com/prosergeant/seriesChecker/internal/service"
)

type Handler struct {
	seriesService *service.SeriesService
	resolver      *resolver.Resolver
}

func NewHandler(seriesService *service.SeriesService) *Handler {
	return &Handler{
		seriesService: seriesService,
		resolver:      resolver.New(),
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query parameter 'q' is required"})
		return
	}

	results, err := h.seriesService.Search(r.Context(), query)
	if err != nil {
		log.Printf("search error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "id is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid id"})
		return
	}

	result, err := h.seriesService.GetByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "series not found"})
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GetSimilar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid id"})
		return
	}

	results, err := h.seriesService.GetSimilar(r.Context(), id)
	if err != nil {
		log.Printf("get similar error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (h *Handler) GetRelations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid id"})
		return
	}

	results, err := h.seriesService.GetRelations(r.Context(), id)
	if err != nil {
		log.Printf("get relations error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(results)
}

// Resolve следует редиректу с sspoisk.ru и парсит итоговую страницу
// в поисках iframe, видео URL и конфигов плеера.
// GET /api/series/{id}/resolve
func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid id"})
		return
	}

	// Получаем серию чтобы знать тип (сериал/фильм)
	seriesInfo, err := h.seriesService.GetByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "series not found"})
		return
	}

	result, err := h.resolver.Resolve(r.Context(), id, seriesInfo.IsSerial)
	if err != nil {
		log.Printf("resolve error for id=%d: %v", id, err)
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to resolve video source"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
