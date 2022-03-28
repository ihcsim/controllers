FROM golang:1.18 as build
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o snapdiff ./cmd/...

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=build /build /bin
ENTRYPOINT ["/bin/snapdiff"]
