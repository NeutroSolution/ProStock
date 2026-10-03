FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/prostock .

FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
LABEL org.opencontainers.image.version=${VERSION}
COPY --from=build /out/prostock /prostock
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/prostock", "healthcheck"]
ENTRYPOINT ["/prostock"]
