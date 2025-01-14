FROM golang:1.23 AS build

WORKDIR /app
COPY . .
RUN go clean --modcache
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build ./cmd/main

FROM alpine:latest

WORKDIR /root
COPY --from=build /app/main .

EXPOSE 8080
CMD ["./main"]
