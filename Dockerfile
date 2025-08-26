# ===== build backend =====
FROM golang:1.25 as gobuilder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /out/sonovel-web ./cmd/sonovel-web

# ===== build frontend =====
FROM node:20-alpine as webbuilder
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm i
COPY web/ ./
RUN npm run build

# ===== runtime =====
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=gobuilder /out/sonovel-web /app/sonovel-web
COPY --from=webbuilder /web/dist /app/web/dist
COPY configs /app/configs
EXPOSE 8080
ENV SOURCES_DIR=/app/configs/sources
USER nonroot:nonroot
ENTRYPOINT ["/app/sonovel-web"]
