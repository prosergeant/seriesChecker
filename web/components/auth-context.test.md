# components/auth-context.test.tsx

Integration-тест для `AuthProvider` и `useAuth` хука.

## Что тестируется
- При монтировании вызывается `api.auth.me()` (проверка существующей сессии)
- `isAuthenticated = true` когда `me()` возвращает user
- `isAuthenticated = false` когда `me()` бросает ошибку (401)
- `login()` вызывает `api.auth.login()` и затем перепроверяет сессию через `me()`
- `logout()` вызывает `api.auth.logout()` и сбрасывает user в null

## Подход
- `vi.mock('@/lib/api')` — мокаем API-клиент
- `vi.mock('sonner')` — мокаем toast (чтобы не падало без DOM-провайдера)
- Вспомогательный `AuthConsumer` компонент рендерит состояние в DOM для assertions

## Что изменено и почему
Создан с нуля. Покрывает state-машину авторизации без реального HTTP.
