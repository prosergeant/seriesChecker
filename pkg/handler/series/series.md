# series.go

## Что делает
HTTP-хендлеры для работы с сериями: поиск, получение по ID, похожие/связанные серии, резолв видео-плеера и HLS-прокси.

## Маршруты
- `GET /api/series/search?q=` — поиск
- `GET /api/series/{id}` — детали
- `GET /api/series/{id}/similar` — похожие
- `GET /api/series/{id}/relations` — связанные
- `GET /api/series/{id}/resolve` — HTML-страница с HLS.js плеером
- `GET /api/hls-proxy?url=` — проксирование HLS-запросов через сервер

## HLSProxy (добавлен)

Проксирует HLS-запросы (master.m3u8, variant плейлисты, .ts сегменты) от CDN через сервер.

**Зачем:** CDN-токены в M3U8 URL привязаны к IP-адресу. Chromedp получает URL с IP сервера — этот токен действителен только для запросов с серверного IP. Чтобы браузер пользователя мог воспроизводить поток, все HLS-запросы направляются через этот прокси.

**Безопасность:** принимает только HTTPS-URL (`strings.HasPrefix(rawURL, "https://")`). Возвращает `Access-Control-Allow-Origin: *` для работы в iframe.
