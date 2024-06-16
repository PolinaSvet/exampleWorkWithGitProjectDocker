FROM golang:latest as build-stage

WORKDIR /app
ENV CGO_ENABLED=0
COPY go.mod go.sum logger ./
RUN go mod download
COPY . .
RUN go build -o /app/server main.go

FROM scratch
LABEL version="1.0.1"
LABEL maintainer="Polina<polinasvet6@gmail.com>"
WORKDIR /
COPY --from=build-stage /app/server /server
CMD ["/server"]