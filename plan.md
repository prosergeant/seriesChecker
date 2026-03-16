# План разработки сервиса "SeriesTracker"

Проект предназначен для отслеживания прогресса просмотра сериалов с личными кабинетами и уведомлениями.

## 1. Проектирование (Архитектура)
- [ ] **База данных:** PostgreSQL (основные данные), Redis (кэширование сессий).
- [ ] **API:** RESTful или GraphQL (рекомендуется REST для простоты интеграции с React Actions).
- [ ] **Авторизация:** JWT (AccessToken + RefreshToken) или сессии на основе HTTP-only cookies.

## 2. Backend (Go 1.22+)
*Использование новейшего встроенного роутера и типизированного кода.*

- [ ] **Инициализация проекта:** `go mod init`, настройка структуры (Clean Architecture или Layered).
- [ ] **Routing:** Использование обновленного `net/http` (теперь поддерживает методы и паттерны в `ServeMux`).
- [ ] **Database:** Использование `pgx/v5` для работы с Postgres и `sqlc` для генерации типобезопасного кода из SQL-запросов.
- [ ] **Middleware:** Реализация логирования через встроенный `slog`, Recovery и CORS.
- [ ] **Бизнес-логика:**
    - Эндпоинты для поиска сериалов (интеграция с TMDB API).
    - CRUD для пользовательских списков ("Смотрю", "Запланировано", "Брошено").
    - Обновление прогресса (отметка конкретной серии).
- [ ] **Concurrency:** Использование `errgroup` для параллельных запросов к внешним API.

## 3. Frontend (React 19 + Next.js 15/Vite)
*Фокус на минимизации клиентского JS и улучшении UX.*

- [ ] **UI Kit:** Tailwind CSS + Shadcn/UI.
- [ ] **React 19 Features:**
    - **Server Components (RSC):** Рендеринг страниц сериалов на сервере для SEO и скорости.
    - **Actions:** Использование `useActionState` для обработки форм добавления серий без ручного управления `loading/error`.
    - **Optimistic Updates:** Использование `useOptimistic` для мгновенного переключения "галочки" просмотра серии.
    - **Ref handling:** Использование упрощенного `ref` как обычного пропса.
- [ ] **State Management:** TanStack Query (React Query) для синхронизации состояния с бэкендом.
- [ ] **Routing:** Использование `File-based routing` (если Next.js) или `TanStack Router` (если SPA).

## 4. База данных (Схема)
- [ ] `users` (id, email, password_hash).
- [ ] `series` (id, tmdb_id, title, total_episodes).
- [ ] `user_progress` (user_id, series_id, episode_number, status, updated_at).

## 5. Deployment & DevOps
- [ ] **Docker:** Написание Dockerfile для Go (multi-stage build) и React.
- [ ] **CI/CD:** GitHub Actions для проверки типов и линтинга.
- [ ] **Monitoring:** Настройка Health-checks на бэкенде.
