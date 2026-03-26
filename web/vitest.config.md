# vitest.config.ts

Конфигурация Vitest для unit-тестов фронтенда.

## Что делает
- `environment: 'jsdom'` — DOM-среда для тестов React-компонентов
- `globals: true` — `describe`/`it`/`expect` доступны без импорта
- `setupFiles: ['./vitest.setup.ts']` — подключает `@testing-library/jest-dom` матчеры
- `exclude: ['tests/**']` — исключает Playwright `.spec.ts` файлы, чтобы Vitest их не подхватывал
- `resolve.alias` — маппит `@/` → корень `web/`, как в `tsconfig.json`

## Что изменено и почему
Создан с нуля. До этого в проекте были только Playwright E2E тесты; Vitest добавлен для unit/integration тестирования компонентов и API-клиента без запуска браузера и backend-сервера.
