FROM golang:1.25-alpine

WORKDIR /app

# Копируем зависимости и скачиваем их
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем статический бинарник (CGO_ENABLED=0 решает проблему с Alpine)
# Флаг -o /app/main гарантирует, что файл окажется именно в /app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/server

EXPOSE 5000

# Запускаем по абсолютному пути, чтобы избежать путаницы с WORKDIR
CMD ["/app/main"]