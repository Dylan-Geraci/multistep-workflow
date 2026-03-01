FROM golang:1.23-alpine AS build

RUN apk add --no-cache git

WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/server /bin/server
EXPOSE 8080
ENTRYPOINT ["/bin/server"]
