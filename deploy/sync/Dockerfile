FROM golang:1.27.3-bookworm AS build
WORKDIR /src
COPY services/sync/go.mod services/sync/go.sum ./
RUN go mod download
COPY services/sync/ ./
ENV CGO_ENABLED=0 GOWORK=off
RUN go build -trimpath -ldflags="-s -w" -o /out/arion-sync ./cmd/sync

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /out/arion-sync /arion-sync
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/arion-sync"]
CMD ["serve"]
