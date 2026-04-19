FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/load-test .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/load-test /load-test
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/load-test"]
