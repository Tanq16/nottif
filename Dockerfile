FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o nottif .

FROM alpine:latest
WORKDIR /app
RUN mkdir -p /app/data
COPY --from=builder /app/nottif .
EXPOSE 8080
CMD ["/app/nottif"]