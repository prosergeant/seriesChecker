package resolver

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
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

// ResolveStream загружает страницу theatre через chromedp и возвращает m3u8 URL.
// Используется для получения потока — chromedp нужен потому что /bnsi/ API
// привязан к серверной сессии, которая создаётся при загрузке страницы в браузере.
func (r *Resolver) ResolveStream(ctx context.Context, theatreURL string) (string, error) {
	browserCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancelTimeout()

	var m3u8URL string
	var once sync.Once
	found := make(chan struct{})

	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		if e, ok := ev.(*network.EventResponseReceived); ok {
			if strings.Contains(e.Response.URL, ".m3u8") {
				once.Do(func() {
					m3u8URL = e.Response.URL
					close(found)
				})
			}
		}
	})

	if err := chromedp.Run(timeoutCtx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			"Sec-Fetch-Dest": "iframe",
			"Sec-Fetch-Mode": "navigate",
			"Sec-Fetch-Site": "cross-site",
			"Referer":        "https://theatre.stloadi.live",
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(
				`Object.defineProperty(window,'isFramed',{value:true,writable:false,configurable:false});`,
			).Do(ctx)
			return err
		}),
		chromedp.Evaluate(`Object.defineProperty(navigator,'webdriver',{get:()=>undefined})`, nil),
		chromedp.Navigate(theatreURL),
	); err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}

	select {
	case <-found:
		log.Printf("ResolveStream: found m3u8=%s", m3u8URL)
		return m3u8URL, nil
	case <-time.After(25 * time.Second):
		return "", fmt.Errorf("m3u8 not found in 25s")
	}
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
			if strings.Contains(u, "bnsi") {
				log.Printf("resolver: [bnsi] %s status=%d", u, e.Response.Status)
			}
			if strings.Contains(u, ".m3u8") {
				m3u8Once.Do(func() {
					m3u8URL = u
					close(m3u8Found)
				})
			}
		}
		if e, ok := ev.(*network.EventRequestWillBeSent); ok {
			u := e.Request.URL
			if strings.Contains(u, "bnsi") {
				log.Printf("resolver: [bnsi-req] %s method=%s hasPostData=%v headers=%v", u, e.Request.Method, e.Request.HasPostData, e.Request.Headers)
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

	if m3u8URL == "" {
		var finalURL string
		_ = chromedp.Run(timeoutCtx, chromedp.Location(&finalURL))
		return nil, fmt.Errorf("m3u8 not found, page at: %s", finalURL)
	}

	// Захватываем HTML страницы плеера — он уже полностью инициализирован
	// (мы дождались M3U8, значит JS отработал). Подождём чуть больше для рендера.
	var rawHTML string
	_ = chromedp.Run(timeoutCtx,
		chromedp.Sleep(300*time.Millisecond),
		chromedp.OuterHTML("html", &rawHTML),
	)

	if rawHTML != "" {
		html := patchHTML(rawHTML, startURL, m3u8URL)
		return &Result{FinalURL: m3u8URL, HTML: html}, nil
	}

	// Fallback: собственный HLS.js плеер
	log.Printf("resolver: html capture failed, using fallback player")
	return &Result{FinalURL: m3u8URL, HTML: buildHLSPage(m3u8URL)}, nil
}

var (
	reSRI       = regexp.MustCompile(` integrity="[^"]*"`)
	reCrossOrig = regexp.MustCompile(` crossorigin(?:="[^"]*")?`)
	reBlobSrc   = regexp.MustCompile(` src="blob:[^"]*"`)
)

// patchHTML подготавливает оригинальный HTML плеера для работы в браузере пользователя:
//  1. base href → все relative URLs плеера разрешаются на theatre.stloadi.live
//  2. Удаление SRI-хешей → скрипты загружаются без проверки целостности
//  3. Inject proxy script → перехватывает fetch/XHR и роутит через наши прокси:
//     - theatre.stloadi.live/* → /api/theatre-proxy  (API-вызовы плеера, без CORS)
//     - *.m3u8 / *.ts         → /api/hls-proxy       (CDN требует Origin сервера)
//     - isFramed lock         → плеер не заменяется экраном ошибки
func patchHTML(raw, startURL, m3u8URL string) string {
	// Извлекаем query string из оригинального URL для инжекции токенов
	queryString := ""
	if idx := strings.Index(startURL, "?"); idx >= 0 {
		queryString = startURL[idx:]
	}

	// 1. base href → все relative URL идут через наш прокси статики
	raw = strings.Replace(raw, "<head>", `<head><base href="/api/theatre-static/">`, 1)

	// 2. SRI + blob src (стейл из chromedp — blob URL из захваченной сессии невалидны)
	raw = reSRI.ReplaceAllString(raw, "")
	raw = reCrossOrig.ReplaceAllString(raw, "")
	raw = reBlobSrc.ReplaceAllString(raw, "")

	// 3. Заменяем абсолютные ссылки на theatre.stloadi.live на проксированные,
	// но НЕ трогаем blob: URL (blob:https://theatre.stloadi.live/uuid)
	raw = strings.ReplaceAll(raw, "blob:https://theatre.stloadi.live/", "blob:__THEATRE_BLOB__")
	raw = strings.ReplaceAll(raw, "https://theatre.stloadi.live/", "/api/theatre-static/")
	raw = strings.ReplaceAll(raw, "blob:__THEATRE_BLOB__", "blob:https://theatre.stloadi.live/")

	// 4. Заменяем абсолютные пути (/build/, /images/) на проксированные.
	// base href не влияет на пути, начинающиеся с "/", поэтому нужно явно.
	raw = strings.ReplaceAll(raw, `"/build/`, `"/api/theatre-static/build/`)
	raw = strings.ReplaceAll(raw, `'/build/`, `'/api/theatre-static/build/`)
	raw = strings.ReplaceAll(raw, `"/images/`, `"/api/theatre-static/images/`)
	raw = strings.ReplaceAll(raw, `'/images/`, `'/api/theatre-static/images/`)
	raw = strings.ReplaceAll(raw, `"/fonts/`, `"/api/theatre-static/fonts/`)
	raw = strings.ReplaceAll(raw, `'/fonts/`, `'/api/theatre-static/fonts/`)
	// CSS url() без кавычек: url(/images/...) → url(/api/theatre-static/images/...)
	raw = strings.ReplaceAll(raw, `url(/images/`, `url(/api/theatre-static/images/`)
	raw = strings.ReplaceAll(raw, `url(/build/`, `url(/api/theatre-static/build/`)
	raw = strings.ReplaceAll(raw, `url(/fonts/`, `url(/api/theatre-static/fonts/`)

	// 5. Inject proxy script сразу после base href (до любых других скриптов)
	raw = strings.Replace(raw,
		`<base href="/api/theatre-static/">`,
		`<base href="/api/theatre-static/">`+proxyScript(queryString),
		1,
	)

	// 6. Inject late-init скрипт перед </body> — принудительная инициализация HLS.js
	// с m3u8 URL, т.к. оригинальный плеер не может реинициализироваться
	// (DOM захвачен chromedp после первой инициализации)
	if m3u8URL != "" {
		raw = strings.Replace(raw, "</body>", hlsInitScript(m3u8URL)+"</body>", 1)
	}

	return raw
}

// hlsInitScript возвращает <script>, который дожидается загрузки страницы
// и принудительно инициализирует HLS.js с m3u8 URL через наш прокси.
// Оригинальный плеер theatre не может реинициализироваться (DOM уже был
// модифицирован в chromedp-сессии), поэтому мы берём на себя запуск видео.
func hlsInitScript(m3u8URL string) string {
	return `<script>
(function(){
  var PROXY='/api/hls-proxy?url=';
  var M3U8=` + fmt.Sprintf("%q", m3u8URL) + `;
  function initHLS(){
    var v=document.querySelector('video');
    if(!v) return;
    // Убираем loader overlay
    var loader=document.querySelector('.loader.active');
    if(loader) loader.classList.remove('active');
    if(typeof Hls!=='undefined'&&Hls.isSupported()){
      var hls=new Hls({
        xhrSetup:function(xhr,url){
          if(url.startsWith('https://'))
            xhr.open('GET',PROXY+encodeURIComponent(url),true);
        },
        maxBufferLength:30,
        maxMaxBufferLength:60
      });
      hls.loadSource(M3U8);
      hls.attachMedia(v);
      hls.on(Hls.Events.MANIFEST_PARSED,function(){
        v.play().catch(function(){});
      });
    } else if(v.canPlayType('application/vnd.apple.mpegurl')){
      v.src=PROXY+encodeURIComponent(M3U8);
      v.addEventListener('loadedmetadata',function(){v.play().catch(function(){});});
    }
  }
  if(document.readyState==='complete') setTimeout(initHLS,500);
  else window.addEventListener('load',function(){setTimeout(initHLS,500);});
})();
</script>`
}

// proxyScript возвращает <script> который:
//   - Блокирует isFramed (плеер не считает себя не в iframe)
//   - Перехватывает fetch и XHR: API-вызовы плеера к theatre.stloadi.live идут
//     через /api/theatre-proxy (обходит CORS); HLS-запросы идут через /api/hls-proxy
//     (сервер добавляет нужный Origin заголовок для CDN)
func proxyScript(queryString string) string {
	return `<script>
(function(){
  // Плеер проверяет isFramed — блокируем присваивание false
  Object.defineProperty(window,'isFramed',{value:true,writable:false,configurable:false});

  // Инжектируем оригинальные query-параметры (token_movie, token, translation)
  // в URL страницы, чтобы плеер мог их прочитать через window.location.search
  try { history.replaceState(null,'',window.location.pathname+` + fmt.Sprintf("%q", queryString) + `); } catch(e){}

  var T='/api/theatre-proxy?url=';
  var H='/api/hls-proxy?url=';
  var S='/api/theatre-static/';

  function rw(url){
    if(typeof url!=='string') return url;
    // API-вызовы плеера к своему серверу (fetch/XHR из JS могут формировать абсолютные URL)
    if(url.startsWith('https://theatre.stloadi.live/')) return T+encodeURIComponent(url);
    if(url.startsWith('https://theatre.stloadi.live')) return T+encodeURIComponent(url);
    // HLS-манифесты и сегменты (CDN требует Origin: https://theatre.stloadi.live)
    if(url.startsWith('https://')&&(url.indexOf('.m3u8')>=0||url.endsWith('.ts')||url.indexOf('.ts?')>=0))
      return H+encodeURIComponent(url);
    return url;
  }

  // Перехват fetch
  var _f=window.fetch;
  window.fetch=function(url,o){ return _f.call(window,rw(url),o); };

  // Перехват XHR
  var _o=XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open=function(){
    if(arguments.length>=2) arguments[1]=rw(arguments[1]);
    return _o.apply(this,arguments);
  };

  // Перехват URL.createObjectURL для blob — разрешаем blob URL
  // (браузер блокировал blob: с другого origin, теперь origin совпадает)
})();
</script>`
}

// buildHLSPage — fallback: простая HTML-страница с HLS.js плеером.
// Используется если не удалось захватить HTML оригинального плеера.
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
