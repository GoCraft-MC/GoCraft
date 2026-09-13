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
ENV GOCRAFT_HEALTHCHECK_PORT=8081

COPY --from=build /out/gocraft /usr/local/bin/gocraft
COPY --from=build --chown=nonroot:nonroot /out/data /data

WORKDIR /data

EXPOSE 25565/tcp
EXPOSE 19106/udp

USER nonroot

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=5 CMD [ "/usr/local/bin/gocraft", "--check-health" ]
ENTRYPOINT [ "/usr/local/bin/gocraft" ]
