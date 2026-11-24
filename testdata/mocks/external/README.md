# Paid external service mocks

Recorded from media `c807f663-b313-4457-b492-e5841d2b21b7` (no video/audio binaries).

| Path | Service |
|------|---------|
| `assemblyai/words.json` | AssemblyAI transcript |
| `assemblyai/provider.job` | AssemblyAI job id marker |
| `llm/llm.json` | Groq / LLM analysis |

## Enable

```env
MOCK_EXTERNAL_DIR=/app/testdata/mocks/external
TRANSCRIBE_PROVIDER=mock
LLM_PROVIDER=mock
```

Workers copy these files into the active media’s MinIO keys instead of calling paid APIs.
