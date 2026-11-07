ARG GO_VERSION=1.27

# Stage 1: compile
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# Copy the dependency files first so Docker caches the download step
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Stage 2: tiny runtime image with just the binary
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]