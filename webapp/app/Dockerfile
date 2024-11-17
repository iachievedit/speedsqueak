FROM node:22-alpine

WORKDIR /app

COPY package*.json ./
COPY .env ./

RUN npm install

COPY backend ./backend
COPY frontend ./frontend

EXPOSE 3333

CMD ["node", "backend/app.js"]
