# QuickDocs

REST API для загрузки, хранения и раздачи электронных документов с аутентификацией, кешированием и разграничением доступа.

Стек: Go, PostgreSQL, Redis, chi.

---

## Возможности

- Регистрация пользователей (через ADMIN_TOKEN) и аутентификация (token)
- Загрузка документов (multipart: meta/json/file)
- Получение списков с фильтрами/сортировкой, пагинация
- Выдача единичного документа: файл (с корректным MIME) или JSON
- Кеширование GET/HEAD (метаданные, списки, байты файлов), выборочная инвалидация при изменениях
- Единый формат ответов (см. ниже), HEAD без тела
- Request-ID логирование для трассировки

---
## Формат ответов

Всегда HTTP 200. Ошибки — в поле `error`.

```
{
  "error": { "code": 401, "text": "неавторизован" },
  "response": { ... },
  "data": { ... }
}
```

- Поля присутствуют только если заполнены
- `response` — подтверждение действия (например, токен/булевы флаги)
- `data` — содержимое (списки/JSON)

---
## Авторизация

- Передавайте токен одним из способов:
  - Заголовок: `Authorization: Bearer <token>`
  - Query-параметр: `?token=<token>`

---
## Эндпоинты

### 1) Регистрация
```
POST /api/register?admin_token=<ADMIN_TOKEN (из env)>
Content-Type: application/json

{
  "login": "userlogin",
  "pswd":  "Aa1!aaaa"
}

-> 200 { "response": { "login": "userlogin" } }
```

Требования: login ≥ 8, латиница/цифры; пароль ≥ 8, буквы разных регистров, цифра, спецсимвол.

### 2) Аутентификация
```
POST /api/auth
Content-Type: application/json | application/x-www-form-urlencoded

{ "login": "userlogin", "pswd": "Aa1!aaaa" }

-> 200 { "response": { "token": "..." } }
```

### 3) Загрузка документа
```
POST /api/docs
Authorization: Bearer <token>
Content-Type: multipart/form-data

meta: {
  "name": "photo.jpg",
  "file": true,
  "public": false,
  "mime": "image/jpeg"
}
json: { ... }   // опционально
file: <binary>

-> 200 { "data": { "json": { ... }, "file": "photo.jpg" } }
```

### 4) Список документов
```
GET /api/docs?limit=10&offset=0&key=name&value=report&sort=created&order=desc
Authorization: Bearer <token>

-> 200 { "data": { "docs": [ ... ] } }
```

- Параметры: `limit`, `offset`, `key` (name|mime|has_file|is_public), `value`, `sort` (name|created), `order` (asc|desc) не обязательны.
По умолчанию мы получаем первые 10 страниц без фильтров и сортировки.
- Публичные документы пользователя: `GET /api/docs?login=other`

### 5) Один документ
```
GET /api/docs/{id}
Authorization: Bearer <token>
```

- Если `file=true` — выдаётся файл (байты берутся из кэша, при промахе — с диска и кладутся в кэш)
- Если JSON — `-> 200 { "data": { ... } }`

HEAD /api/docs/{id} — нужные заголовки, без тела.

### 6) Удаление документа
```
DELETE /api/docs/{id}
Authorization: Bearer <token>

-> 200 { "response": { "{id}": true } }
```

### 7) Завершение сессии
```
DELETE /api/auth/{token}

-> 200 { "response": { "{token}": true } }
```

---
## Кэширование

- Списки пользователя (`GET /api/docs`, offset=0) — Redis (JSON), ttl 5m
- Метаданные документа — Redis (JSON), ttl 10m
- Байты файла — Redis (binary), ttl 10m
- Инвалидация: загрузка/удаление инвалидирует списки владельца и конкретный документ

---
## Запуск

1) Настройте переменные окружения (например, в `config/.env`):
```
APP_PORT=8080
ADMIN_TOKEN=change-me
POSTGRES_DSN=postgres://user:pass@localhost:5432/docs_db?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
APP_ENV=local
```

2) Разверните БД:
```
psql -d docs_db -f init.sql
```

3) Запустите приложение:
```
go run ./cmd/server
```

---
## Тесты

```
go test ./...
```

Примечание: один тест `auth` использует Redis (должен быть доступен на `REDIS_ADDR`).

---



