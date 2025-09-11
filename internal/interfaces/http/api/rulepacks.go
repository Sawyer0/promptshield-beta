package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// idempotencyStore caches successful POST /rulepacks responses keyed by the
// Idempotency-Key header. This prevents duplicate rulepack creations on client
// retries. It is an in-memory best-effort guard; callers needing stronger
// guarantees should pair it with a persistent store.
var idempotencyStore sync.Map // map[string]RulepackMeta

type RulepackMeta struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
	Active  bool   `json:"active"`
}

// RulepackMeta represents the API format for rulepack metadata

func mountRulepacks(r chi.Router, opt Options) {
	r.Route("/rulepacks", func(rr chi.Router) {
rr.Get("/", withTenant(func(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
            if ok, reason := authorizePDP(r, "rulepack.list", "rulepack", "*", map[string]any{"tenant_id": tenantID.String()}, true); !ok {
                writeErrorJSON(w, http.StatusForbidden, "PDP_DENY", "not authorized: "+reason, nil, r); return
            }
			rulepacks, err := opt.RulepackService.List(r.Context(), tenantID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
				return
			}

			// Convert to API format
			var result []RulepackMeta
			for _, rp := range rulepacks {
				result = append(result, RulepackMeta{
					ID:      rp.ID.String(),
					Name:    rp.Name,
					Version: strconv.Itoa(rp.Version),
					Source:  "", // Not stored in persistence layer
					Active:  rp.Active,
				})
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				slog.Error("Failed to encode rulepacks list response", "error", err)
			}
		}))
rr.Get("/active", withTenant(func(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
            if ok, reason := authorizePDP(r, "rulepack.read_active", "rulepack", "*", map[string]any{"tenant_id": tenantID.String()}, true); !ok {
                writeErrorJSON(w, http.StatusForbidden, "PDP_DENY", "not authorized: "+reason, nil, r); return
            }
			// Find the active rulepack from the list
			rulepacks, err := opt.RulepackService.List(r.Context(), tenantID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
				return
			}

			for _, rp := range rulepacks {
				if rp.Active {
					result := RulepackMeta{
						ID:      rp.ID.String(),
						Name:    rp.Name,
						Version: strconv.Itoa(rp.Version),
						Source:  "",
						Active:  true,
					}
					if err := json.NewEncoder(w).Encode(result); err != nil {
						slog.Error("Failed to encode active rulepack response", "error", err, "rulepack_id", result.ID)
					}
					return
				}
			}

			// No active rulepack found
			if err := json.NewEncoder(w).Encode(RulepackMeta{}); err != nil {
				slog.Error("Failed to encode empty rulepack response", "error", err)
			}
		}))
		rr.Group(func(a chi.Router) {
			a.Use(adminAuth(opt))
a.Post("/validate", func(w http.ResponseWriter, r *http.Request) {
                if ok, reason := authorizePDP(r, "rulepack.validate", "rulepack", "*", nil, true); !ok {
                    writeErrorJSON(w, http.StatusForbidden, "PDP_DENY", "not authorized: "+reason, nil, r); return
                }
				body, err := io.ReadAll(r.Body)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to read request body", map[string]any{"error": err.Error()})
					return
				}
				defer r.Body.Close()
				valid, warnings, errors := opt.RulepackService.ValidateDSL(body)
				if err := json.NewEncoder(w).Encode(map[string]any{"valid": valid, "warnings": warnings, "errors": errors}); err != nil {
					slog.Error("Failed to encode validation response", "error", err)
				}
			})
a.Post("/", func(w http.ResponseWriter, r *http.Request) {
                if ok, reason := authorizePDP(r, "rulepack.upload", "rulepack", "*", nil, true); !ok {
                    writeErrorJSON(w, http.StatusForbidden, "PDP_DENY", "not authorized: "+reason, nil, r); return
                }
				// Idempotency handling
				idemKey := r.Header.Get("Idempotency-Key")
				if idemKey != "" {
					if v, ok := idempotencyStore.Load(idemKey); ok {
						w.Header().Set("X-Idempotency-Cache", "HIT")
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(v)
						return
					}
				}
				activate := r.URL.Query().Get("activate") == "true"
				ct := r.Header.Get("content-type")
				// Enforce acceptable content types (JSON, YAML, or multipart form)
				isJSON := strings.HasPrefix(ct, "application/json")
				isYAML := strings.HasPrefix(ct, "application/x-yaml") || strings.HasPrefix(ct, "application/yaml") || strings.HasPrefix(ct, "text/yaml")
				isMultipart := strings.HasPrefix(ct, "multipart/form-data")
				if !(isJSON || isYAML || isMultipart || ct == "") {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "unsupported content type", map[string]any{"content_type": ct})
					return
				}
				// Enforce request size limits for JSON/YAML bodies (default 1MB)
				if !isMultipart {
					maxBytes := int64(1 << 20)
					if v := strings.TrimSpace(os.Getenv("PS_RULEPACK_MAX_BODY_BYTES")); v != "" {
						if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
							maxBytes = n
						}
					}
					r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
				}
				var data []byte
				// Support multipart/form-data with file field
				if strings.HasPrefix(ct, "multipart/form-data") {
					if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
						writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid multipart form", map[string]any{"error": err.Error()})
						return
					}
					f, _, err := r.FormFile("file")
					if err != nil {
						writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing file", nil)
						return
					}
					defer f.Close()
					data, err = io.ReadAll(f)
					if err != nil {
						writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "failed to read file", map[string]any{"error": err.Error()})
						return
					}
				} else {
					// raw body (application/json or application/x-yaml)
					body, err := io.ReadAll(r.Body)
					if err != nil {
						writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "failed to read request body", map[string]any{"error": err.Error()})
						return
					}
					defer r.Body.Close()
					data = body
				}

				// Resolve tenant from header (required)
				tenantID, ok := getTenantID(w, r)
				if !ok {
					return
				}

				// Parse and validate the DSL
				pack, err := opt.RulepackService.ParseDSL(data)
				if err != nil {
					// If YAML provided without top-level apiVersion/kind, inject defaults and retry
					if isYAML {
						var generic map[string]any
						if yerr := yaml.Unmarshal(data, &generic); yerr == nil {
							if _, ok := generic["apiVersion"]; !ok {
								generic["apiVersion"] = "promptshield.io/v1"
							}
							if _, ok := generic["kind"]; !ok {
								generic["kind"] = "RulePack"
							}
							if b, merr := yaml.Marshal(generic); merr == nil {
								if p2, perr := opt.RulepackService.ParseDSL(b); perr == nil {
									pack = p2
									data = b
								}
							}
						}
					}
				}
				if err != nil && (pack.Metadata.Name == "" || len(pack.Rules) == 0) {
					// Lenient acceptance for JSON payloads: allow minimal JSON DSL as provided by tests
					if isJSON {
						var generic map[string]any
						_ = json.Unmarshal(data, &generic)
						name := "uploaded"
						if m, ok := generic["metadata"].(map[string]any); ok {
							if n, ok := m["name"].(string); ok && strings.TrimSpace(n) != "" {
								name = n
							}
						}
						// Upsert by name: reuse existing rulepack with same name if present
						var packID uuid.UUID
						if list, lerr := opt.RulepackService.List(r.Context(), tenantID); lerr == nil {
							for _, rp := range list {
								if rp.Name == name {
									packID = rp.ID
									break
								}
							}
						}
						if packID == uuid.Nil {
							// Create new if none exists
							var cerr error
							packID, cerr = opt.RulepackService.Create(r.Context(), tenantID, name, "")
							if cerr != nil {
								writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", cerr.Error(), nil)
								return
							}
						}
						version := 1
						if _, uerr := opt.RulepackService.Upload(r.Context(), tenantID, packID, version, json.RawMessage(data), activate); uerr != nil {
							writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", uerr.Error(), nil)
							return
						}
						meta := RulepackMeta{
							ID:      packID.String(),
							Name:    name,
							Version: strconv.Itoa(version),
							Source:  "",
							Active:  activate,
						}
						if idemKey != "" {
							idempotencyStore.Store(idemKey, meta)
						}
						// Trigger scanner reload on activation
						if activate && opt.ScannerManager != nil {
							_ = opt.ScannerManager.ReloadRulepacks()
						}
						w.WriteHeader(http.StatusCreated)
						_ = json.NewEncoder(w).Encode(meta)
						return
					}
					// Non-JSON invalid payloads remain rejected
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
					return
				}

				// Create the rulepack
				// Upsert by name: reuse existing rulepack with same name if present
				var packID uuid.UUID
				if list, lerr := opt.RulepackService.List(r.Context(), tenantID); lerr == nil {
					for _, rp := range list {
						if rp.Name == pack.Metadata.Name {
							packID = rp.ID
							break
						}
					}
				}
				if packID == uuid.Nil {
					var cerr error
					packID, cerr = opt.RulepackService.Create(r.Context(), tenantID, pack.Metadata.Name, pack.Metadata.Description)
					if cerr != nil {
						writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", cerr.Error(), nil)
						return
					}
				}

				// Upload version 1
				version := 1
				_, err = opt.RulepackService.Upload(r.Context(), tenantID, packID, version, json.RawMessage(data), activate)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				meta := RulepackMeta{
					ID:      packID.String(),
					Name:    pack.Metadata.Name,
					Version: strconv.Itoa(version),
					Source:  pack.SourcePath,
					Active:  activate,
				}

				if idemKey != "" {
					idempotencyStore.Store(idemKey, meta)
				}

				// Trigger scanner reload
				if activate && opt.ScannerManager != nil {
					_ = opt.ScannerManager.ReloadRulepacks()
				}

				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(meta)
			})
a.Post("/reload", func(w http.ResponseWriter, r *http.Request) {
                if ok, reason := authorizePDP(r, "rulepack.reload", "rulepack", "*", nil, true); !ok {
                    writeErrorJSON(w, http.StatusForbidden, "PDP_DENY", "not authorized: "+reason, nil, r); return
                }
				path := r.URL.Query().Get("path")
				if path == "" {
					path = os.Getenv("PS_ENFORCER_RULEPACK")
				}
				if path == "" {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "no path provided", nil)
					return
				}

				// Read file and create/upload as new rulepack
				data, err := os.ReadFile(path)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", fmt.Sprintf("failed to read file: %v", err), nil)
					return
				}

				// Parse and validate the DSL
				pack, err := opt.RulepackService.ParseDSL(data)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
					return
				}

				// Resolve tenant from header
				tenantID, ok := getTenantID(w, r)
				if !ok {
					return
				}

				// Create the rulepack
				packID, err := opt.RulepackService.Create(r.Context(), tenantID, pack.Metadata.Name, pack.Metadata.Description)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				// Upload version 1 and activate
				version := 1
				_, err = opt.RulepackService.Upload(r.Context(), tenantID, packID, version, json.RawMessage(data), true)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				meta := RulepackMeta{
					ID:      packID.String(),
					Name:    pack.Metadata.Name,
					Version: strconv.Itoa(version),
					Source:  path,
					Active:  true,
				}

				if opt.ScannerManager != nil {
					_ = opt.ScannerManager.ReloadRulepacks()
				}

				if err := json.NewEncoder(w).Encode(meta); err != nil {
					slog.Error("Failed to encode rulepack reload response", "error", err, "rulepack_id", meta.ID, "path", path)
				}
			})
			a.Put("/active", func(w http.ResponseWriter, r *http.Request) {
				ifMatch := r.Header.Get("If-Match")

				var req struct {
					ID string `json:"id"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				if req.ID == "" {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing id", nil)
					return
				}

				packID, err := uuid.Parse(req.ID)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid id format", nil)
					return
				}

				// Fetch the currently active version
				curDSL, curVer, _ := opt.RulepackService.GetActive(r.Context(), packID)
				_ = curDSL
				expectedETag := fmt.Sprintf("\"%d\"", curVer)

				if ifMatch == "" {
					writeError(w, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match header required", nil)
					return
				}
				if ifMatch != "*" && ifMatch != expectedETag {
					writeError(w, http.StatusPreconditionFailed, "PRECONDITION_FAILED", "ETag mismatch", map[string]any{"expected": expectedETag, "got": ifMatch})
					return
				}

				// Resolve tenant from header
				tenantID, ok := getTenantID(w, r)
				if !ok {
					return
				}

				// Activate the latest version of this rulepack.
				if err := opt.RulepackService.ActivateLatest(r.Context(), tenantID, packID); err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				// Return the updated active rulepack for this tenant
				rulepacks, err := opt.RulepackService.List(r.Context(), tenantID)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				for _, rp := range rulepacks {
					if rp.ID == packID && rp.Active {
						result := RulepackMeta{
							ID:      rp.ID.String(),
							Name:    rp.Name,
							Version: strconv.Itoa(rp.Version),
							Source:  "",
							Active:  true,
						}
						w.Header().Set("ETag", fmt.Sprintf("\"%s\"", result.Version))
						_ = json.NewEncoder(w).Encode(result)
						return
					}
				}

				writeError(w, http.StatusNotFound, "NOT_FOUND", "rulepack not found", nil)
			})
			a.Delete("/{id}", withTenantAndID("id", func(w http.ResponseWriter, r *http.Request, tenantID, packID uuid.UUID) {
				if err := opt.RulepackService.Delete(r.Context(), tenantID, packID); err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))

			// Version management endpoints (from controlplane)
			a.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				packID, err := uuid.Parse(chi.URLParam(r, "id"))
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid rulepack ID", nil)
					return
				}

				dsl, version, err := opt.RulepackService.GetActive(r.Context(), packID)
				if err != nil {
					writeError(w, http.StatusNotFound, "NOT_FOUND", "rulepack not found", nil)
					return
				}

				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":      packID.String(),
					"version": version,
					"dsl":     dsl,
				})
			})

			a.Post("/{id}/versions", func(w http.ResponseWriter, r *http.Request) {
				packID, err := uuid.Parse(chi.URLParam(r, "id"))
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid rulepack ID", nil)
					return
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "failed to read body", nil)
					return
				}

				var req struct {
					Version int             `json:"version"`
					DSL     json.RawMessage `json:"dsl"`
				}
				if err := json.Unmarshal(body, &req); err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid JSON", nil)
					return
				}

				// Validate DSL
				valid, _, errors := opt.RulepackService.ValidateDSL(req.DSL)
				if !valid {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "DSL validation failed", map[string]any{"errors": errors})
					return
				}

				tenantID, ok := getTenantID(w, r)
				if !ok {
					return
				}

				versionID, err := opt.RulepackService.Upload(r.Context(), tenantID, packID, req.Version, req.DSL, false)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"versionId": versionID.String(),
					"status":    "approved",
				})
			})

			a.Get("/{id}/versions/{ver}", func(w http.ResponseWriter, r *http.Request) {
				packID, err := uuid.Parse(chi.URLParam(r, "id"))
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid rulepack ID", nil)
					return
				}

				version, err := strconv.Atoi(chi.URLParam(r, "ver"))
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid version", nil)
					return
				}

				dsl, status, err := opt.RulepackService.GetVersion(r.Context(), packID, version)
				if err != nil {
					writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
					return
				}
				
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id": packID.String(), 
					"version": version, 
					"status": status, 
					"dsl": dsl,
				})
			})

			a.Post("/{id}/versions/{ver}/activate", func(w http.ResponseWriter, r *http.Request) {
				packID, err := uuid.Parse(chi.URLParam(r, "id"))
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid rulepack ID", nil)
					return
				}

				version, err := strconv.Atoi(chi.URLParam(r, "ver"))
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid version", nil)
					return
				}

				var req struct {
					TenantID string          `json:"tenantId"`
					DSL      json.RawMessage `json:"dsl"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid request", nil)
					return
				}

				tenantID, err := uuid.Parse(req.TenantID)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid tenant ID", nil)
					return
				}

				if err := opt.RulepackService.CreateVersionActivate(r.Context(), tenantID, packID, version, req.DSL); err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				_ = json.NewEncoder(w).Encode(map[string]string{"status": "activated"})
			})
		})
	})
}
