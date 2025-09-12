# Offline DeBERTa Inference (ONNX Runtime) for PromptShield

This folder contains scripts and an example FastAPI microservice to serve DeBERTa predictions locally in air‑gapped environments with high throughput.

What you get
- Export to ONNX (and optional INT8 quantization) for CPU throughput
- FastAPI service using ONNX Runtime
- Windows PowerShell scripts to automate steps

## Prerequisites
- Python 3.10+
- PowerShell (Windows)
- Model files downloaded offline to a local directory, e.g.: C:\models\protectai-deberta-v2
- Tokenizer: place tokenizer.json in C:\Users\Dawan\promptshield-v0.2.0\assets\tokenizers

## 1) Export to ONNX and Quantize (Windows)

Run `Export-DebertaOnnx.ps1` to export to ONNX and optional INT8 quantization.

```powershell
# Example usage
# -ModelDir: source HF model directory copied offline
# -OutputDir: destination for ONNX
# -Quantize: switch to enable INT8 model

# From repository root
pwsh -File .\tools\deberta-offline\Export-DebertaOnnx.ps1 -ModelDir "C:\models\protectai-deberta-v2" -OutputDir "C:\models\deberta-onnx" -Quantize
```

## 2) Run the Inference Server (Windows)

Start the FastAPI + ONNX Runtime server with `Start-DebertaServer.ps1`:

```powershell
# From repository root
pwsh -File .\tools\deberta-offline\Start-DebertaServer.ps1 -OnnxDir "C:\models\deberta-onnx" -TokenizerDir "C:\Users\Dawan\promptshield-v0.2.0\assets\tokenizers" -Port 8089 -Workers 4
```

It will listen on http://127.0.0.1:8089/infer

## 3) Configure PromptShield
Set the following environment variables for PromptShield:

- PS_SEMANTIC_ENABLED=true
- PS_DEBERTA_ENDPOINT=http://127.0.0.1:8089/infer
- PS_DEBERTA_TOKENIZER_JSON=C:\Users\Dawan\promptshield-v0.2.0\assets\tokenizers\tokenizer.json
- PS_DEBERTA_LOWERCASE=true (or false if your model is cased)
- HF_HUB_OFFLINE=1
- TRANSFORMERS_OFFLINE=1

## Files
- `app.py` — FastAPI microservice using ONNX Runtime
- `Export-DebertaOnnx.ps1` — Exports model to ONNX and optional INT8 quantization
- `Start-DebertaServer.ps1` — Starts the server with desired worker count and EP

## Notes
- Tune workers to CPU cores for best throughput.
- For GPU, change providers in app.py to include "CUDAExecutionProvider".
- Keep max_length aligned with your latency/accuracy tradeoffs.

