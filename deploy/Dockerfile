# syntax=docker/dockerfile:1
# One image: the Go binary with the web app embedded (spec §14). Build from the repository root:
#   docker build -f deploy/Dockerfile -t uniquest .

FROM node:22-alpine AS web
WORKDIR /src/apps/web
COPY apps/web/package.json apps/web/package-lock.json apps/web/.npmrc ./
RUN npm ci --no-audit --no-fund
COPY content /src/content
COPY packages /src/packages
COPY apps/web /src/apps/web
ARG APP_VERSION=dev
ENV PUBLIC_API_MODE=http APP_VERSION=${APP_VERSION}
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY content content
COPY apps/api apps/api
COPY --from=web /src/apps/web/build apps/api/internal/webui/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/uniquest ./apps/api/cmd/uniquest

FROM gcr.io/distroless/static-debian12:nonroot
ARG APP_VERSION=dev
ENV UQ_ADDR=:8080 UQ_VERSION=${APP_VERSION}
COPY --from=api /out/uniquest /uniquest
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=15s --timeout=4s --retries=3 CMD ["/uniquest", "health"]
ENTRYPOINT ["/uniquest"]
CMD ["serve"]
