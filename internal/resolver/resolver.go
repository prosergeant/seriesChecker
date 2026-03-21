package resolver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Result содержит всё что удалось извлечь из страницы с видео
type Result struct {
	FinalURL   string   `json:"final_url"`
	Iframes    []string `json:"iframes"`
	VideoURLs  []string `json:"video_urls"`
	Scripts    []string `json:"scripts_with_video,omitempty"`
}

var (
	reIframeSrc = regexp.MustCompile(`(?i)<iframe[^>]+src=["']([^"']+)["']`)
	reVideoSrc  = regexp.MustCompile(`(?i)<(?:video|source)[^>]+src=["']([^"']+)["']`)
	reM3U8      = regexp.MustCompile(`["'](https?://[^"']+\.m3u8[^"']*)["']`)
	reMp4       = regexp.MustCompile(`["'](https?://[^"']+\.mp4[^"']*)["']`)
	reMpd       = regexp.MustCompile(`["'](https?://[^"']+\.mpd[^"']*)["']`)
	reScript    = regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`)
)

// videoKeywords — ключевые слова в скриптах, сигнализирующие о конфиге плеера
var videoKeywords = []string{
	"playerConfig", "videoConfig", "hlsUrl", "videoUrl",
	"file:", "source:", "hls:", "dash:", "mp4", "m3u8",
}

type Resolver struct {
	client *http.Client
}

func New() *Resolver {
	return &Resolver{
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (r *Resolver) Resolve(ctx context.Context, kinopoiskID int, isSerial bool) (*Result, error) {
	mediaType := "film"
	if isSerial {
		mediaType = "series"
	}

	startURL := fmt.Sprintf("https://www.sspoisk.ru/%s/%d", mediaType, kinopoiskID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, startURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // макс 5MB
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	body := string(bodyBytes)
	finalURL := resp.Request.URL.String()

	result := &Result{
		FinalURL: finalURL,
	}

	// Ищем iframe src
	for _, match := range reIframeSrc.FindAllStringSubmatch(body, -1) {
		if len(match) > 1 {
			result.Iframes = append(result.Iframes, match[1])
		}
	}

	// Ищем video/source src
	for _, match := range reVideoSrc.FindAllStringSubmatch(body, -1) {
		if len(match) > 1 {
			result.VideoURLs = append(result.VideoURLs, match[1])
		}
	}

	// Ищем в скриптах .m3u8 / .mp4 / .mpd URLs
	for _, scriptMatch := range reScript.FindAllStringSubmatch(body, -1) {
		if len(scriptMatch) < 2 {
			continue
		}
		scriptContent := scriptMatch[1]

		foundVideo := false
		for _, re := range []*regexp.Regexp{reM3U8, reMp4, reMpd} {
			for _, urlMatch := range re.FindAllStringSubmatch(scriptContent, -1) {
				if len(urlMatch) > 1 {
					result.VideoURLs = append(result.VideoURLs, urlMatch[1])
					foundVideo = true
				}
			}
		}

		// Если не нашли прямой URL но скрипт содержит ключевые слова — сохраняем первые 500 символов
		if !foundVideo && containsAny(scriptContent, videoKeywords) {
			snippet := strings.TrimSpace(scriptContent)
			if len(snippet) > 500 {
				snippet = snippet[:500] + "..."
			}
			result.Scripts = append(result.Scripts, snippet)
		}
	}

	// Дедупликация
	result.Iframes = unique(result.Iframes)
	result.VideoURLs = unique(result.VideoURLs)

	return result, nil
}

func containsAny(s string, keywords []string) bool {
	lower := strings.ToLower(s)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func unique(slice []string) []string {
	seen := make(map[string]struct{}, len(slice))
	result := slice[:0]
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}
