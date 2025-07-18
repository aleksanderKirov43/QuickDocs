# QuickDocs

**QuickDocs** — это REST API-сервис для загрузки, хранения и получения документов с поддержкой аутентификации пользователей, кешированием и разграничением доступа.

Проект написан на **Go**, реализован **в чистом виде без фреймворков**, с применением слоистой архитектуры, инициализацией зависимостей и использованием `Redis` и `PostgreSQL`.

---

## Функциональность

-  Загрузка документов
-  Аутентификация (логин/регистрация)
-  Кеширование документов (в памяти)
-  Middleware для проверки доступа (только владелец может получить файл)
-  PostgreSQL для хранения пользователей и метаданных документов
-  Redis для хранения сессий
-  Простая и расширяемая архитектура

---
## Примеры API-запросов

### Регистрация
```
POST /api/register
Content-Type: application/json

{
"login": "testuser",
"pswd": "secure123"
}
```

###  Авторизация
```
POST /api/auth
Content-Type: application/json

{
  "login": "testuser",
  "password": "secure123"
}
```
### Загрузка документа (авторизован)

```
POST /api/docs/upload
Authorization:<session_token>
Content-Type: multipart/form-data

file=<ваш файл>
meta=<{"public":true}> (или false, в зависимости от того публичный докукмент или нет)
```
### Получение документа (по ID)
```
GET /api/docs/{id}
Authorization: <session_token>
```
### Получение всех документов 
```
GET /api/docs
Authorization:<session_token>
```
### Получение всех документов отдельного пользователя по id
```
GET /api/docs/user/{userID}
Authorization:<session_token>
```
### Удаление документа (по ID)
```
DELETE /api/docs/{id}
Authorization:<session_token>
```
### Проверка авторизации польлзователя
```
HEAD /api/docs/user
Authorization:<session_token>
```
### Проверка cуществования документа
```
HEAD /api/docs/{id}
Authorization:<session_token>
```


###  Запуск

#### Создайте файл конфигурации
На основе .env.example создайте файл .env
```
Отредактируйте .env, указав:
POSTGRES_DSN — данные подключения к вашей PostgreSQL базе;
REDIS_ADDR, REDIS_PASSWORD — параметры Redis;
ADMIN_TOKEN — секретный токен администратора.
```
Создайте БД docs_db заранее.
Структуру разверните используя файл init.sql.


### ✅ TODO для продакшн-версии

- [ ] Написать юнит-тесты для `auth.Service`, `docs.Service`
- [ ] Контейнеризация



