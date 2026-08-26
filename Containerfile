# Build
FROM docker.io/library/golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/discord-mcp ./cmd/discord-mcp

# Run
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/discord-mcp /discord-mcp
USER 65532:65532
EXPOSE 9091
ENTRYPOINT ["/discord-mcp"]
CMD ["serve"]
