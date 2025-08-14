### Install

Prebuilt binaries (recommended):
- Visit Releases page and download the archive for your OS/arch
- Verify checksum and signature (when available), then place the binary on your PATH

Homebrew (macOS/Linux):
```
brew tap promptshield/tap
brew install promptshield
```

Scoop (Windows):
```
scoop bucket add promptshield https://github.com/promptshield/scoop-bucket
scoop install promptshield
```

Curl‑to‑bash (macOS/Linux):
```
curl -sSL https://get.promptshield.io/install.sh | bash
```

Shell completion:
```
promptshield completion bash > /etc/bash_completion.d/promptshield
promptshield completion zsh  > ~/.zsh/completions/_promptshield
promptshield completion fish > ~/.config/fish/completions/promptshield.fish
promptshield completion powershell | Out-String | Set-Content -Path $PROFILE -Encoding utf8 -Append
```


