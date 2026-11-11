# Build stage: compile a static binary.
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/calculator-api .

# Runtime stage: distroless image with only the binary (~10 MB, non-root).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/calculator-api /calculator-api
ENV PORT=8080 CORS_ORIGIN=""
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/calculator-api"]
