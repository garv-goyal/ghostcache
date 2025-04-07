# GhostCache

## Summary 

GhostCache is a fast distributed cache built for scale and reliability. It features consistent hashing, adaptive TTL with LRU, optional persistence, a simple HTTP JSON API, and a resilient Go client with retries and circuit breaking.

## Run

### Coordinator
```
go run main.go --mode=coordinator --port=9000
```

### Cache Node
```
go run main.go --mode=node --port=8080
```

### Register Node
```
curl -X POST -H "Content-Type: application/json" \
  -d '{"address": "http://localhost:8080"}' \
  http://localhost:9000/register
```
