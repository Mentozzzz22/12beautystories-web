# syntax=docker/dockerfile:1

# ---- Web app ----
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- Go server ----
FROM golang:1.27-alpine AS server
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/admin ./cmd/admin \
 && mkdir -p /out/data/uploads

# ---- Runtime: no shell, no package manager, non-root ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=server /out/server /out/admin /app/
COPY --from=web /src/web/dist /app/web
# Pre-create the upload dir owned by the nonroot user so a fresh volume inherits the ownership.
COPY --from=server --chown=65532:65532 /out/data /data

ENV APP_ADDR=:8080 \
    STATIC_DIR=/app/web \
    UPLOAD_DIR=/data/uploads \
    TZ=Asia/Almaty

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/app/server", "-healthcheck"]
ENTRYPOINT ["/app/server"]
