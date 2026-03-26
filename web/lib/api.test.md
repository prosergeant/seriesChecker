# lib/api.test.ts

Unit-тесты для `api.ts` — типизированного API-клиента.

## Что тестируется
- `api.auth.login` — отправляет POST с нужным body и `credentials: 'include'`
- `api.auth.me` — GET запрос с include-credentials, возвращает user
- Ошибки: при `!ok` бросает ошибку с сообщением из JSON-тела (`{ error: '...' }`)
- `api.auth.logout` — POST на `/api/auth/logout`
- `api.progress.getAll()` — без `?status` параметра
- `api.progress.getAll('watching')` — добавляет `?status=watching`
- `api.progress.delete(42)` — DELETE на `/api/progress/42`
- `api.series.search('...')` — корректно encode-ит кириллицу в URL

## Подход
Мокаем `globalThis.fetch` через `vi.stubGlobal`. Реальных HTTP-запросов нет.

## Что изменено и почему
Создан с нуля. Покрывает критическую логику построения URL и параметров запроса без необходимости запускать backend.
