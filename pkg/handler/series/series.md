# series.go

## Что делает
HTTP-хендлеры для работы с сериями: поиск, получение по ID, похожие/связанные серии, резолв видео-плеера и прокси-слой для theatre.stloadi.live.

## Маршруты
- `GET /api/series/search?q=` — поиск
- `GET /api/series/{id}` — детали
- `GET /api/series/{id}/similar` — похожие
- `GET /api/series/{id}/relations` — связанные
- `GET /api/series/{id}/resolve` — HTML-страница плеера (пропатченная через patchHTML)
- `GET /api/hls-proxy?url=` — проксирование HLS-запросов через сервер
- `ANY /api/theatre-proxy?url=` — проксирование API-вызовов плеера к theatre.stloadi.live
- `GET /api/theatre-static/{path}` — проксирование статики (JS, CSS, изображения, шрифты) с theatre

## Изменения

### ResolveV2 (добавлен)
Отдаёт HTML-страницу с fullscreen iframe на `theatre.stloadi.live`. Браузер загружает theatre напрямую — JS работает в контексте theatre origin, `/bnsi/` идёт same-origin, всё работает без проксирования.

**Важно:** `referrerpolicy` на iframe НЕ должен быть `no-referrer` — theatre проверяет наличие Referer заголовка и отдаёт 404 без него.

**TODO:** URL с токенами захардкожен — нужно генерировать динамически для каждого kinopoisk ID.

### chromeClient (добавлен)
HTTP-клиент с TLS fingerprint Chrome через utls. Нужен для TheatreProxy — theatre.stloadi.live блокирует Go default TLS (JA3 fingerprint). ALPN ограничен http/1.1, т.к. Go http.Transport не поддерживает h2 через custom DialTLS.

### TheatreStatic
Проксирует статические ресурсы (img, css, js, fonts) с `theatre.stloadi.live`. Поддерживает GET и POST (для `/bnsi/`). Пробрасывает оригинальный HTTP метод и тело запроса.

Поддерживает два режима:
- `/api/theatre-static/{path}` — основной (через base href)
- `/images/{path}`, `/build/{path}`, `/bnsi/{path}` — catch-all для абсолютных путей из JS плеера

### TheatreProxy
Проксирует API-вызовы плеера к `theatre.stloadi.live` через chromeClient (Chrome TLS fingerprint). Пробрасывает все клиентские заголовки, кроме hop-by-hop.

### HLSProxy
Проксирует HLS-запросы (master.m3u8, variant плейлисты, .ts сегменты) через сервер.
**Зачем:** CDN-токены привязаны к IP сервера. Для .m3u8 переписывает relative URL на абсолютные.

### Мёртвый код (можно удалить)
- `injectProxyInterceptor()` — был частью старого V2 подхода с fetch/XHR перехватом, заменён на iframe
- `sessionEntry`, `proxySessions`, `genSessionID()`, `getSessionCookies()` — управление cookie-сессиями для старого proxy подхода, сейчас используется только в TheatreProxy (session query param)

## Роутинг в main.go
`/images/`, `/build/`, `/js/`, `/bnsi/` (GET+POST) маршрутизируются в apiHandler (а не в Next.js прокси) для поддержки catch-all абсолютных путей из JS плеера.
