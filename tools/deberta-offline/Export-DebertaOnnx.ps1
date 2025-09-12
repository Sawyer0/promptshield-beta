param(
    [Parameter(Mandatory=$true)][string]$ModelDir,
    [Parameter(Mandatory=$true)][string]$OutputDir,
    [switch]$Quantize
)

$ErrorActionPreference = "Stop"

python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install --upgrade pip
pip install transformers optimum[onnxruntime] onnx onnxruntime onnxruntime-tools

Write-Host "Exporting model to ONNX..."
python - << 'PY'
import sys, os
from optimum.exporters.onnx import main as onnx_export
# optimum-cli export onnx --model "$env:ModelDir" --task text-classification "$env:OutputDir"
args = [
    "export", "onnx",
    "--model", os.environ["ModelDir"],
    "--task", "text-classification",
    os.environ["OutputDir"],
]
onnx_export.main(args)
PY

if ($Quantize) {
    Write-Host "Quantizing to INT8..."
    python - << 'PY'
import os
from onnxruntime.quantization import quantize_dynamic, QuantType
inp = os.path.join(os.environ["OutputDir"], "model.onnx")
outp = os.path.join(os.environ["OutputDir"], "model-int8.onnx")
quantize_dynamic(model_input=inp, model_output=outp, weight_type=QuantType.QInt8)
print("Quantized model saved:", outp)
PY
}

Write-Host "Done. ONNX exported to" $OutputDir

