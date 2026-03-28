# resolver.go

## Что делает
Через headless Chromium (chromedp) перехватывает M3U8 URL HLS-потока и возвращает HTML-страницу с HLS.js плеером, который воспроизводит поток через прокси-эндпоинт сервера.

Маршрут: `GET /api/series/{id}/resolve`

## Архитектурный подход (текущий)

1. Открывает новую вкладку в общем браузерном процессе
2. Устанавливает заголовки (`Referer`, `Sec-Fetch-*`) и инжектирует init-скрипт
3. Навигирует к URL плеера
4. Ждёт события `network.EventResponseReceived` с URL содержащим `.m3u8` (максимум 25 секунд)
5. Возвращает простую HTML-страницу с HLS.js и перехваченным M3U8 URL

## Почему именно перехват M3U8 (а не возврат HTML плеера)

Предыдущий подход (возврат HTML страницы плеера) не работал из-за двух проблем:
- **Cross-origin SVG `<use>`**: Chrome блокирует `<use href="https://theatre.stloadi.live/images/allplay.svg#...">` из iframe на localhost → 50+ ошибок в консоли
- **CORS на API плеера**: плеер делал запрос к CDN API для получения `id_file`, CORS блокировал → `TypeError: Cannot read properties of undefined (reading 'length')`

Решение: перехватить M3U8 URL в chromedp до возникновения CORS-проблем.

## Обход защиты theatre.stloadi.live

**Слой 1 — серверный (Referer):** требует наличия `Referer` заголовка.
Решение: `network.SetExtraHTTPHeaders` с `Referer: https://theatre.stloadi.live` + `Sec-Fetch-*`.

**Слой 2 — клиентский JS (isFramed):**
```javascript
var isFramed = false;
try { isFramed = window != window.top || ... } catch(e) { isFramed = true; }
if (!isFramed) { /* заменяет всю страницу на ошибку */ }
```
Решение: `page.AddScriptToEvaluateOnNewDocument` запускается ДО скриптов страницы:
```javascript
Object.defineProperty(window, 'isFramed', {value: true, writable: false, configurable: false})
```

## Проблема IP-привязанных CDN-токенов

M3U8 URL содержит токен, привязанный к IP-адресу запрашивающего. Chromedp делает запрос с IP сервера — токен валиден для сервера. Если отдать URL напрямую браузеру пользователя (другой IP) — CDN вернёт ошибку.

Решение: HLS.js кастомный загрузчик (`ProxyLoader`) маршрутизирует все запросы (m3u8 + ts-сегменты) через `/api/hls-proxy?url=...` — всё идёт через сервер с нужным IP.

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
