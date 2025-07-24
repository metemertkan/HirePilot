FROM golang:1.24 AS builder

WORKDIR /app

# Copy shared library first
COPY shared ./shared

# Copy Backend files
COPY Backend ./Backend

# Set working directory to JobService
WORKDIR /app/Backend

RUN go mod tidy && go build -o main .

FROM gcr.io/distroless/base

COPY --from=builder /app/Backend/main /main

CMD ["/main"]