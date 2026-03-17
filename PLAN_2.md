# План разработки сервиса "SeriesTracker" (v2)

## 1. Проектирование (Архитектура)
- [x] **База данных:** PostgreSQL (основные данные), Redis (кэширование сессий).
- [x] **API:** RESTful
- [x] **Авторизация:** HTTP-only cookies + сессии в Redis

---

## 2. Backend (Go 1.25)

### 2.1 Инициализация проекта
- [ ] `go mod init`
- [ ] Структура Clean Architecture

### 2.2 База данных
- [ ] Подключение PostgreSQL через pgx/v5
- [ ] Настройка sqlc для генерации кода
- [ ] Написание SQL-миграций

### 2.3 Routing и Middleware
- [ ] Настройка net/http ServeMux
- [ ] Middleware: slog (логирование), Recovery, CORS
- [ ] Health-check эндпоинт

### 2.4 Бизнес-логика
- [ ] Интеграция с kinopoiskapiunofficial
- [ ] Эндпоинты поиска сериалов
- [ ] CRUD для списков пользователя
- [ ] Обновление прогресса просмотра

### 2.5 Concurrency
- [ ] errgroup для параллельных запросов

---

## 3. Frontend (React 19 + Next.js 15)

- [ ] Инициализация Next.js проекта
- [ ] Настройка Tailwind CSS + Shadcn/UI
- [ ] Server Components для страниц сериалов
- [ ] Server Actions для форм
- [ ] TanStack Query для синхронизации
- [ ] useOptimistic для галочек

---

## 4. Схема базы данных (PostgreSQL)

### users
| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| id | UUID | PK, default gen_random_uuid() |
| email | VARCHAR(255) | UNIQUE, NOT NULL |
| password_hash | VARCHAR(60) | NOT NULL (bcrypt) |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() |

**Индексы:** idx_users_email

### series
| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| id | UUID | PK, default gen_random_uuid() |
| kinopoisk_id | INTEGER | UNIQUE, NOT NULL |
| title | VARCHAR(500) | NOT NULL |
| original_title | VARCHAR(500) | |
| poster_url | TEXT | |
| year | INTEGER | |
| description | TEXT | |
| total_episodes | INTEGER | |
| total_seasons | INTEGER | |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() |

**Индексы:** idx_series_kinopoisk_id, idx_series_title

### user_progress
| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| id | UUID | PK, default gen_random_uuid() |
| user_id | UUID | FK -> users(id), ON DELETE CASCADE |
| series_id | UUID | FK -> series(id), ON DELETE CASCADE |
| current_season | INTEGER | NOT NULL, DEFAULT 1 |
| current_episode | INTEGER | NOT NULL, DEFAULT 0 |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'watching' |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() |

**Индексы:** idx_user_progress_user_status (user_id, status), idx_user_progress_user

**Уникальный ключ:** (user_id, series_id)

---

## 5. Deployment & DevOps

- [ ] Docker: multi-stage build для Go и React
- [ ] Docker-compose для локальной разработки
- [ ] CI/CD: GitHub Actions (типы, линтинг)
- [ ] Health-checks

---

## Технологический стек

- **Backend:** Go 1.25, pgx/v5, sqlc, Redis
- **Frontend:** Next.js 15, React 19, Tailwind CSS, Shadcn/UI, TanStack Query
- **DB:** PostgreSQL 16, Redis
- **DevOps:** Docker, GitHub Actions
