---
name: local-first
description: >-
  Prefer already-installed local Ollama models and the local GPU over cloud tokens. Use when the user says "just use one of those models, locally", "ollama is already set up", "pull down a model from huggingface using ollama", "using my local graphics card", "minimize token consumption", or "quit wasting tokens."
user-invocable: true
---

# local-first

Use a local model first. Ollama is already set up. Do not reach for a cloud provider as the default.

## Steps

1. Prefer an already-pulled Ollama model on this machine and the local NVIDIA GPU.
2. If a coding model is missing, pull it from Hugging Face through Ollama onto the local card.
3. Minimize token consumption. Do not burn cloud tokens when a local model can do the job.
4. Do not default to SpaceXAI or other cloud AI SDKs unless the user names them.
5. Do not poll for GPU or job status. If asked how it is going, report current artifacts and GPU use once.
