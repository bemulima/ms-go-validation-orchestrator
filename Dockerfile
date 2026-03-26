FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/ms-go-validation-orchestrator ./cmd/ms-go-validation-orchestrator

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=build /out/ms-go-validation-orchestrator /usr/local/bin/ms-go-validation-orchestrator

EXPOSE 8080

ENTRYPOINT ["ms-go-validation-orchestrator"]
