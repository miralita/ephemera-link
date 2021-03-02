FROM golang:1.15.7-buster AS builder
WORKDIR /build
COPY . ./
RUN go build -o ephemera-link .

FROM debian:buster
WORKDIR /app
COPY --from=builder /build/ephemera-link /app/ephemera-link
COPY --from=builder /build/templates /app/templates
COPY --from=builder /build/static /app/static
RUN apt update && apt install -y ca-certificates
CMD ["./ephemera-link"]
EXPOSE 8834
