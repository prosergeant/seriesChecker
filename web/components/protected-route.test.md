# components/protected-route.test.tsx

Unit-тест для `ProtectedRoute` компонента.

## Что тестируется
- `isLoading = true` → рендерит спиннер (SVG), children не показывает
- `isAuthenticated = false` + `isLoading = false` → вызывает `router.push('/login')`, children не рендерит
- `isAuthenticated = true` → рендерит children, `router.push` не вызывается

## Подход
- `vi.mock('next/navigation')` — мокаем `useRouter` с `mockPush`
- `vi.mock('@/components/auth-context')` — мокаем `useAuth` для контроля состояния

## Что изменено и почему
Создан с нуля. Компонент — охранник маршрутов; важно убедиться, что редирект происходит именно в правильных состояниях.
