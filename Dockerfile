FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o pokemon-web .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/pokemon-web .
COPY frontend/ ./frontend/
COPY roms/ ./roms/

ENV FRONTEND_DIR=/app/frontend
ENV ROMS_DIR=/app/roms

EXPOSE 10000
CMD ["./pokemon-web"]
