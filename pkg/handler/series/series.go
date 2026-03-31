package series

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"

	"github.com/prosergeant/seriesChecker/pkg/resolver"
	"github.com/prosergeant/seriesChecker/pkg/service"
)

// chromeClient — HTTP-клиент с TLS fingerprint Chrome.
// Нужен потому что theatre.stloadi.live блокирует Go default TLS (JA3 fingerprint).
// ForceAttemptHTTP2 не работает с custom DialTLS, поэтому пробуем h2 через ALPN,
// а при fallback используем HTTP/1.1.
var chromeClient = newChromeClient()

// cdnClient — стандартный HTTP/2 клиент для CDN запросов.
// CDN-токены выдаются Chrome который использует h2 — серверы могут отклонять HTTP/1.1.
// Если задана CDN_PROXY_URL — запросы идут через неё (для проверки IP-бана).
var cdnClient = newCDNClient()

func newCDNClient() *http.Client {
	t := &http.Transport{
		ForceAttemptHTTP2: true,
	}
	if proxyURLStr := os.Getenv("CDN_PROXY_URL"); proxyURLStr != "" {
		if proxyURL, err := url.Parse(proxyURLStr); err == nil {
			t.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return &http.Client{Transport: t, Timeout: 30 * time.Second}
}

func newChromeClient() *http.Client {
	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(network, addr, 10*time.Second)
			if err != nil {
				return nil, err
			}
			host, _, _ := net.SplitHostPort(addr)
			tlsConn := utls.UClient(conn, &utls.Config{
				ServerName: host,
				// Только http/1.1 — Go http.Transport не умеет h2 через custom DialTLS
				NextProtos: []string{"http/1.1"},
			}, utls.HelloCustom)
			// Берём Chrome spec и заменяем ALPN на http/1.1 only
			spec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("utls spec: %w", err)
			}
			// Заменяем ALPN extension
			for i, ext := range spec.Extensions {
				if alpn, ok := ext.(*utls.ALPNExtension); ok {
					spec.Extensions[i] = &utls.ALPNExtension{AlpnProtocols: []string{"http/1.1"}}
					_ = alpn
					break
				}
			}
			if err := tlsConn.ApplyPreset(&spec); err != nil {
				conn.Close()
				return nil, fmt.Errorf("apply preset: %w", err)
			}
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				conn.Close()
				return nil, err
			}
			return tlsConn, nil
		},
		DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

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

// sessionEntry хранит cookies, полученные при загрузке страницы театра.
// Используется для проксирования /bnsi/ запросов с правильной сессией.
type sessionEntry struct {
	cookies   []*http.Cookie
	createdAt time.Time
}

// proxySessions — кэш cookies по session ID (TTL ~10 минут, чистится лениво).
var proxySessions sync.Map

func genSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getSessionCookies(sessionID string) []*http.Cookie {
	v, ok := proxySessions.Load(sessionID)
	if !ok {
		return nil
	}
	entry := v.(*sessionEntry)
	if time.Since(entry.createdAt) > 10*time.Minute {
		proxySessions.Delete(sessionID)
		return nil
	}
	return entry.cookies
}

const theatreBase = "https://theatre.stloadi.live"
const hlsOrigin = theatreBase
const hlsUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"

// TheatreProxy проксирует запросы к theatre.stloadi.live через сервер,
// чтобы браузер мог обращаться к API плеера без CORS-блокировок.
// /api/theatre-proxy?url=ENCODED_THEATRE_URL (любой метод)
func (h *Handler) TheatreProxy(w http.ResponseWriter, r *http.Request) {
	// Preflight CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" || !strings.HasPrefix(rawURL, theatreBase) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Читаем тело для логирования и пересылки
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, rawURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	// Пробрасываем все заголовки от клиента, затем перезаписываем критичные
	for k, vv := range r.Header {
		lower := strings.ToLower(k)
		// Пропускаем hop-by-hop, служебные и cookies (могут сломать upstream)
		if lower == "host" || lower == "connection" || lower == "accept-encoding" || lower == "cookie" {
			continue
		}
		for _, v := range vv {
			req.Header.Set(k, v)
		}
	}
	// Перезаписываем критичные заголовки для upstream
	req.Header.Set("User-Agent", hlsUserAgent)
	req.Header.Set("Referer", theatreBase+"/")
	req.Header.Del("Origin") // chromedp не шлёт Origin для /bnsi/
	req.Header.Set("Sec-Fetch-Dest", "iframe")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	// Прикрепляем cookies из сохранённой сессии (если передан session ID)
	if sid := r.URL.Query().Get("session"); sid != "" {
		for _, c := range getSessionCookies(sid) {
			req.AddCookie(c)
		}
	}

	log.Printf("TheatreProxy: %s %s headers=%v body=%q", r.Method, rawURL, req.Header, string(bodyBytes))

	resp, err := chromeClient.Do(req)
	if err != nil {
		log.Printf("TheatreProxy: error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	preview := string(respBody)
	if len(preview) > 300 {
		preview = preview[:300]
	}
	log.Printf("TheatreProxy: resp status=%d len=%d preview=%q", resp.StatusCode, len(respBody), preview)

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody) //nolint:errcheck
}

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

	// CDN-токены привязаны к TLS-сессии Chrome (chromedp).
	// Go-клиент с тем же IP получает 403. Используем FetchCDNURL — fetch() внутри
	// живого Chrome-браузера с правильной TLS-сессией.
	body, ct, err := h.resolver.FetchCDNURL(rawURL)
	if err != nil {
		log.Printf("HLSProxy: FetchCDNURL failed (%v), falling back to Go client", err)
		// Fallback: прямой Go-запрос (может 403ить, но лучше чем ничего)
		req, reqErr := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
		if reqErr != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		req.Header.Set("Origin", hlsOrigin)
		req.Header.Set("Referer", hlsOrigin+"/")
		req.Header.Set("User-Agent", hlsUserAgent)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("Authorization", "Bearer pXzvbyDGLYyB6VkwsWZDv3iMKZtsXNzpzRyxZUcsKHXxsSeaYakbo3hw9mBFRc5VQTpqAX6BW8aDEqyLaHYcXSQiV6KHYTVTK6MYRphNAy5sBjtrevqkDzKmLqNdfMZGEU9NELjmtKfZy3RNGzCd767sNh1mXEj4tCcvqndHtzmwAbZNkhm4ghDEasodotMBewypNQ56uotJAQGX11csfeRfBAPk8DcUWWkkqzxca8vbnEw12vUFbBzT6hz8ZB3F3dzUhUXoL2cr1WM1bXQArRCS1MUNMz3X5WDMMQoZKxj2AMTRqp7QQX4dDB9B7VzEZTmyFULhm1AcHHMkoMvSVvKYoBoAKLycYAgMHeD4ECJcGEAGpnkJhrV57zQ7")
		resp, doErr := cdnClient.Do(req)
		if doErr != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		body, _ = io.ReadAll(resp.Body)
		ct = resp.Header.Get("Content-Type")
		if resp.StatusCode != 200 {
			log.Printf("HLSProxy: fallback CDN returned %d", resp.StatusCode)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
			w.WriteHeader(resp.StatusCode)
			return
		}
	}

	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Для .m3u8 переписываем все URLs через наш прокси
	if strings.Contains(rawURL, ".m3u8") {
		body = rewriteM3U8ThroughProxy(body, rawURL)
	}

	w.Write(body) //nolint:errcheck
}

func (h *Handler) HLSProxyV2(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" || !strings.HasPrefix(targetURL, "https://") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 2. Создаем новый запрос к целевому серверу
	proxyReq, err := http.NewRequest(r.Method, targetURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Копируем заголовки из оригинального запроса (если нужно)
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// 4. ПЕРЕЗАПИСЫВАЕМ критические заголовки
	proxyReq.Header.Set("Origin", "https://theatre.stloadi.live")
	// proxyReq.Header.Set("Referer", "http://localhost:8080/api/series/861614/player?token_movie=4f942c2ef97f5097b4690e45316e0a&season=1&episode=1&token=45e20a5f584becf7a64dffb7174ddf")

	// 5. Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// 6. Добавляем CORS-заголовки, чтобы ваш браузер (localhost) принял ответ
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	// Если это preflight-запрос OPTIONS, просто отвечаем OK
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Копируем заголовки ответа (кроме Transfer-Encoding и Content-Length, которые могут быть неактуальны)
	for name, values := range resp.Header {
		lower := strings.ToLower(name)
		if lower == "transfer-encoding" || lower == "content-length" {
			continue
		}
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Для .m3u8 переписываем все URLs через наш прокси V2
	if strings.Contains(targetURL, ".m3u8") {
		body = rewriteM3U8ThroughProxyV2(body, targetURL)
		// Перезаписываем Content-Length после rewrite (размер изменился)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}
	// Для не-.m3u8 (.ts сегменты) — не меняем Content-Length, пусть остаётся от CDN

	w.WriteHeader(resp.StatusCode)
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

// rewriteM3U8ThroughProxy делает все URL в m3u8 абсолютными, затем оборачивает их
// в /api/hls-proxy?url=... чтобы HLS.js не шёл напрямую на CDN (CORS-блок).
func rewriteM3U8ThroughProxy(content []byte, baseURL string) []byte {
	// Сначала делаем все URL абсолютными
	absolute := rewriteM3U8Absolute(content, baseURL)

	lines := strings.Split(string(absolute), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "https://") {
			lines[i] = "/api/hls-proxy?url=" + url.QueryEscape(trimmed)
		}
	}
	result := strings.Join(lines, "\n")
	// URI="https://..." атрибуты тоже оборачиваем
	result = rewriteURIAttrsThroughProxy(result)
	return []byte(result)
}

// rewriteURIAttrsThroughProxy оборачивает URI="https://..." через /api/hls-proxy.
func rewriteURIAttrsThroughProxy(content string) string {
	var sb strings.Builder
	i := 0
	for i < len(content) {
		idx := strings.Index(content[i:], `URI="https://`)
		if idx < 0 {
			sb.WriteString(content[i:])
			break
		}
		sb.WriteString(content[i : i+idx+5]) // до URI="
		i += idx + 5
		end := strings.Index(content[i:], `"`)
		if end < 0 {
			sb.WriteString(content[i:])
			break
		}
		uri := content[i : i+end]
		sb.WriteString("/api/hls-proxy?url=" + url.QueryEscape(uri))
		i += end
	}
	return sb.String()
}

// rewriteURIAttrsThroughProxyV2 оборачивает URI="https://..." через /api/hls-proxyV2.
func rewriteURIAttrsThroughProxyV2(content string) string {
	var sb strings.Builder
	i := 0
	for i < len(content) {
		idx := strings.Index(content[i:], `URI="https://`)
		if idx < 0 {
			sb.WriteString(content[i:])
			break
		}
		sb.WriteString(content[i : i+idx+5]) // до URI="
		i += idx + 5
		end := strings.Index(content[i:], `"`)
		if end < 0 {
			sb.WriteString(content[i:])
			break
		}
		uri := content[i : i+end]
		sb.WriteString("/api/hls-proxyV2?url=" + url.QueryEscape(uri))
		i += end
	}
	return sb.String()
}

// rewriteM3U8ThroughProxyV2 делает все URL в m3u8 абсолютными, затем оборачивает их
// в /api/hls-proxyV2?url=...
func rewriteM3U8ThroughProxyV2(content []byte, baseURL string) []byte {
	absolute := rewriteM3U8Absolute(content, baseURL)

	lines := strings.Split(string(absolute), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "https://") {
			lines[i] = "/api/hls-proxyV2?url=" + url.QueryEscape(trimmed)
		}
	}
	result := strings.Join(lines, "\n")
	result = rewriteURIAttrsThroughProxyV2(result)
	return []byte(result)
}

// TheatreStatic проксирует статические ресурсы (img, css, js, fonts) с theatre.stloadi.live.
// Все relative URL в HTML плеера резолвятся через <base href="/api/theatre-static/">,
// поэтому браузер обращается сюда вместо прямого запроса к theatre.
// GET /api/theatre-static/{path...}
func (h *Handler) TheatreStatic(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Извлекаем путь — поддерживаем /api/theatre-static/{path} и catch-all
	// для /images/{path}, /build/{path} (абсолютные пути из JS плеера)
	path := strings.TrimPrefix(r.URL.Path, "/api/theatre-static/")
	if path == r.URL.Path {
		// Не содержит /api/theatre-static/ — значит это catch-all роут,
		// путь начинается с / (напр. /images/allplay.svg) — убираем ведущий /
		path = strings.TrimPrefix(r.URL.Path, "/")
	}
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	targetURL := theatreBase + "/" + path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	req.Header.Set("User-Agent", hlsUserAgent)
	req.Header.Set("Referer", theatreBase+"/")
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Пробрасываем Content-Type и Cache-Control
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "" {
		w.Header().Set("Cache-Control", cc)
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body) //nolint:errcheck
}

// BNSIProxy проксирует POST /bnsi/ запросы на theatre.stloadi.live через chromedp.
func (h *Handler) BNSIProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ref := r.Header.Get("Referer")
	if ref == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(ref)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	q := parsed.Query()
	tokenMovie := q.Get("token_movie")
	token := q.Get("token")
	translation := q.Get("translation")
	season := q.Get("season")
	episode := q.Get("episode")

	if tokenMovie == "" || token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	theatreURL := theatreBase + "/?" + url.Values{
		"token_movie": {tokenMovie},
		"token":       {token},
		"translation": {translation},
		"season":      {season},
		"episode":     {episode},
	}.Encode()

	log.Printf("BNSIProxy: theatre_url=%s", theatreURL)

	result, err := h.resolver.ResolveStreamFullWithToken(r.Context(), theatreURL, token)
	if err != nil {
		log.Printf("BNSIProxy: ResolveStreamFullWithToken error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	log.Printf("BNSIProxy: bnsi_response len=%d", len(result.BNSIResponse))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result.BNSIResponse))
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

// ResolveV2 отдаёт wrapper-страницу с iframe на /api/series/{id}/player.
// Iframe загружает проксированный HTML theatre (same-origin), в который инжектирован
// postMessage-скрипт для трекинга прогресса (сезон, серия, перевод, время).
// Wrapper слушает postMessage и может сохранять прогресс через API.
// GET /api/series/{id}/resolveV2
func (h *Handler) ResolveV2(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid id"})
		return
	}

	log.Printf("ResolveV2 called for series id=%d", id)

	playerURL := fmt.Sprintf("/api/series/%d/player", id)
	if qs := r.URL.RawQuery; qs != "" {
		playerURL += "?" + qs
	}

	// straightLink := "https://theatre.stloadi.live/?token_movie=4f942c2ef97f5097b4690e45316e0a&translation=34&season=1&episode=1&token=45e20a5f584becf7a64dffb7174ddf"

	html := fmt.Sprintf(`<!DOCTYPE html>
<html><head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>*{margin:0;padding:0}html,body{width:100%%;height:100%%;overflow:hidden;background:#000}iframe{width:100%%;height:100%%;border:none}</style>
</head><body>
<iframe id="player" src="%s" allow="autoplay;fullscreen;encrypted-media" allowfullscreen></iframe>
<script>
window.addEventListener('message', function(e) {
  if (!e.data || e.data.type !== 'theatre-progress') return;
  //   console.log('theatre-progress:', JSON.stringify(e.data));
  console.log('theatre-progress:', e.data);
  // e.data: { type, season, episode, translation, currentTime, duration, paused }
  // TODO: отправлять на /api/progress
});
</script>
</body></html>`, playerURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// Player проксирует HTML theatre.stloadi.live через наш сервер.
// HTML патчится: isFramed блокируется, URL переписываются через прокси,
// инжектируется postMessage-скрипт для трекинга прогресса.
// GET /api/series/{id}/player
func (h *Handler) Player(w http.ResponseWriter, r *http.Request) {
	// Формируем URL theatre с query параметрами
	theatreURL := theatreBase + "/?" + r.URL.RawQuery
	if r.URL.RawQuery == "" {
		// fallback — захардкоженный токен для тестов
		theatreURL = theatreBase + "/?token_movie=4f942c2ef97f5097b4690e45316e0a&translation=34&token=45e20a5f584becf7a64dffb7174ddf"
	}

	log.Printf("Player: fetching %s", theatreURL)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, theatreURL, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	req.Header.Set("User-Agent", hlsUserAgent)
	req.Header.Set("Referer", theatreBase+"/")
	req.Header.Set("Sec-Fetch-Dest", "iframe")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	resp, err := chromeClient.Do(req)
	if err != nil {
		log.Printf("Player: fetch error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Player: theatre returned %d", resp.StatusCode)
		w.WriteHeader(resp.StatusCode)
		w.Write(body) //nolint:errcheck
		return
	}

	html := patchPlayerHTML(string(body), r.URL.Query())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// patchPlayerHTML патчит HTML плеера theatre для работы через прокси:
// 1. base href → static assets идут через /api/theatre-static/
// 2. isFramed = true → плеер не удаляет себя
// 3. fetch/XHR interceptor → /bnsi/ через TheatreProxy, HLS через HLSProxy
// 4. postMessage tracker → отправляет прогресс в parent window
func patchPlayerHTML(html string, params url.Values) string {
	// Блокируем isFramed проверки
	// html = strings.Replace(html, "if(!isFramed){document.querySelectorAll('body')[0].remove()", "if(false && isFramed){document.querySelectorAll('body')[0].remove()", 1)
	// html = strings.Replace(html, "if(!isFramed){document.querySelectorAll('html')[0].innerHTML=", "if(false){document.querySelectorAll('html')[0].innerHTML=", 1)

	// base href для static assets
	// html = strings.Replace(html, "<head>", `<head><base href="/api/theatre-static/">`, 1)

	// // Переписываем relative ./js/ пути
	html = strings.ReplaceAll(html, `"./js/`, `"/api/theatre-static/js/`)
	html = strings.ReplaceAll(html, `'./js/`, `'/api/theatre-static/js/`)

	// // Абсолютные пути /build/, /images/, /fonts/ → через прокси
	// for _, prefix := range []string{"build", "images", "fonts"} {
	// 	html = strings.ReplaceAll(html, `"/`+prefix+`/`, `"/api/theatre-static/`+prefix+`/`)
	// 	html = strings.ReplaceAll(html, `'/`+prefix+`/`, `'/api/theatre-static/`+prefix+`/`)
	// 	html = strings.ReplaceAll(html, `url(/`+prefix+`/`, `url(/api/theatre-static/`+prefix+`/`)
	// }

	// // Абсолютные URL theatre.stloadi.live → прокси (кроме blob:)
	// html = strings.ReplaceAll(html, "blob:https://theatre.stloadi.live/", "blob:__THEATRE_BLOB__")
	// html = strings.ReplaceAll(html, "https://theatre.stloadi.live/", "/api/theatre-static/")
	// html = strings.ReplaceAll(html, "blob:__THEATRE_BLOB__", "blob:https://theatre.stloadi.live/")

	// Инжектируем скрипты сразу после <base>
	scripts := playerInterceptorScript() + playerPostMessageScript()
	// html = strings.Replace(html,
	// 	`<base href="/api/theatre-static/">`,
	// 	`<base href="/api/theatre-static/">`+scripts,
	// 	1,
	// )

	html = strings.Replace(html,
		`<head>`,
		`<head>`+scripts,
		1,
	)

	return html
}

// playerInterceptorScript — перехватывает fetch/XHR:
// /bnsi/* → вызывает /api/series/stream через chromedp для получения m3u8, инициализирует HLS.js
// CDN .m3u8/.ts → /api/hls-proxy (HLS потоки)
func playerInterceptorScript() string {
	return `<script>
Object.defineProperty(window,'isFramed',{value:true,writable:false,configurable:false});
(function(){
  var H='/api/hls-proxyV2?url=';
  var THEATRE='https://theatre.stloadi.live';
  var _hlsInstance=null;

  // Инициализирует HLS.js плеер с m3u8 URL
  function initHLS(m3u8){
    var v=document.querySelector('video');
    if(!v) return;
    // Убираем loader overlay
    var loader=document.querySelector('.loader.active');
    if(loader) loader.classList.remove('active');
    // Уничтожаем предыдущий инстанс
    if(_hlsInstance){try{_hlsInstance.destroy();}catch(e){}}
    // CDN токены привязаны к IP сервера — проксируем через сервер
    var proxyM3u8=H+encodeURIComponent(m3u8);
    if(typeof Hls!=='undefined'&&Hls.isSupported()){
      var hls=new Hls({
        maxBufferLength:30,
        maxMaxBufferLength:60,
        xhrSetup:function(xhr,url){
          // Все CDN URLs перенаправляем через hls-proxy
          if(url.indexOf('https://')===0&&url.indexOf('/api/')<0){
            xhr.open('GET',H+encodeURIComponent(url),true);
          }
        },
        fetchSetup:function(context){
          // Также проксируем fetch запросы
          var url=context.url;
          if(url.indexOf('https://')===0&&url.indexOf('/api/')<0){
            console.log('[initHLS] fetch redirect:', url.substring(0,80));
            return new Request(H+encodeURIComponent(url),context);
          }
          return context;
        }
      });
      _hlsInstance=hls;
      hls.loadSource(proxyM3u8);
      hls.attachMedia(v);
      hls.on(Hls.Events.MANIFEST_PARSED,function(){console.log('[initHLS] manifest parsed');v.play().catch(function(e){console.log('[initHLS] play error:',e);});});
      hls.on(Hls.Events.ERROR,function(e,data){console.log('[initHLS] error:',data.type,data.details,data.reason);});
    } else if(v.canPlayType('application/vnd.apple.mpegurl')){
      v.src=proxyM3u8;
      v.addEventListener('loadedmetadata',function(){v.play().catch(function(e){console.log('[initHLS] native play error:',e);});});
    } else {
      console.log('[initHLS] HLS not supported');
    }
  }

  // Перехват fetch: /bnsi/ → наш stream endpoint через chromedp
  // уже не надо перехватывать, оно перехватывается через бек ("POST /bnsi/", seriesHandler.BNSIProxy)
  const _f=window.fetch;
  window.fetch_backup = function(input,init){
    var u = (typeof input === 'string') ? input : (input && input.url) || '';
    // /bnsi/ запросы → резолвим через chromedp
    if(typeof u === 'string' && (u.indexOf('/bnsi/') === 0 || u.indexOf(THEATRE + '/bnsi/') === 0)){
      console.log('[interceptor] /bnsi/ intercepted, resolving via chromedp...');
      // Получаем текущие параметры theatre из URL страницы
      var qs=window.location.search||'?'+document.querySelector('base').href.split('?')[1]||'';
      fetch('/api/series/stream'+qs)
        .then(function(r){return r.json();})
        .then(function(data){
          console.log('[interceptor] got stream data, keys:', Object.keys(data));
          if(data.bnsi_response){
            console.log('[interceptor] got bnsi_response, parsing...');
            try {
              var bnsi = JSON.parse(data.bnsi_response);
              if(bnsi.hlsSource && bnsi.hlsSource[0] && bnsi.hlsSource[0].quality){
                var q = bnsi.hlsSource[0].quality;
                var best = q['1080'] || q['720'] || q['480'] || q['360'];
                if(best){
                  var urls = best.split(' or ');
                  // Try second URL first (might be different CDN)
                  var useUrl = urls.length > 1 ? urls[1] : urls[0];
                  console.log('[interceptor] using quality URL from bnsi:', useUrl.substring(0,80));
                  initHLS(useUrl);
                  return;
                }
              }
            } catch(e){console.error('[interceptor] bnsi parse error:', e);}
          }
          if(data.m3u8_url){
            console.log('[interceptor] got m3u8:', data.m3u8_url);
            initHLS(data.m3u8_url);
          } else if(data.m3u8_all && data.m3u8_all.length > 0){
            console.log('[interceptor] got all m3u8:', data.m3u8_all);
            initHLS(data.m3u8_all[0]);
          } else if(data.html){
            console.log('[interceptor] got html, injecting...');
            document.open();
            document.write(data.html);
            document.close();
          }
        })
        .catch(function(err){
          console.error('[interceptor] stream resolve failed:', err);
        });
      // Возвращаем pending promise — theatre остаётся в состоянии "загрузка",
      // не показывает error UI (TypeError), пока initHLS не запустит видео
      return new Promise(function(){});
    }
    // CDN HLS запросы — пропускаем напрямую (CORS разрешён для *)
    return _f.call(this,input,init);
  };

  // Перехват XHR (fallback, theatre может использовать XHR вместо fetch)
  var _o=XMLHttpRequest.prototype.open;
  var _s=XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.open=function(method,url){
    this._interceptedURL=url;
    // HLS CDN — CORS разрешён, не проксируем
    return _o.apply(this,arguments);
  };
  XMLHttpRequest.prototype.send=function(body){
    const url = this._interceptedURL || '';

	const data = {type: 'theatre-progress'};
	data.url = url
	data.body = body
    window.parent.postMessage(data,'*');

	if(typeof url !== 'string') return _s.apply(this,arguments);

	if(!url.includes('?url=') && url.includes('.m3u8')) { 
		// почему то на master.m3u8 делается полноценный запрос
		// а на index-f1-v1 относительны и не понятно как оно работает в оригинале хз
		// поэтому сохраним урл от мастера
		if(url.includes('master.m3u8')) {
			window.baseMasterUrl = url.replace('master.m3u8', '')
		}
		var self = this;
		const resultUrl = url.startsWith('https://') ? H + url : H + window.baseMasterUrl + url;
		fetch(resultUrl)
			.then(r => r.text())
			.then(text => {
				Object.defineProperty(self, 'readyState', {get: function(){ return 4; }, configurable: true});
				Object.defineProperty(self, 'status', {get: function(){ return 200; }, configurable: true});
				Object.defineProperty(self, 'responseText', {get: function(){ return text; }, configurable: true});
				Object.defineProperty(self, 'response', {get: function(){ return text; }, configurable: true});
				if(typeof self.onreadystatechange === 'function') self.onreadystatechange();
				if(typeof self.onload === 'function') self.onload();
			})
			.catch(() => {});
		return;
	}

	return _s.apply(this,arguments);
  }
})();
</script>`
}

func oldInterceptorXhr() string {
	return `
	if(url.indexOf('/bnsi/') === 0 || url.indexOf(THEATRE+'/bnsi/') === 0){
      const qs=window.location.search || '';
      const self=this;
        fetch('/api/series/stream'+qs)
        .then(function(r){return r.json();})
        .then(function(data){
          if(data.m3u8_url){
            initHLS(data.m3u8_url);
          } else if(data.m3u8_all && data.m3u8_all.length > 0){
            initHLS(data.m3u8_all[0]);
          } else if(data.html){
            document.open();
            document.write(data.html);
            document.close();
          }
          // Не вызываем onload/onreadystatechange — XHR остаётся pending,
          // theatre не получает ответ и не показывает error UI
        })
        .catch(function(err){console.error('[interceptor] XHR stream failed:', err);});
      return;
    }`
}

// Stream использует chromedp для загрузки theatre страницы и возвращает все данные.
// Нужен потому что /bnsi/ API theatre привязан к серверной сессии браузера —
// только chromedp (полноценный браузер) может корректно пройти весь flow.
// GET /api/series/stream?token_movie=...&translation=...&token=...
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	theatreURL := theatreBase + "/?" + r.URL.RawQuery
	if r.URL.RawQuery == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query params required"})
		return
	}

	log.Printf("Stream: resolving %s", theatreURL)

	result, err := h.resolver.ResolveStreamFull(r.Context(), theatreURL)
	if err != nil {
		log.Printf("Stream: error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to resolve stream"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(result)
}

// playerPostMessageScript — мониторит состояние плеера и шлёт postMessage
// в parent window с данными о текущем сезоне, серии, переводе и позиции.
func playerPostMessageScript() string {
	return `<script>
(function(){
  var last='';
  function send(){
    var data={type:'theatre-progress'};
    // Кнопки выбора: ищем активные элементы
    var btns=document.querySelectorAll('button');
    btns.forEach(function(b){
      var txt=b.textContent.trim();
      // Определяем по тексту кнопки
      //   if(/^Сезон\s+\d+/.test(txt)) data.season=txt.replace('Сезон ','');
      //   if(/^Серия\s+\d+/.test(txt)) data.episode=txt.replace('Серия ','');
	  // достаем с localStorage
	  try {
	  	const url = new URL(window.location.href);
	  	const parsedData = JSON.parse(localStorage.getItem('save-' + url.searchParams.get('token_movie')));
		data.serial = parsedData.serial
	  } catch {}
      // Перевод — кнопка в шапке плеера (рядом с Сезон/Серия), исключаем контролы и служебные
      if(b.closest&&!b.closest('[class*="control"]')&&!/Сезон|Серия|Воспроизвести|Предыдущая|Следующая|Настройки|PIP|Полноэкранный|Выключить|Пауза|Отправить|space|stat/.test(txt)&&txt.length>1&&txt.length<30){
        data.translation=txt;
      }
    });
    var v=document.querySelector('video');
    if(v){
      data.currentTime=Math.floor(v.currentTime);
      data.duration=Math.floor(v.duration)||0;
      data.paused=v.paused;
    }
    var key=JSON.stringify(data);
    if(key!==last){
      last=key;
      try{window.parent.postMessage(data,'*');}catch(e){}
    }
  }
  // Запускаем через 2 секунды (ждём инициализацию плеера), потом каждые 3 секунды
  setTimeout(function(){
    send();
    setInterval(send,3000);
  },2000);
})();
</script>`
}
