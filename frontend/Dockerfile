FROM node:22-alpine

WORKDIR /app

# 1. Копируем только файлы зависимостей
COPY package.json package-lock.json ./

# 2. Устанавливаем зависимости (создается node_modules)
RUN npm install

# 3. Копируем остальной код
COPY . .

# 4. Запускаем dev-сервер
CMD ["npm", "run", "dev"]