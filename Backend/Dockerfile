FROM golang:1.24 AS builder

WORKDIR /app

COPY . .

RUN go mod tidy && go build -o main .

FROM gcr.io/distroless/base

COPY --from=builder /app/main /main

CMD ["/main"]