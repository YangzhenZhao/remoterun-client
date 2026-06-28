FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/remoterun-server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/remoterun-server /usr/local/bin/remoterun-server

EXPOSE 8080

CMD ["remoterun-server"]
