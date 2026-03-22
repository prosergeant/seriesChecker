package resolver

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

// Result содержит всё что удалось извлечь из страницы с видео
type Result struct {
	FinalURL  string   `json:"final_url"`
	Iframes   []string `json:"iframes"`
	VideoURLs []string `json:"video_urls"`
}

type Resolver struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
}

func New() *Resolver {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		// Скрываем признаки автоматизации
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-infobars", true),
		chromedp.WindowSize(1920, 1080),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	// В Docker (Alpine) Chromium ставится по этому пути
	if execPath := os.Getenv("CHROMIUM_PATH"); execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	return &Resolver{
		allocCtx:    allocCtx,
		allocCancel: cancel,
	}
}

// Close освобождает браузерный процесс. Вызывать при shutdown сервера.
func (r *Resolver) Close() {
	r.allocCancel()
}

func (r *Resolver) Resolve(ctx context.Context, kinopoiskID int, isSerial bool) (*Result, error) {
	mediaType := "film"
	if isSerial {
		mediaType = "series"
	}

	startURL := fmt.Sprintf("https://www.sspoisk.ru/%s/%d", mediaType, kinopoiskID)

	// Новая вкладка в общем браузере
	browserCtx, cancelBrowser := chromedp.NewContext(r.allocCtx)
	defer cancelBrowser()

	// 30 секунд на всё
	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancelTimeout()

	// Убираем navigator.webdriver до навигации
	if err := chromedp.Run(timeoutCtx,
		chromedp.Evaluate(`Object.defineProperty(navigator,'webdriver',{get:()=>undefined})`, nil),
		chromedp.Navigate(startURL),
	); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	// Ждём до 20 секунд пока Kinobox установит src у своего iframe
	waitCtx, waitCancel := context.WithTimeout(timeoutCtx, 20*time.Second)
	_ = chromedp.Run(waitCtx, chromedp.Poll(
		`document.querySelector('iframe.kinobox_iframe[src]') !== null`,
		nil,
		chromedp.WithPollingInterval(500*time.Millisecond),
	))
	waitCancel()

	// Вытаскиваем финальный URL и все kinobox iframe src
	var finalURL string
	var iframeSrcs []string
	_ = chromedp.Run(timeoutCtx,
		chromedp.Location(&finalURL),
		chromedp.Evaluate(
			`Array.from(document.querySelectorAll('iframe.kinobox_iframe[src]')).map(f => f.src).filter(Boolean)`,
			&iframeSrcs,
		),
	)
	log.Printf("resolver: found %d iframes, finalURL=%s", len(iframeSrcs), finalURL)

	return &Result{
		FinalURL: finalURL,
		Iframes:  unique(iframeSrcs),
	}, nil
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
