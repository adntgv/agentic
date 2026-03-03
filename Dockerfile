FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o agentic .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/agentic .
COPY --from=builder /app/public ./public
EXPOSE 8080
CMD ["./agentic", "serve", "--addr", ":8080", "--data", "/data"]
