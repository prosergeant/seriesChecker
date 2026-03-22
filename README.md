# SeriesTracker

Трекер для отслеживания сериалов. Позволяет искать сериалы через Kinopoisk API, добавлять в список и отслеживать прогресс просмотра.

## Технологии

- **Backend:** Go 1.25, PostgreSQL, Redis
- **Frontend:** Next.js 16, React 19, Tailwind CSS, TanStack Query
- **Деплой:** Docker, Render.com

## Архитектура

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Container                     │
│                                                         │
│   ┌──────────────────┐     ┌──────────────────────────┐ │
│   │   Next.js (3000) │     │   Go Server (8080)       │ │
│   │                  │     │                          │ │
│   │   Фронтенд       │     │   API handlers           │ │
│   │   App Router     │◄────│   /api/*                 │ │
│   │                  │     │                          │ │
│   └──────────────────┘     └──────────────────────────┘ │
│           │                          │                  │
└───────────┼──────────────────────────┼──────────────────┘
            │                          │
      ┌─────▼─────-┐            ┌──────▼──────┐
      │ PostgreSQL │            │    Redis    │
      │   (DB)     │            │ (Sessions)  │
      └───────────-┘            └─────────────┘
```

## Структура проекта

```
seriesChecker/
├── cmd/server/main.go        # Точка входа Go сервера
├── internal/
│   ├── config/               # Конфигурация
│   ├── database/             # Подключение к PostgreSQL, Redis
│   ├── handler/              # HTTP обработчики (auth, series, progress)
│   ├── kinopoisk/            # Клиент для Kinopoisk API
│   ├── middleware/           # CORS, логирование, auth
│   ├── model/                # Модели данных
│   ├── repository/           # Работа с БД
│   └── service/              # Бизнес-логика
├── migrations/               # SQL миграции
├── scripts/
│   └── entrypoint.sh         # Entrypoint для Docker
├── web/                      # Next.js приложение
│   ├── app/                  # App Router страницы
│   ├── components/           # UI компоненты
│   └── lib/                  # Утилиты, API клиент
├── docker-compose.yml        # Локальная разработка (PostgreSQL, Redis)
└── Dockerfile                # Production билд
```

## Локальная разработка

### Требования

- Docker Desktop
- Go 1.25+
- Node.js 20+

### 1. Запуск баз данных

```bash
docker compose up -d postgres redis
```

### 2. Запуск бэкенда

```bash
go build -o ./bin/server ./cmd/server
./bin/server
```

Сервер запустится на `http://localhost:8080`

### 3. Запуск фронтенда

```bash
cd web
npm install
npm run dev
```

Фронтенд доступен на `http://localhost:3000`

### 4. Настройка переменных окружения

Скопируй `.env.example` в `.env` и заполни:

```bash
cp .env.example .env
```

## Переменные окружения

| Переменная | Описание | Пример |
|------------|----------|--------|
| `DB_HOST` | Хост PostgreSQL | `localhost` |
| `DB_PORT` | Порт PostgreSQL | `5432` |
| `DB_USER` | Пользователь БД | `seriestracker` |
| `DB_PASSWORD` | Пароль БД | `seriestracker` |
| `DB_NAME` | Имя базы данных | `seriestracker` |
| `REDIS_HOST` | Хост Redis | `localhost` |
| `REDIS_PORT` | Порт Redis | `6379` |
| `KINOPOISK_API_KEY` | API ключ Kinopoisk | `xxx-xxx-xxx` |
| `SESSION_COOKIE_NAME` | Имя cookie сессии | `session_id` |
| `COOKIE_DOMAIN` | Домен для cookie | `.example.com` |
| `ALLOWED_ORIGINS` | CORS origins (через запятую) | `https://example.com` |
| `NEXT_PUBLIC_API_URL` | URL API для фронтенда | `http://localhost:8080` |

## Docker (Production)

### Сборка образа

```bash
# Билд с локальным API URL
docker build -t series-tracker:test --build-arg NEXT_PUBLIC_API_URL=http://localhost:8080 .

# Билд для прода
docker build -t series-tracker --build-arg NEXT_PUBLIC_API_URL=https://your-domain.com .
```

### Запуск контейнера

```bash
docker run -d \
  --name series-tracker \
  -p 8080:8080 \
  -e DB_HOST=your_postgres_host \
  -e DB_PORT=5432 \
  -e DB_USER=your_user \
  -e DB_PASSWORD=your_password \
  -e DB_NAME=your_db \
  -e REDIS_HOST=your_redis_host \
  -e REDIS_PORT=6379 \
  -e ALLOWED_ORIGINS=https://your-domain.com \
  -e COOKIE_DOMAIN=.your-domain.com \
  series-tracker
```

## Работа с базой данных

### Локальный дамп

```bash
# Создать дамп локальной базы
docker exec seriestracker_postgres pg_dump -U seriestracker seriestracker > dump.sql

# Схема + данные
pg_dump -U seriestracker seriestracker > full_dump.sql

# Только схема (без данных)
pg_dump --schema-only -U seriestracker seriestracker > schema_only.sql

# Только данные конкретной таблицы
pg_dump --data-only --column-inserts -t series -U seriestracker seriestracker > series_data.sql
```

### Восстановление на удалённом сервере

```bash
# Через Docker (подключение к Render, например)
cat dump.sql | docker run -i --rm postgres:16 psql "postgres://user:pass@host:5432/dbname"

# Или проще
docker run -i --rm postgres:16 psql "postgres://user:pass@host:5432/dbname" < dump.sql
```

### Варианты синхронизации баз

#### Вариант 1: Полная замена
Удаляет все таблицы на сервере и заливает заново.

```bash
# Подключиться к серверу и удалить таблицы
psql "postgres://user:pass@host:5432/dbname" -c "DROP TABLE IF EXISTS schema_migrations, users, series, user_progress CASCADE;"

# Залить дамп
cat dump.sql | docker run -i --rm postgres:16 psql "postgres://user:pass@host:5432/dbname"
```

**⚠️ Внимание:** Это удалит ВСЕ данные на сервере!

#### Вариант 2: Добавить данные (без конфликтов)
Добавляет только новые строки, пропуская дубликаты.

```bash
# Дамп данных (без схемы)
pg_dump --data-only -U seriestracker seriestracker > data_only.sql

# Восстановление (INSERT добавит строки)
docker run -i --rm postgres:16 psql "postgres://user:pass@host:5432/dbname" < data_only.sql
```

#### Вариант 3: Дамп с REPLACE
Использует `INSERT ... ON CONFLICT DO UPDATE`.

```bash
# Экспорт в формате COPY с обновлением
pg_dump --data-only --column-inserts -t series -t user_progress -U seriestracker seriestracker > updatable_data.sql

# Восстановление
docker run -i --rm postgres:16 psql "postgres://user:pass@host:5432/dbname" < updatable_data.sql
```

```bash
# сделать копию базы с сервера (рендер)
docker run --rm postgres:18 pg_dump "external db url" > render_backup.sql
```

### Утилиты для работы с PostgreSQL

```bash
# Подключиться к локальной базе
docker exec -it seriestracker_postgres psql -U seriestracker -d seriestracker

# Посмотреть таблицы
\dt

# Посмотреть данные
SELECT * FROM users;
SELECT * FROM series LIMIT 5;

# Удалить таблицы
DROP TABLE IF EXISTS users, series, user_progress, schema_migrations CASCADE;

# Выйти
\q
```

## Деплой на Render.com

### 1. Создать сервисы

1. **PostgreSQL**: New → PostgreSQL → настроить имя, план Free
2. **Redis**: New → Redis → настроить имя, план Free
3. **Web Service**: New → Web Service → подключить GitHub репозиторий

### 2. Настройки Web Service

- **Environment**: Docker
- **Build Command**: (пусто)
- **Start Command**: (пусто)

### 3. Environment Variables

В настройках сервиса добавить:

| Key | Value |
|-----|-------|
| `DB_HOST` | Из PostgreSQL |
| `DB_PORT` | `5432` |
| `DB_USER` | Из PostgreSQL |
| `DB_PASSWORD` | Из PostgreSQL |
| `DB_NAME` | Из PostgreSQL |
| `REDIS_HOST` | Из Redis |
| `REDIS_PORT` | `6379` |
| `REDIS_PASSWORD` | Из Redis (или пусто) |
| `KINOPOISK_API_KEY` | Из Render Secrets |
| `SESSION_COOKIE_NAME` | `session_id` |
| `COOKIE_DOMAIN` | `.onrender.com` |
| `ALLOWED_ORIGINS` | `https://your-service.onrender.com` |

### 4. Build Arguments

В секции Build добавить:

| Key | Value |
|-----|-------|
| `NEXT_PUBLIC_API_URL` | `https://your-service.onrender.com` |

### 5. Пересборка

После изменений:
- Settings → Clear build cache & deploy
- Или Manual Deploy → Deploy latest commit

## API Endpoints

### Auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Регистрация |
| POST | `/api/auth/login` | Вход |
| POST | `/api/auth/logout` | Выход |
| GET | `/api/auth/me` | Текущий пользователь |

### Series

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/series/search?q=` | Поиск сериалов |
| GET | `/api/series/:id` | Информация о сериале |

### Progress

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/progress` | Список сериалов пользователя |
| POST | `/api/progress` | Добавить/обновить прогресс |
| DELETE | `/api/progress/:seriesId` | Удалить из списка |

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Проверка работоспособности |

## Статус миграций

Таблица `schema_migrations` отслеживает выполненные миграции. При запуске контейнера автоматически применяются новые миграции.

```sql
-- Проверить статус миграций
SELECT * FROM schema_migrations;
```

## Лицензия

MIT
