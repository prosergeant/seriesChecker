package resolver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
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

// ResolveStream загружает страницу theatre через chromedp и возвращает m3u8 URL.
// Используется для получения потока — chromedp нужен потому что /bnsi/ API
// привязан к серверной сессии, которая создаётся при загрузке страницы в браузере.
//
// Стратегия: инжектируем в chromedp-браузер скрипт, который перехватывает fetch(/bnsi/)
// и сохраняет m3u8 URL из JSON-ответа ДО того как chromedp обращается к CDN.
// Это важно: CDN-токен привязан к IP/сессии первого запроса — если chromedp его «сжигает»,
// наш Go-прокси получит 403. Если URL извлечён из /bnsi/ до CDN-запроса,
// токен остаётся свежим.
// ResolveStream загружает страницу theatre через chromedp и возвращает m3u8 URL.
// После нахождения URL browserCtx сохраняется в r.streamCtx — он остаётся жив,
// чтобы FetchCDNURL мог делать fetch() в контексте Chrome-сессии, у которой
// есть валидный CDN-токен (токены привязаны к TLS-сессии Chrome, Go-клиент 403ит).
func (r *Resolver) ResolveStream(ctx context.Context, theatreURL string) (string, error) {
	// Отменяем предыдущую сессию, если есть
	r.streamMu.Lock()
	if r.streamCancel != nil {
		r.streamCancel()
	}
	r.streamCtx = nil
	r.streamCancel = nil
	r.streamMu.Unlock()

	browserCtx, cancel := chromedp.NewContext(r.allocCtx)

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancelTimeout()

	var m3u8URL string
	// var responseAll any
	var once sync.Once
	found := make(chan struct{})

	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		if e, ok := ev.(*network.EventResponseReceived); ok {
			if strings.Contains(e.Response.URL, ".m3u8") {
				// Логируем статус CDN-ответа: если 403 — IP забанен/rate-limit
				log.Printf("ResolveStream: CDN m3u8 response status=%d url=%s", e.Response.Status, e.Response.URL[:min(60, len(e.Response.URL))])
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
		cancel()
		return "", fmt.Errorf("navigate: %w", err)
	}

	select {
	case <-found:
		// Сохраняем browserCtx живым для FetchCDNURL
		r.streamMu.Lock()
		r.streamCtx = browserCtx
		r.streamCancel = cancel
		r.streamMu.Unlock()
		return m3u8URL, nil
	case <-time.After(25 * time.Second):
		cancel()
		return "", fmt.Errorf("m3u8 not found in 25s")
	}
}

type StreamResult struct {
	M3U8URL      string   `json:"m3u8_url"`
	M3U8All      []string `json:"m3u8_all"`
	HTML         string   `json:"html"`
	FinalURL     string   `json:"final_url"`
	BNSIResponse string   `json:"bnsi_response,omitempty"`
}

// ResolveStreamFull загружает страницу theatre и возвращает данные от /bnsi/movies/{id}
func (r *Resolver) ResolveStreamFull(ctx context.Context, theatreURL string) (*StreamResult, error) {
	// Don't cancel previous streamCtx - it might still be used by HLSProxy
	r.streamMu.Lock()
	r.streamCtx = nil
	r.streamCancel = nil
	r.streamMu.Unlock()

	browserCtx, cancelBrowser := chromedp.NewContext(r.allocCtx)

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 30*time.Second)
	// Don't defer cancelTimeout - we want browserCtx to stay alive after function returns

	var bnsiResponse string
	var m3u8URLs []string
	var found = make(chan struct{})
	var reqID network.RequestID

	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		if e, ok := ev.(*network.EventResponseReceived); ok {
			url := e.Response.URL
			if strings.Contains(url, "/bnsi/movies") && e.Response.Status == 200 {
				log.Printf("ResolveStreamFull: bnsi response status=%d url=%s", e.Response.Status, url[:min(80, len(url))])
				reqID = e.RequestID
			}
			if strings.Contains(url, ".m3u8") {
				log.Printf("ResolveStreamFull: CDN m3u8 response status=%d url=%s", e.Response.Status, url[:min(60, len(url))])
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
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			pageLoaded = true
			return nil
		}),
	); err != nil {
		cancelBrowser()
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if !pageLoaded {
		cancelBrowser()
		return nil, fmt.Errorf("page failed to load")
	}

	_ = chromedp.Run(timeoutCtx,
		chromedp.Sleep(500*time.Millisecond),
	)

	select {
	case <-found:
		cancelTimeout()
	case <-time.After(20 * time.Second):
		cancelTimeout()
	}

	r.streamMu.Lock()
	r.streamCtx = browserCtx
	r.streamCancel = cancelBrowser
	r.streamMu.Unlock()

	m3u8URL := ""
	if len(m3u8URLs) > 0 {
		m3u8URL = m3u8URLs[0]
	}

	var rawHTML string
	_ = chromedp.Run(timeoutCtx,
		chromedp.OuterHTML("html", &rawHTML),
	)

	// Try to get the bnsi data directly from theatre - extract movie ID from various sources
	var pageData string
	_ = chromedp.Run(timeoutCtx,
		chromedp.Evaluate(`(function(){try{var html=document.body.innerHTML;var m=html.match(/\"movie[iI]d\"[\\s:]*(\\d+)/);if(m)return'm='+m[1];m=html.match(/kinopoisk[_-]?id[\\s:]*(\\d+)/i);if(m)return'm='+m[1];m=html.match(/\\/movies\\/(\\d+)/);if(m)return'm='+m[1];return'nofound'}catch(e){return'err:'+e}})()`, &pageData),
	)
	log.Printf("ResolveStreamFull: page data: %s", pageData)

	// Try to get bnsi response from within browser context
	var bnsiFromBrowser string
	_ = chromedp.Run(timeoutCtx,
		chromedp.Evaluate(`(function(){try{var html=document.body.innerHTML;var m=html.match(/\\/movies\\/(\\d+)/);var id=m?m[1]:'176192';var f=new FormData();f.append('token','45e20a5f584becf7a64dffb7174ddf');var x=new XMLHttpRequest();x.open('POST','/bnsi/movies/'+id,false);x.send(f);return x.responseText.substring(0,500)}catch(e){return'err:'+e}})()`, &bnsiFromBrowser),
	)
	if bnsiFromBrowser != "" && !strings.HasPrefix(bnsiFromBrowser, "err:") {
		bnsiResponse = bnsiFromBrowser
	}

	// Try to get bnsi response body using request ID
	if bnsiResponse == "" && reqID != "" {
		var body []byte
		var bodyErr error
		_ = chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			body, bodyErr = network.GetResponseBody(reqID).Do(ctx)
			return nil
		}))
		if bodyErr == nil && len(body) > 0 {
			bnsiResponse = string(body)
			log.Printf("ResolveStreamFull: got bnsi body: %d bytes", len(body))
		} else if bodyErr != nil {
			log.Printf("ResolveStreamFull: get body error: %v", bodyErr)
		}
	}

	if bnsiResponse == "" {
		bnsiResponse = "{}"
	}

	return &StreamResult{
		M3U8URL:      m3u8URL,
		M3U8All:      m3u8URLs,
		HTML:         rawHTML,
		FinalURL:     theatreURL,
		BNSIResponse: bnsiResponse,
	}, nil
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

// BNSIRequest делает POST запрос к /bnsi/movies/{id} через chromedp браузер.
// theatreURLWithParams - полный URL theatre.stloadi.live с token_movie, token, translation.
func (r *Resolver) BNSIRequest(ctx context.Context, theatreURLWithParams, theatrePath, token string) (*BNSIResult, error) {
	browserCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 25*time.Second)
	defer cancelTimeout()

	var responseBody string
	var statusCode int

	log.Printf("BNSIRequest: navigating to %s", theatreURLWithParams)

	if err := chromedp.Run(timeoutCtx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			"Sec-Fetch-Dest": "empty",
			"Sec-Fetch-Mode": "cors",
			"Sec-Fetch-Site": "same-origin",
			"Origin":         theatreBase,
			"Referer":        theatreURLWithParams,
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(
				`Object.defineProperty(window,'isFramed',{value:true,writable:false,configurable:false});`,
			).Do(ctx)
			return err
		}),
		chromedp.Evaluate(`Object.defineProperty(navigator,'webdriver',{get:()=>undefined})`, nil),
		chromedp.Navigate(theatreURLWithParams),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`
			(function(){
				try{
					var html=document.body.innerHTML;
					var m=html.match(new RegExp("/movies/(\\d+)"));
					var id=m?m[1]:'176192';
					var pageInfo='id='+id;
					var f=new FormData();
					f.append('token','45e20a5f584becf7a64dffb7174ddf');
					var x=new XMLHttpRequest();
					x.withCredentials=true;
					x.open('POST','/bnsi/movies/'+id,false);
					x.onreadystatechange=function(){
						pageInfo+='|rs='+x.readyState+'|st='+x.status;
					};
					x.send(f);
					var resp = x.responseText;
					if(!resp) resp='empty';
					return pageInfo+'|'+resp;
				}catch(e){return 'err:'+e}
			})()
		`, &responseBody),
	); err != nil {
		return nil, fmt.Errorf("chromedp run: %w", err)
	}

	if strings.HasPrefix(responseBody, "err:") {
		return nil, fmt.Errorf("js error: %s", responseBody)
	}

	parts := strings.SplitN(responseBody, "|", 2)
	if len(parts) == 2 {
		statusCode, _ = strconv.Atoi(parts[0])
		responseBody = parts[1]
	} else {
		statusCode = 200
	}

	log.Printf("BNSIRequest: path=%s status=%d body_len=%d", theatrePath, statusCode, len(responseBody))
	return &BNSIResult{
		Body:       responseBody,
		StatusCode: statusCode,
	}, nil
}

// FetchCDNURL загружает CDN-ресурс через fetch() внутри живого Chrome-браузера,
// у которого есть валидный TLS-токен для CDN. Возвращает тело и Content-Type.
// Используется HLSProxy когда Go-клиент получает 403 от CDN.
func (r *Resolver) FetchCDNURL(url string) ([]byte, string, error) {
	r.streamMu.Lock()
	sessCtx := r.streamCtx
	r.streamMu.Unlock()

	if sessCtx == nil {
		return nil, "", fmt.Errorf("no active stream session")
	}

	// JS fetch внутри Chrome: возвращает "content-type\nbase64data"
	script := fmt.Sprintf(`(async function(){
  const r = await fetch(%q);
  const ct = r.headers.get('content-type')||'';
  const buf = await r.arrayBuffer();
  const arr = new Uint8Array(buf);
  let b64='';
  const C=8192;
  for(let i=0;i<arr.length;i+=C)
    b64+=btoa(String.fromCharCode.apply(null,arr.subarray(i,Math.min(i+C,arr.length))));
  return ct+'\n'+b64;
})()`, url)

	fetchCtx, cancel := context.WithTimeout(sessCtx, 30*time.Second)
	defer cancel()

	var result *cdpruntime.RemoteObject
	var exception *cdpruntime.ExceptionDetails
	if err := chromedp.Run(fetchCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		result, exception, err = cdpruntime.Evaluate(script).
			WithAwaitPromise(true).
			WithReturnByValue(true).
			Do(ctx)
		if exception != nil {
			return fmt.Errorf("js: %s", exception.Text)
		}
		return err
	})); err != nil {
		return nil, "", fmt.Errorf("evaluate: %w", err)
	}

	var combined string
	if err := json.Unmarshal(result.Value, &combined); err != nil {
		return nil, "", fmt.Errorf("unmarshal: %w", err)
	}
	idx := strings.IndexByte(combined, '\n')
	if idx < 0 {
		return nil, "", fmt.Errorf("unexpected result format")
	}
	ct := combined[:idx]
	data, err := base64.StdEncoding.DecodeString(combined[idx+1:])
	if err != nil {
		return nil, "", fmt.Errorf("base64: %w", err)
	}
	return data, ct, nil
}

// FetchTheatrePageWithCookies загружает страницу театра через chromedp (реальный браузер),
// ждёт выполнения JS (установки session cookies), возвращает HTML и cookies.
// Cookies нужны для последующих /bnsi/ запросов — театр валидирует сессию.
func (r *Resolver) FetchTheatrePageWithCookies(ctx context.Context, theatreURL string) (html string, cookies []*network.Cookie, err error) {
	browserCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 20*time.Second)
	defer cancelTimeout()

	var rawHTML string
	if err = chromedp.Run(timeoutCtx,
		network.Enable(),
		chromedp.Navigate(theatreURL),
		chromedp.Sleep(2*time.Second), // ждём JS для установки cookies
		chromedp.OuterHTML("html", &rawHTML),
	); err != nil {
		return "", nil, fmt.Errorf("chromedp navigate: %w", err)
	}

	// Извлекаем все cookies установленные театром (через Set-Cookie и через JS)
	if err2 := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var e error
		cookies, e = network.GetCookies().Do(ctx)
		return e
	})); err2 != nil {
		log.Printf("FetchTheatrePageWithCookies: get cookies error: %v", err2)
		// не фатально — продолжаем без cookies
	}

	log.Printf("FetchTheatrePageWithCookies: got %d cookies for %s", len(cookies), theatreURL[:min(80, len(theatreURL))])
	return rawHTML, cookies, nil
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
