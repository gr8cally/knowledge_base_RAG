# knowledge_base_RAG

## Bootstrap Run

1. Set required env vars:

```bash
export OPENROUTER_API_KEY="your-key"
export MODEL_NAME="nvidia/nemotron-3-nano-30b-a3b:free"
```

2. Generate templates and run:

```bash
templ generate
go build ./cmd/server
go run ./cmd/server
```

3. Health checks:

```bash
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
```
