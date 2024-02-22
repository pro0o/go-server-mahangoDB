FROM golang:1.17.2-alpine

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o server

EXPOSE 8080

CMD ["./server"]