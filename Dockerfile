FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /out/gateway /app/gateway
COPY config.example.yaml /app/config.yaml

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/gateway"]
CMD ["-config", "/app/config.yaml"]
