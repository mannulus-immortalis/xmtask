# build
FROM golang:latest as builder 
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api


# deploy
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api .
EXPOSE 8080 8080
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.9.0/wait ./wait
RUN chmod +x ./wait
CMD ./wait && ./api
