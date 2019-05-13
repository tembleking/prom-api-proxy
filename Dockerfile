FROM golang:1.12 as builder

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build /app
RUN strip /app/prom-api-proxy

FROM alpine
COPY --from=builder /app/prom-api-proxy /bin/prom-api-proxy
ENTRYPOINT ["prom-api-proxy"]