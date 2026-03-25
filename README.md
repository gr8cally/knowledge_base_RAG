# knowledge_base_RAG

## Setup

### Required environment variables

These must be set for the app to function correctly:

```bash
OPENROUTER_API_KEY=your_openrouter_key
MODEL_NAME=nvidia/nemotron-3-nano-30b-a3b:free
HF_TOKEN=your_huggingface_token
```

`OPENROUTER_API_KEY` and `MODEL_NAME` are required by the app at startup.  
`HF_TOKEN` is required for embedding generation with the default Hugging Face router setup.

### Bootstrap Run

1. Copy the example env file and fill in the required values:

```bash
cp .env.example .env
```

2. Generate templates and run:

```bash
go build ./cmd/server
go run ./cmd/server
```

3. Health checks:

```bash
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
```

## Environment Variables

See [.env.example](/Users/calugo/Projects/kno_base_RAG/.env.example) for the full list of required and optional variables.
