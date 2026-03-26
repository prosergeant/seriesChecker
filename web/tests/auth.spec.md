# tests/auth.spec.ts

Playwright E2E тесты для флоу авторизации.

## Что тестируется
- Страница `/login` содержит email/password инпуты и кнопку
- Регистрация нового пользователя → редирект на `/` → логаут → редирект на `/login`
- Логин с неверным паролем → остаёмся на `/login` (редиректа нет)
- Логин с правильными credentials → редирект на `/` (только с `TEST_EMAIL`/`TEST_PASSWORD`)

## Переменные окружения
- `TEST_EMAIL`, `TEST_PASSWORD` — для теста "login with correct credentials"; без них тест пропускается

## Что изменено и почему
Создан с нуля. Заменяет smoke-тест из `basic.spec.ts` более конкретными assertions по auth-флоу.
