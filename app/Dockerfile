FROM golang:1.16-buster AS build
# go env can be removed later
RUN go env -w GOPROXY=direct

WORKDIR /app
COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY main.go ./
RUN go build -o /memorydb-go-app

FROM gcr.io/distroless/base-debian10
WORKDIR /
COPY --from=build /memorydb-go-app /memorydb-go-app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/memorydb-go-app"]