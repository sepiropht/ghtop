FROM golang:1.22-alpine

RUN apk add --no-cache gcc musl-dev

ENV CGO_ENABLED=1

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o web-server .

EXPOSE 8080

CMD ["./web-server"]
