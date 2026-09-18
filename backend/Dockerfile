FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api && CGO_ENABLED=0 go build -o /importer ./cmd/importer

FROM alpine:3.20
WORKDIR /app
COPY --from=build /api /importer ./
COPY migrations ./migrations
ENV PORT=8080 MIGRATIONS_DIR=/app/migrations
EXPOSE 8080
CMD ["/app/api"]
