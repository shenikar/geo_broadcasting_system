# Используем официальный образ Go в качестве базового образа для сборки
FROM golang:1.25-alpine AS builder

# Устанавливаем необходимые зависимости для сборки CGO (например, для PostGIS)
RUN apk add --no-cache gcc musl-dev postgresql-dev

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем go.mod и go.sum и загружаем зависимости
COPY go.mod .
COPY go.sum .
RUN go mod download

# Копируем исходный код приложения
COPY . .

# Собираем приложение
RUN CGO_ENABLED=1 go build -o geo_broadcasting_system cmd/main.go

FROM alpine:latest

# Устанавливаем ca-certificates для работы с HTTPS
RUN apk add --no-cache ca-certificates

# Устанавливаем клиент PostgreSQL (для утилиты `psql` и работы с БД)
RUN apk add --no-cache postgresql-client

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем собранный исполняемый файл из предыдущего этапа
COPY --from=builder /app/geo_broadcasting_system .

# Копируем файлы миграций
COPY migrations ./migrations

# Открываем порт, на котором будет работать приложение
EXPOSE 8080

# Команда для запуска приложения
CMD ["./geo_broadcasting_system"]