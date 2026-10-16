FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /kitchen ./cmd/kitchen
RUN go build -o /restaurant ./cmd/restaurant

FROM alpine:3.22 AS kitchen
RUN apk add --no-cache ca-certificates wget
COPY --from=build /kitchen /kitchen
ENTRYPOINT ["/kitchen"]

FROM alpine:3.22 AS restaurant
RUN apk add --no-cache ca-certificates wget
COPY --from=build /restaurant /restaurant
ENTRYPOINT ["/restaurant"]
