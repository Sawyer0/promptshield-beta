package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	// Default tenant ID - in real implementation this would come from auth context
	defaultTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	r.Route("/rulepacks", func(rr chi.Router) {
		rr.Get("/", func(w http.ResponseWriter, r *http.Request) {
			rulepacks, err := opt.RulepackService.List(r.Context(), defaultTenantID)
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

			_ = json.NewEncoder(w).Encode(result)
		})
		rr.Get("/active", func(w http.ResponseWriter, r *http.Request) {
			// Find the active rulepack from the list
			rulepacks, err := opt.RulepackService.List(r.Context(), defaultTenantID)
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
					_ = json.NewEncoder(w).Encode(result)
					return
				}
			}

			// No active rulepack found
			_ = json.NewEncoder(w).Encode(RulepackMeta{})
		})
		rr.Group(func(a chi.Router) {
			a.Use(adminAuth(opt))
			a.Post("/validate", func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				defer r.Body.Close()
				valid, warnings, errors := opt.RulepackService.ValidateDSL(body)
				_ = json.NewEncoder(w).Encode(map[string]any{"valid": valid, "warnings": warnings, "errors": errors})
			})
			a.Post("/", func(w http.ResponseWriter, r *http.Request) {
				// Idempotency handling: if the caller supplied an Idempotency-Key
				// header and we have a cached response, short-circuit and return
				// the prior result.
				idemKey := r.Header.Get("Idempotency-Key")
				if idemKey != "" {
					if v, ok := idempotencyStore.Load(idemKey); ok {
						// Return cached response with informative header.
						w.Header().Set("X-Idempotency-Cache", "HIT")
						w.Header().Set("Content-Type", "application/json")
						// Use 200 OK for subsequent identical requests.
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(v)
						return
					}
				}
				activate := r.URL.Query().Get("activate") == "true"
				ct := r.Header.Get("content-type")
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
					data, _ = io.ReadAll(f)
				} else {
					// raw body (application/x-yaml)
					body, _ := io.ReadAll(r.Body)
					defer r.Body.Close()
					data = body
				}

				// Parse and validate the DSL
				pack, err := opt.RulepackService.ParseDSL(data)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
					return
				}

				// Create the rulepack
				packID, err := opt.RulepackService.Create(r.Context(), defaultTenantID, pack.Metadata.Name, pack.Metadata.Description)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				// Upload version 1
				version := 1
				_, err = opt.RulepackService.Upload(r.Context(), defaultTenantID, packID, version, json.RawMessage(data), activate)
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

				// Persist successful result in the idempotency cache *before*
				// writing the response to ensure subsequent retries are served.
				if idemKey != "" {
					idempotencyStore.Store(idemKey, meta)
				}

				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(meta)
			})
			a.Post("/reload", func(w http.ResponseWriter, r *http.Request) {
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

				// Create the rulepack
				defaultTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
				packID, err := opt.RulepackService.Create(r.Context(), defaultTenantID, pack.Metadata.Name, pack.Metadata.Description)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				// Upload version 1 and activate
				version := 1
				_, err = opt.RulepackService.Upload(r.Context(), defaultTenantID, packID, version, json.RawMessage(data), true)
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

				_ = json.NewEncoder(w).Encode(meta)
			})
			a.Put("/active", func(w http.ResponseWriter, r *http.Request) {
				// Optimistic locking via If-Match / ETag. Require the caller to
				// supply an If-Match header containing the current active
				// version number (quoted). Reject when it doesn’t match.
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

				// Fetch the currently active version to compute the expected ETag.
				curDSL, curVer, _ := opt.RulepackService.GetActive(r.Context(), packID)
				_ = curDSL // content not needed; only version number matters

				expectedETag := fmt.Sprintf("\"%d\"", curVer)

				// Enforce optimistic lock if caller provided If-Match.
				if ifMatch == "" {
					writeError(w, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match header required", nil)
					return
				}
				if ifMatch != "*" && ifMatch != expectedETag {
					writeError(w, http.StatusPreconditionFailed, "PRECONDITION_FAILED", "ETag mismatch", map[string]any{"expected": expectedETag, "got": ifMatch})
					return
				}

				// Activate the latest version of this rulepack.
				if err := opt.RulepackService.ActivateLatest(r.Context(), defaultTenantID, packID); err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				// Return the updated active rulepack
				rulepacks, err := opt.RulepackService.List(r.Context(), defaultTenantID)
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
						// Emit new ETag reflecting the updated version.
						w.Header().Set("ETag", fmt.Sprintf("\"%s\"", result.Version))
						_ = json.NewEncoder(w).Encode(result)
						return
					}
				}

				writeError(w, http.StatusNotFound, "NOT_FOUND", "rulepack not found", nil)
			})
			a.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
				id := chi.URLParam(r, "id")

				packID, err := uuid.Parse(id)
				if err != nil {
					writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid id format", nil)
					return
				}

				if err := opt.RulepackService.Delete(r.Context(), defaultTenantID, packID); err != nil {
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			})
		})
	})
}
