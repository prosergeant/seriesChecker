# series.go

## Что делает

HTTP хендлер для всех эндпоинтов связанных с сериалами/фильмами.

Маршруты:
- `GET /api/series/search?q=` — поиск через Kinopoisk API
- `GET /api/series/{id}` — получить сериал по Kinopoisk ID
- `GET /api/series/{id}/similar` — похожие
- `GET /api/series/{id}/relations` — связанные (сиквелы, приквелы)
- `GET /api/series/{id}/resolve` — найти источник видео (новый)

## Что было изменено

Добавлен метод `Resolve` и зависимость от пакета `internal/resolver`:

- `Handler` теперь содержит поле `resolver *resolver.Resolver`
- `NewHandler` инициализирует `resolver.New()` автоматически
- `Resolve` — новый хендлер: получает серию по ID, определяет тип (сериал/фильм),
  вызывает `resolver.Resolve` и возвращает JSON с `final_url`, `iframes`, `video_urls`, `scripts_with_video`
