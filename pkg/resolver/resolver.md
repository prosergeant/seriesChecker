# resolver.go

## Что делает
Через headless Chromium (chromedp) перехватывает M3U8 URL HLS-потока и возвращает HTML-страницу оригинального плеера theatre.stloadi.live, пропатченную для работы через прокси.

Маршрут: `GET /api/series/{id}/resolve`

## Архитектурный подход (текущий)

1. Открывает новую вкладку в общем браузерном процессе
2. Устанавливает заголовки (`Referer`, `Sec-Fetch-*`) и инжектирует init-скрипт
3. Навигирует к URL плеера
4. Ждёт события `network.EventResponseReceived` с URL содержащим `.m3u8` (максимум 25 секунд)
5. Захватывает HTML плеера (уже инициализированного) через `chromedp.OuterHTML`
6. Патчит HTML через `patchHTML()` для работы в контексте localhost
7. Fallback: если HTML не удалось захватить, возвращает простой HLS.js плеер с m3u8 URL

## patchHTML — проксирование ресурсов плеера

HTML плеера theatre.stloadi.live патчится в несколько этапов:

1. **`<base href="/api/theatre-static/">`** — relative URL идут через наш прокси статики
2. **Удаление SRI-хешей** и blob src (стейл из chromedp-сессии)
3. **Замена абсолютных URL** `https://theatre.stloadi.live/` → `/api/theatre-static/`
4. **Замена абсолютных путей** `"/build/`, `"/images/`, `"/fonts/` → проксированные
5. **Замена CSS `url()`** без кавычек
6. **Inject proxy script** — перехватывает fetch/XHR:
   - `theatre.stloadi.live/*` → `/api/theatre-proxy` (API-вызовы плеера, обход CORS)
   - `*.m3u8 / *.ts` → `/api/hls-proxy` (CDN требует Origin сервера)
   - `isFramed` lock (плеер не показывает ошибку "не в iframe")

## Известные ограничения

- **Видео не воспроизводится**: плеер при реинициализации читает `window.location` для получения токенов (`token_movie`, `token`), но URL теперь `localhost:8080/api/series/.../resolve`, а не оригинальный URL с параметрами. Нужно инжектировать токены в JS-контекст.
- **blob URL**: `URL.createObjectURL()` создаёт blob: с origin theatre, а документ на localhost — неизбежно при проксировании
- **TypeError в setCaptionsMenu**: внутренний баг минифицированного JS плеера

## Обход защиты theatre.stloadi.live

**Слой 1 — серверный (Referer):** требует наличия `Referer` заголовка.
Решение: `network.SetExtraHTTPHeaders` с `Referer: https://theatre.stloadi.live` + `Sec-Fetch-*`.

**Слой 2 — клиентский JS (isFramed):**
```javascript
var isFramed = false;
try { isFramed = window != window.top || ... } catch(e) { isFramed = true; }
if (!isFramed) { /* заменяет всю страницу на ошибку */ }
```
Решение: `page.AddScriptToEvaluateOnNewDocument` + proxyScript inject.

## Проблема IP-привязанных CDN-токенов

M3U8 URL содержит токен, привязанный к IP-адресу запрашивающего. Решение: HLS.js кастомный загрузчик (`ProxyLoader`) маршрутизирует все запросы через `/api/hls-proxy?url=...`.

## Архитектура
- Один shared `ExecAllocator` на весь сервер (один процесс Chrome)
- На каждый запрос — новый browser context (новая вкладка), закрывается через defer
- `Resolver.Close()` — освобождает Chrome-процесс, вызывать при shutdown сервера

## Docker
В Alpine финальном образе требуется `chromium`:
```
RUN apk add --no-cache ... chromium
ENV CHROMIUM_PATH=/usr/bin/chromium-browser
```
