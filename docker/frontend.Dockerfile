FROM node:18-alpine AS builder

WORKDIR /workspace

COPY web/package*.json ./
RUN npm install

COPY web .

RUN npm run build

FROM nginx:1.25-alpine

COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /workspace/dist /usr/share/nginx/html

