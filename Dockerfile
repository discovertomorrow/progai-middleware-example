FROM golang:1.22-alpine3.20 AS build
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY ./cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /llamacpp ./cmd/llamacpp
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /general ./cmd/general

FROM scratch AS final
WORKDIR /app
USER 1001:1001
COPY --from=build /general ./general
COPY --from=build /llamacpp ./llamacpp
EXPOSE 8080
ENTRYPOINT ["./general"]
