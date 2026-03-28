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
	startURL := "https://theatre.stloadi.live/?token_movie=4f942c2ef97f5097b4690e45316e0a&translation=34&token=45e20a5f584becf7a64dffb7174ddf"

	// Новая вкладка в общем браузере
	browserCtx, cancelBrowser := chromedp.NewContext(r.allocCtx)
	defer cancelBrowser()

	// 30 секунд на всё
	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancelTimeout()

	// Перехватываем сетевые запросы: ищем M3U8 манифест который плеер загружает для HLS.
	// Когда плеер получит токен у API и запросит поток — URL попадёт в listener.
	var m3u8URL string
	var m3u8Once sync.Once
	m3u8Found := make(chan struct{})

	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		if e, ok := ev.(*network.EventResponseReceived); ok {
			u := e.Response.URL
			if strings.Contains(u, ".m3u8") {
				m3u8Once.Do(func() {
					m3u8URL = u
					close(m3u8Found)
				})
			}
		}
	})

	// Инициализируем браузер: заголовки + блокировка isFramed + навигация
	if err := chromedp.Run(timeoutCtx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			"Sec-Fetch-Dest": "iframe",
			"Sec-Fetch-Mode": "navigate",
			"Sec-Fetch-Site": "cross-site",
			"Referer":        "https://theatre.stloadi.live",
		}),
		// Блокируем переменную isFramed: страница не сможет присвоить false (non-writable),
		// поэтому if(!isFramed) никогда не выполнится и плеер не заменится ошибкой
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(
				`Object.defineProperty(window,'isFramed',{value:true,writable:false,configurable:false});`,
			).Do(ctx)
			return err
		}),
		chromedp.Evaluate(`Object.defineProperty(navigator,'webdriver',{get:()=>undefined})`, nil),
		chromedp.Navigate(startURL),
	); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	// Ждём M3U8 URL (максимум 25 секунд)
	select {
	case <-m3u8Found:
		log.Printf("resolver: found m3u8=%s", m3u8URL)
	case <-time.After(25 * time.Second):
		log.Printf("resolver: m3u8 not found in 25s, falling back")
	}

	if m3u8URL != "" {
		// Отдаём минимальный HLS-плеер — никаких cross-origin проблем,
		// никаких CORS-блокировок плеерского API, никаких SVG-спрайт issues.
		html := buildHLSPage(m3u8URL)
		return &Result{FinalURL: m3u8URL, HTML: html}, nil
	}

	// Fallback: вернуть финальный URL страницы для диагностики
	var finalURL string
	_ = chromedp.Run(timeoutCtx, chromedp.Location(&finalURL))
	return nil, fmt.Errorf("m3u8 not found, page at: %s", finalURL)
}

// buildHLSPage возвращает HTML-страницу с HLS.js плеером.
// Все HLS-запросы (m3u8, ts-сегменты) маршрутизируются через /api/hls-proxy,
// чтобы CDN-токены (привязаны к IP сервера) оставались валидными в браузере пользователя.
func buildHLSPage(m3u8URL string) string {
	return `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#000;width:100%;height:100vh;display:flex;align-items:center;justify-content:center}
video{width:100%;height:100%;object-fit:contain}
</style>
</head>
<body>
<video id="v" controls playsinline></video>
<script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
<script>
var v = document.getElementById('v');
var src = ` + fmt.Sprintf("%q", m3u8URL) + `;
var PROXY = '/api/hls-proxy?url=';
function ProxyLoader(config) {
  this._loader = new (Hls.DefaultConfig.loader)(config);
}
Object.defineProperty(ProxyLoader.prototype, 'stats', {
  get: function() { return this._loader.stats; }
});
Object.defineProperty(ProxyLoader.prototype, 'context', {
  get: function() { return this._loader.context; }
});
ProxyLoader.prototype.destroy = function() { this._loader.destroy(); };
ProxyLoader.prototype.abort  = function() { this._loader.abort(); };
ProxyLoader.prototype.load   = function(ctx, cfg, cbs) {
  ctx.url = PROXY + encodeURIComponent(ctx.url);
  this._loader.load(ctx, cfg, cbs);
};
if (Hls.isSupported()) {
  var hls = new Hls({loader: ProxyLoader, maxBufferLength:30, maxMaxBufferLength:60});
  hls.loadSource(src);
  hls.attachMedia(v);
  hls.on(Hls.Events.MANIFEST_PARSED, function() { v.play().catch(function(){}); });
} else if (v.canPlayType('application/vnd.apple.mpegurl')) {
  v.src = src;
  v.addEventListener('loadedmetadata', function() { v.play().catch(function(){}); });
}
</script>
</body>
</html>`
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
