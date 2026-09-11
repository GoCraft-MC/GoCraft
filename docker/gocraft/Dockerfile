# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/gocraft .

RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/gocraft /usr/local/bin/gocraft
COPY --from=build --chown=nonroot:nonroot /out/data /data

WORKDIR /data

EXPOSE 25565/tcp
EXPOSE 19106/udp

USER nonroot

ENTRYPOINT ["/usr/local/bin/gocraft"]
