package series

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prosergeant/seriesChecker/pkg/resolver"
	"github.com/prosergeant/seriesChecker/pkg/service"
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

const hlsOrigin = "https://theatre.stloadi.live"
const hlsUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// HLSProxy проксирует HLS-запросы (m3u8, ts-сегменты) через сервер.
// CDN требует Origin: https://theatre.stloadi.live.
// Для .m3u8 файлов переписывает relative URLs на абсолютные CDN URLs,
// чтобы HLS.js мог корректно их разрешить и снова пропустить через прокси.
// GET /api/hls-proxy?url=ENCODED_CDN_URL
func (h *Handler) HLSProxy(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" || !strings.HasPrefix(rawURL, "https://") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	req.Header.Set("Origin", hlsOrigin)
	req.Header.Set("Referer", hlsOrigin+"/")
	req.Header.Set("User-Agent", hlsUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(resp.StatusCode)

	// Для .m3u8 переписываем relative URLs на абсолютные, чтобы HLS.js
	// мог снова пустить их через ProxyLoader
	if strings.Contains(rawURL, ".m3u8") {
		body = rewriteM3U8Absolute(body, rawURL)
	}

	w.Write(body) //nolint:errcheck
}

// rewriteM3U8Absolute заменяет relative URLs в M3U8 плейлисте на абсолютные CDN URLs.
func rewriteM3U8Absolute(content []byte, baseURL string) []byte {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return content
	}
	// Базовый путь — директория файла (до последнего слэша)
	lastSlash := strings.LastIndex(parsed.Path, "/")
	if lastSlash >= 0 {
		parsed.Path = parsed.Path[:lastSlash+1]
	}
	parsed.RawQuery = ""

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, "http") {
			rel, err := url.Parse(trimmed)
			if err == nil {
				lines[i] = parsed.ResolveReference(rel).String()
			}
		}
	}
	// Также переписываем URI="..." в атрибутах (например, аудио-дорожки)
	result := strings.Join(lines, "\n")
	result = rewriteURIAttrs(result, parsed)
	return []byte(result)
}

// rewriteURIAttrs переписывает URI="relative" → URI="https://cdn/absolute" в строках атрибутов M3U8.
func rewriteURIAttrs(content string, base *url.URL) string {
	var sb strings.Builder
	i := 0
	for i < len(content) {
		idx := strings.Index(content[i:], `URI="`)
		if idx < 0 {
			sb.WriteString(content[i:])
			break
		}
		sb.WriteString(content[i : i+idx+5]) // до и включая URI="
		i += idx + 5
		end := strings.Index(content[i:], `"`)
		if end < 0 {
			sb.WriteString(content[i:])
			break
		}
		uri := content[i : i+end]
		if !strings.HasPrefix(uri, "http") {
			rel, err := url.Parse(uri)
			if err == nil {
				uri = base.ResolveReference(rel).String()
			}
		}
		sb.WriteString(uri)
		i += end
	}
	return sb.String()
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

	// w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, result.HTML)
	// w.Write(result.HTML)
	// json.NewEncoder(w).Encode(result.HTML)
}
