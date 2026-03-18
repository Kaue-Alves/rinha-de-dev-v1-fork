FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /bin/rinha-api ./main.go

FROM gcr.io/distroless/static-debian12:nonroot

ENV APP_PORT=8080

COPY --from=builder /bin/rinha-api /rinha-api

EXPOSE 8080

ENTRYPOINT ["/rinha-api"]
