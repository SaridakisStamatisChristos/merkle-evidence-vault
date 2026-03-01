FROM golang:1.20-alpine AS build
WORKDIR /src
COPY . .
WORKDIR /src/services/checkpoint-svc
RUN go build -o /bin/checkpoint ./cmd/checkpoint

FROM alpine:3.18
COPY --from=build /bin/checkpoint /bin/checkpoint
EXPOSE 8081
ENTRYPOINT ["/bin/checkpoint"]
