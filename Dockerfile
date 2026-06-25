FROM --platform=$BUILDPLATFORM golang:1.23.0-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN echo "Building on $BUILDPLATFORM for $TARGETPLATFORM"

WORKDIR /build
COPY . ./
RUN GOOS=linux GOARCH=$(echo ${TARGETPLATFORM} | cut -d '/' -f2) go build -o ephemera-link .


FROM alpine:3

LABEL org.opencontainers.image.description="Simple web app for creating encrypted secrets that can be viewed only once via unique random link."

WORKDIR /app

COPY --from=builder /build/ephemera-link /app/ephemera-link
COPY --from=builder /build/templates /app/templates
COPY --from=builder /build/static /app/static
COPY --from=builder /build/locales /app/locales

ENV GIN_MODE=release

CMD ["./ephemera-link"]
