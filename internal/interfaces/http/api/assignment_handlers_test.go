package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	services "github.com/promptshield/promptshield/internal/application/services"
	memrepo "github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
	repository "github.com/promptshield/promptshield/internal/repository"
)

func TestCreateAssignment_Validation_BadMethod(t *testing.T) {
	mux := NewMux(Options{RulepackService: services.RulepackServiceCstor(memrepo.NewRulepackRepository(), nil), AssignmentRepository: repository.NewMockRulepackAssignmentRepository()})
	tenantID := uuid.New().String()
	body := map[string]any{
		"rulepack_id":  uuid.New().String(),
		"target_scope": "/api/*",
		"method":       "BAD",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/tenants/"+tenantID+"/assignments", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-User-Admin", "true")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid method, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateAssignment_Validation_BadScope(t *testing.T) {
	mux := NewMux(Options{RulepackService: services.RulepackServiceCstor(memrepo.NewRulepackRepository(), nil), AssignmentRepository: repository.NewMockRulepackAssignmentRepository()})
	tenantID := uuid.New().String()
	body := map[string]any{
		"rulepack_id":  uuid.New().String(),
		"target_scope": "api/no-leading-slash",
		"method":       "GET",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/tenants/"+tenantID+"/assignments", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-User-Admin", "true")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid scope, got %d body=%s", rec.Code, rec.Body.String())
	}
}
