# PromptShield Version Matrix

| PromptShield | Helm Chart | Terraform Module | Enforcer Image Tags | Envoy Tested | Kubernetes | Notes |
|--------------|-----------|------------------|---------------------|--------------|------------|-------|
| v0.2.0       | 0.2.0     | 0.2.0            | `0.2.0` (amd64, arm64) | 1.26 / 1.27 / 1.28 | 1.26+ | Signed with Cosign; SBOM (CycloneDX) published |

## Image Signing & SBOM

* Multi-arch images built via GitHub Actions.
* **Cosign** used for keyless signing:
  ```bash
  COSIGN_EXPERIMENTAL=1 cosign verify ghcr.io/promptshield/enforcer:0.2.0
  ```
* **Syft** emits CycloneDX SBOM (`enforcer_0.2.0.sbom.cdx.json`) uploaded as release asset.

CI snippet (simplified):
```yaml
- name: Build & Push
  uses: docker/build-push-action@v5
  with:
    push: true
    platforms: linux/amd64,linux/arm64
    tags: ghcr.io/promptshield/enforcer:${{ env.VERSION }}
- name: Generate SBOM
  run: syft –o cyclonedx-json ghcr.io/promptshield/enforcer:${{ env.VERSION }} > sbom.json
- name: Sign image
  env:
    COSIGN_YES: "true"
  run: cosign sign --key cosign.key ghcr.io/promptshield/enforcer:${{ env.VERSION }}
```
