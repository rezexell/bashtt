# bashtt

Сервис для удаленного создания и мониторинга bash-скриптов на Linux-машинах.

## Что делает проект

Сервис умеет:

- подключаться к Linux-машинам по SSH;
- создавать bash-скрипты по шаблонам;
- устанавливать агент мониторинга;
- отслеживать:
  - открытие скрипта (`open`);
  - выполнение скрипта (`execute`);
- принимать события от агента через HTTP callback;
- сохранять данные в PostgreSQL:
  - машины;
  - созданные скрипты;
  - события выполнения.

Используемые технологии:

- Go
- PostgreSQL
- pgx
- golang-migrate
- Docker Compose
- SSH
- fanotify (Linux)

---

## Запуск проекта

### 1. Запустить PostgreSQL

```bash
make up
````

---

### 2. Установить миграции

Установить golang-migrate:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Применить миграции:

```bash
make migrate-up
```

---

### 3. Собрать проект

```bash
make build
```

---

### 4. Запустить сервер

```bash
make run
```

После запуска будут доступны:

```
Create API:
http://localhost:8080

Callback API:
http://localhost:8081
```

---

## Тестовый SSH контейнер

Запуск тестовой машины:

```bash
make ssh-up
```

Контейнер будет доступен:

```
host: 127.0.0.1
port: 2222
user: root
password: testpassword
```

---

## Пример создания скрипта

```bash
curl \
-X POST \
http://127.0.0.1:8080/create \
-H 'Content-Type: application/json' \
-d '{
    "host": "127.0.0.1",
    "user": "root",
    "password": "testpassword",
    "template": "template1"
}'
```

После этого:

* на машине будет создан bash-скрипт;
* будет установлен агент;
* агент начнет отправлять события.

---

## Переменные окружения

Создать файл `.env`:

```env
HTTP_CREATE_ADDR=:8080
HTTP_CALLBACK_ADDR=:8081

SSH_PORT=2222
SSH_TIMEOUT=10s

POSTGRES_URL=postgres://bashtt:bashtt@localhost:5432/bashtt?sslmode=disable

AGENT_BINARY_PATH=./bin/agent
AGENT_REMOTE_BINARY_PATH=/tmp/bashtt/agent
AGENT_WATCH_DIR=/tmp/bashtt
AGENT_CALLBACK_URL=http://host.docker.internal:8081/callback

LOG_LEVEL=debug
BASH_TEST=1
```

---

## Make команды

Запуск PostgreSQL:

```bash
make up
```

Остановка:

```bash
make down
```

Логи PostgreSQL:

```bash
make logs
```

Миграции:

```bash
make migrate-up
make migrate-down
make migrate-version
```

Сборка:

```bash
make build
```

Запуск сервера:

```bash
make run
```

Тесты:

```bash
make test
```

