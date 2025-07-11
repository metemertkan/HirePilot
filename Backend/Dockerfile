FROM golang:1.24 AS builder

WORKDIR /app


COPY cv_utils.go ./
COPY cv.go ./
COPY db.go ./
COPY main.go ./
COPY go.mod ./
COPY job.go ./
COPY prompt.go ./

RUN go mod tidy && go build -o main .

FROM gcr.io/distroless/base

COPY --from=builder /app/main /main

CMD ["/main"]