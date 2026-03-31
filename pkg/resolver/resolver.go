package resolver

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const theatreBase = "https://theatre.stloadi.live"

// Result содержит всё что удалось извлечь из страницы с видео
type Result struct {
	FinalURL  string   `json:"final_url"`
	Iframes   []string `json:"iframes"`
	VideoURLs []string `json:"video_urls"`
	HTML      string   `json:"html"`
}

type Resolver struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc

	streamMu     sync.Mutex
	streamCtx    context.Context // живой chromedp-контекст с CDN-сессией (nil если нет)
	streamCancel context.CancelFunc

	fetchMu sync.Mutex // сериализует concurrent FetchCDNURL вызовы (chromedp не поддерживает параллельный Evaluate)
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

	// CDN_PROXY_URL — HTTP(S) прокси для CDN запросов (например webshare.io).
	// Если задан, Chrome будет делать CDN-запросы через него.
	// Используется для тестирования IP-бана: если через прокси CDN отдаёт 200 — IP сервера забанен.
	// if proxyURL := os.Getenv("CDN_PROXY_URL"); proxyURL != "" {
	// 	log.Printf("resolver: using CDN proxy: %s", proxyURL)
	// 	opts = append(opts, chromedp.Flag("proxy-server", proxyURL))
	// }

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

type StreamResult struct {
	M3U8URL      string   `json:"m3u8_url"`
	M3U8All      []string `json:"m3u8_all"`
	HTML         string   `json:"html"`
	FinalURL     string   `json:"final_url"`
	BNSIResponse string   `json:"bnsi_response,omitempty"`
}

// ResolveStreamFullWithToken работает как ResolveStreamFull, но принимает token параметр
// для корректного запроса к /bnsi/movies/{id}.
func (r *Resolver) ResolveStreamFullWithToken(ctx context.Context, theatreURL, token string) (*StreamResult, error) {
	r.streamMu.Lock()
	r.streamCtx = nil
	r.streamCancel = nil
	r.streamMu.Unlock()

	browserCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 30*time.Second)

	var bnsiResponse string
	var m3u8URLs []string
	var found = make(chan struct{})
	var reqID network.RequestID

	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		if e, ok := ev.(*network.EventResponseReceived); ok {
			url := e.Response.URL
			if strings.Contains(url, "/bnsi/movies") && e.Response.Status == 200 {
				log.Printf("ResolveStreamFullWithToken: bnsi response status=%d url=%s", e.Response.Status, url[:min(80, len(url))])
				reqID = e.RequestID
			}
			if strings.Contains(url, ".m3u8") {
				log.Printf("ResolveStreamFullWithToken: CDN m3u8 response status=%d url=%s", e.Response.Status, url[:min(60, len(url))])
				m3u8URLs = append(m3u8URLs, url)
				if len(m3u8URLs) == 1 {
					close(found)
				}
			}
		}
	})

	var pageLoaded bool
	if err := chromedp.Run(timeoutCtx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			// "Sec-Fetch-Dest": "iframe",
			// "Sec-Fetch-Mode": "navigate",
			// "Sec-Fetch-Site": "cross-site",
			"Referer": "https://theatre.stloadi.live",
			"Origin":  "https://theatre.stloadi.live",
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(
				`Object.defineProperty(window,'isFramed',{value:true,writable:false,configurable:false});`,
			).Do(ctx)
			return err
		}),
		chromedp.Evaluate(`Object.defineProperty(navigator,'webdriver',{get:()=>undefined})`, nil),
		chromedp.Navigate(theatreURL),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			pageLoaded = true
			return nil
		}),
	); err != nil {
		cancel()
		cancelTimeout()
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if !pageLoaded {
		cancel()
		cancelTimeout()
		return nil, fmt.Errorf("page failed to load")
	}

	select {
	case <-found:
	case <-time.After(20 * time.Second):
	}

	m3u8URL := ""
	if len(m3u8URLs) > 0 {
		m3u8URL = m3u8URLs[0]
	}

	// Create a new context for body fetch (the timeoutCtx might be nearly expired)
	bodyCtx, cancelBody := context.WithTimeout(browserCtx, 15*time.Second)

	// Try to get bnsi response via JavaScript XHR with correct token
	var bnsiFromBrowser string
	_ = chromedp.Run(bodyCtx,
		chromedp.Evaluate(fmt.Sprintf(`
			(function(){
				try{
					var html=document.body.innerHTML;
					var m=html.match(new RegExp("/movies/(\\d+)"));
					var id=m?m[1]:'176192';
					var f=new FormData();
					f.append('token','%s');
					var x=new XMLHttpRequest();
					x.open('POST','/bnsi/movies/'+id,false);
					x.send(f);
					return x.responseText;
				}catch(e){return 'err:'+e}
			})()
		`, token), &bnsiFromBrowser),
	)
	if bnsiFromBrowser != "" && !strings.HasPrefix(bnsiFromBrowser, "err:") && strings.Contains(bnsiFromBrowser, "hlsSource") {
		bnsiResponse = bnsiFromBrowser
		log.Printf("ResolveStreamFullWithToken: got bnsi from browser: %d bytes", len(bnsiResponse))
	}

	// Try to get bnsi response body using request ID
	if bnsiResponse == "" && reqID != "" {
		var body []byte
		var bodyErr error
		_ = chromedp.Run(bodyCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			body, bodyErr = network.GetResponseBody(reqID).Do(ctx)
			return nil
		}))
		if bodyErr == nil && len(body) > 0 {
			bnsiResponse = string(body)
			log.Printf("ResolveStreamFullWithToken: got bnsi body from network: %d bytes", len(body))
		} else if bodyErr != nil {
			log.Printf("ResolveStreamFullWithToken: get body error: %v", bodyErr)
		}
	}

	cancelBody()

	if bnsiResponse == "" {
		bnsiResponse = "{}"
	}

	r.streamMu.Lock()
	r.streamCtx = browserCtx
	r.streamCancel = cancel
	r.streamMu.Unlock()

	cancelTimeout()
	cancel()
	return &StreamResult{
		M3U8URL:      m3u8URL,
		M3U8All:      m3u8URLs,
		HTML:         "",
		FinalURL:     theatreURL,
		BNSIResponse: bnsiResponse,
	}, nil
}

type BNSIResult struct {
	Body       string
	StatusCode int
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
