package series

import (
	"encoding/json"
	"net/http"

	"github.com/prosergeant/seriesChecker/internal/service"
)

type Handler struct {
	seriesService *service.SeriesService
}

func NewHandler(seriesService *service.SeriesService) *Handler {
	return &Handler{seriesService: seriesService}
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
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(results)
}
