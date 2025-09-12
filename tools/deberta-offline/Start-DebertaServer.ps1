param(
    [Parameter(Mandatory=$true)][string]$OnnxDir,
    [Parameter(Mandatory=$true)][string]$TokenizerDir,
    [int]$Port=8089,
    [int]$Workers=4,
    [ValidateSet("cpu","cuda")][string]$Device="cpu"
)

$ErrorActionPreference = "Stop"

python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install --upgrade pip
pip install fastapi uvicorn onnxruntime transformers numpy

$env:HF_HUB_OFFLINE = "1"
$env:TRANSFORMERS_OFFLINE = "1"

Write-Host "Starting offline DeBERTa server..."
$env:MODEL_DIR = $OnnxDir
$env:TOKENIZER_DIR = $TokenizerDir
$env:DEVICE = $Device

python - << 'PY'
import os
from fastapi import FastAPI
from pydantic import BaseModel
from typing import List, Union
import onnxruntime as ort
from transformers import AutoTokenizer
import numpy as np

MODEL_DIR = os.environ["MODEL_DIR"]
TOKENIZER_DIR = os.environ["TOKENIZER_DIR"]
DEVICE = os.environ.get("DEVICE", "cpu").lower()

model_path = os.path.join(MODEL_DIR, "model-int8.onnx")
if not os.path.exists(model_path):
    model_path = os.path.join(MODEL_DIR, "model.onnx")

so = ort.SessionOptions()
so.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
providers = ["CPUExecutionProvider"] if DEVICE != "cuda" else ["CUDAExecutionProvider", "CPUExecutionProvider"]
session = ort.InferenceSession(model_path, sess_options=so, providers=providers)

# Load tokenizer strictly from local dir
os.environ["TRANSFORMERS_OFFLINE"] = "1"
tokenizer = AutoTokenizer.from_pretrained(TOKENIZER_DIR, use_fast=True, local_files_only=True)

app = FastAPI()

class InferRequest(BaseModel):
    inputs: Union[str, List[str]]

def softmax(x):
    x = np.array(x)
    x = x - x.max(axis=-1, keepdims=True)
    e = np.exp(x)
    return e / e.sum(axis=-1, keepdims=True)

@app.post("/infer")
def infer(req: InferRequest):
    texts = [req.inputs] if isinstance(req.inputs, str) else req.inputs
    enc = tokenizer(texts, padding=True, truncation=True, max_length=512, return_tensors="np")
    inputs = {k: v for k, v in enc.items() if k in [i.name for i in session.get_inputs()]}
    outputs = session.run(None, inputs)
    logits = outputs[0]
    probs = softmax(logits)
    idx = probs.argmax(-1)
    # Example: adjust your label mapping here if needed
    result = []
    for i, p in zip(idx, probs):
        label = "PROMPT_INJECTION" if i == 1 else "SAFE"
        score = float(p[i])
        result.append({"label": label, "score": score})
    return result
PY

# Use uvicorn with workers for throughput
uvicorn app:app --host 127.0.0.1 --port $Port --workers $Workers

