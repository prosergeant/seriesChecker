# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

COPY web/package*.json ./
RUN npm ci

COPY web/ ./

ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend-builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Final stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates postgresql-client bash

WORKDIR /app

COPY --from=backend-builder /app/server .
COPY --from=backend-builder /app/migrations ./migrations
COPY scripts/entrypoint.sh .

RUN chmod +x entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["./entrypoint.sh"]
