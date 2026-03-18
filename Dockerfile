# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

COPY web/package*.json ./
RUN npm ci --legacy-peer-deps

COPY web/ ./

ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

RUN echo "=== Checking Next.js build ===" && \
    ls -la .next/standalone/ && \
    echo "=== server.js exists ===" && \
    test -f .next/standalone/server.js

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

RUN apk add --no-cache ca-certificates postgresql-client bash nodejs

WORKDIR /app

# Copy backend
COPY --from=backend-builder /app/server .
COPY --from=backend-builder /app/migrations ./migrations
COPY scripts/entrypoint.sh .

# Copy Next.js standalone
COPY --from=frontend-builder /app/frontend/.next/standalone ./
COPY --from=frontend-builder /app/frontend/.next/static ./.next/static
COPY --from=frontend-builder /app/frontend/public ./public

RUN echo "=== Final image files ===" && \
    ls -la . && \
    ls -la .next/standalone/ || true

RUN chmod +x entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["./entrypoint.sh"]
