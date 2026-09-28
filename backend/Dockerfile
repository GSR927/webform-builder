FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o /webform-api ./main.go

FROM alpine:3.19
COPY --from=build /webform-api /webform-api
EXPOSE 8080
ENTRYPOINT ["/webform-api"]
