# resolver.go

## Что делает
Находит iframe-ссылки плеера для фильма/сериала через headless Chromium.

Маршрут: `GET /api/series/{id}/resolve`

## Логика
1. Формирует URL `https://www.sspoisk.ru/{series|film}/{kinopoisk_id}`
2. Открывает страницу в headless Chromium (через `chromedp`)
3. Убирает `navigator.webdriver` до навигации (анти-бот детекция)
4. Ждёт до 20 секунд появления `iframe.kinobox_iframe[src]` через JS-poll каждые 500мс
5. Извлекает src всех `iframe.kinobox_iframe[src]` через `querySelectorAll`
6. Возвращает `{ final_url, iframes, video_urls }`

## Почему chromedp (было переписано с HTTP-клиента)
Целевой сайт использует Kinobox — JS-агрегатор плееров. При обычном HTTP-запросе HTML содержит пустой `<div class="kinobox">`, iframe загружается через JS.

### Ключевые решения при отладке
- `WaitVisible('iframe')` не работало — срабатывало на рекламный iframe раньше Kinobox
- `querySelectorAll('iframe').src` давало пустые строки — src устанавливается JS-ом асинхронно
- Решение: `Poll('iframe.kinobox_iframe[src] !== null')` + `querySelectorAll('iframe.kinobox_iframe[src]')`
- `disable-blink-features=AutomationControlled` + удаление `navigator.webdriver` — нужно чтобы Kinobox не детектировал headless

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
