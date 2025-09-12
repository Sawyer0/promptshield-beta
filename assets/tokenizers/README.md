# Tokenizer assets for DeBERTa (exact token accounting)

Place your tokenizer files here to enable precise token counting in PromptShield.

Recommended: tokenizer.json (HuggingFace format)
- Path (example): C:\Users\Dawan\promptshield-v0.2.0\assets\tokenizers\tokenizer.json
- Set environment variable:
  - PS_DEBERTA_TOKENIZER_JSON=C:\Users\Dawan\promptshield-v0.2.0\assets\tokenizers\tokenizer.json
- Optional casing flag (default true):
  - PS_DEBERTA_LOWERCASE=true|false

Alternative: vocab.txt (one token per line)
- Path (example): C:\Users\Dawan\promptshield-v0.2.0\assets\tokenizers\deberta-vocab.txt
- Set environment variable:
  - PS_DEBERTA_VOCAB_FILE=C:\Users\Dawan\promptshield-v0.2.0\assets\tokenizers\deberta-vocab.txt
- Optional casing flag:
  - PS_DEBERTA_LOWERCASE=true|false

Behavior in PromptShield
- If PS_DEBERTA_TOKENIZER_JSON is set, it is used first.
- Otherwise PS_DEBERTA_VOCAB_FILE is used when provided.
- If neither is set and no fallback file is present, PromptShield falls back to an approximation for token counting.

Air-gapped deployment notes
- Keep tokenizer files local and versioned with your model.
- Do not include secrets or API keys. Tokenizer files are non-secret artifacts.

