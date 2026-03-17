package progress

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/prosergeant/seriesChecker/internal/middleware"
	"github.com/prosergeant/seriesChecker/internal/service"
)

type Handler struct {
	progressService *service.ProgressService
}

func NewHandler(progressService *service.ProgressService) *Handler {
	return &Handler{progressService: progressService}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type UpdateProgressRequest struct {
	SeriesID       int    `json:"series_id"`
	CurrentSeason  int    `json:"current_season"`
	CurrentEpisode int    `json:"current_episode"`
	Status         string `json:"status"`
}

func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
		return
	}

	status := r.URL.Query().Get("status")
	results, err := h.progressService.GetListByStatus(r.Context(), userID, status)
	if err != nil {
		log.Printf("get progress list error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
		return
	}

	var req UpdateProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.SeriesID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "series_id is required"})
		return
	}

	result, err := h.progressService.UpdateProgress(r.Context(), userID, req.SeriesID, req.CurrentSeason, req.CurrentEpisode, req.Status)
	if err != nil {
		log.Printf("update progress error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
		return
	}

	seriesIDStr := r.PathValue("seriesId")
	seriesID, err := strconv.Atoi(seriesIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid series_id"})
		return
	}

	err = h.progressService.DeleteProgress(r.Context(), userID, seriesID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
}
