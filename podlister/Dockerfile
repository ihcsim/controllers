FROM golang:1.15-buster AS builder
WORKDIR /podlister
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
      -a -o /podlister/bin/podlister .

FROM alpine:3.12.3
COPY --from=builder /podlister/bin/podlister /podlister
CMD ["/podlister"]
