# PromptShield Frontend Requirements Document

This document outlines the API endpoints and expected frontend pages for the PromptShield LLM security gateway dashboard.

## Application Overview

PromptShield provides real-time LLM content filtering through dynamic policy management. The frontend should provide security teams with policy management and real-time testing capabilities.

**Base URL**: `http://127.0.0.1:9090`  
**Authentication**: Bearer token (`Authorization: Bearer {token}`)

## API Endpoints

### System Health
- `GET /healthz` - Liveness check (200 OK if healthy)
- `GET /readyz` - Readiness check (200 OK if ready)
- `GET /metrics` - Prometheus metrics endpoint

### Policy Management
- `GET /v1/admin/policies` - List all policies
- `POST /v1/admin/policies` - Create new policy
- `GET /v1/admin/policies/{id}` - Get specific policy
- `PUT /v1/admin/policies/{id}` - Update policy
- `DELETE /v1/admin/policies/{id}` - Delete policy
- `POST /v1/admin/policies/{id}/activate` - Activate policy for real-time enforcement
- `POST /v1/admin/policies/{id}/deactivate` - Deactivate policy

### RulePack Management  
- `GET /v1/admin/rulepacks` - List rulepacks
- `POST /v1/admin/rulepacks` - Create rulepack
- `GET /v1/admin/rulepacks/{id}` - Get rulepack

### Real-Time Enforcement
- `POST /v1/check` - Test content against active policies (Content-Type: text/plain)

## Expected Pages

### 1. Dashboard (Home)
**Route**: `/`

**Purpose**: Overview of system status and key metrics

**Components**:
- System health indicators (health/readiness checks)
- Policy activation summary (X active policies)
- Real-time enforcement testing section
- Recent policy activity log

**Key Features**:
- Quick content testing input with immediate results
- Visual status indicators for system health
- One-click policy activation/deactivation toggles

### 2. Policy Management
**Route**: `/policies`

**Purpose**: Full CRUD management of security policies

**Components**:
- Policy list with search/filter capabilities
- Policy creation/editing forms
- Policy activation status indicators
- Bulk operations (activate/deactivate multiple)

**Key Features**:
- YAML editor for RulePack content with syntax highlighting
- Policy preview/validation before saving
- Import/export policy functionality
- Policy versioning display

### 3. Policy Creator/Editor
**Route**: `/policies/new` and `/policies/{id}/edit`

**Purpose**: Create or modify policy RulePacks

**Components**:
- Policy metadata form (name, type, description)
- YAML editor with RulePack schema
- Rule builder UI (optional - for non-technical users)
- Preview and validation section

**Key Features**:
- Template selection for common policy types
- Real-time YAML validation
- Rule testing against sample content
- Save as draft functionality

### 4. Real-Time Testing
**Route**: `/test`

**Purpose**: Test content against active policies

**Components**:
- Content input area (text/file upload)
- Test results display
- Policy execution trace
- Historical test results

**Key Features**:
- Batch testing capabilities
- Result export (CSV/JSON)
- Performance metrics (response time, throughput)
- Different content type testing (text, JSON, etc.)

### 5. RulePack Library
**Route**: `/rulepacks`

**Purpose**: Manage reusable RulePack templates

**Components**:
- RulePack gallery with categories
- RulePack details/documentation
- Import from external sources
- Community/shared RulePacks (future)

**Key Features**:
- RulePack composition (combining multiple packs)
- Dependency management
- Version control and rollback
- Performance impact analysis

### 6. Monitoring & Analytics
**Route**: `/monitoring`

**Purpose**: System metrics and enforcement analytics

**Components**:
- Real-time metrics dashboard
- Enforcement decision trends
- Policy effectiveness analytics
- System performance graphs

**Key Features**:
- Configurable time ranges
- Alert thresholds and notifications
- Violation pattern analysis
- Export capabilities for reports

## Data Models

### Policy Object
```typescript
interface Policy {
  id: string;
  name: string;
  type: "security" | "compliance" | "custom";
  content: string; // YAML RulePack definition
  version: number;
  created_at: string;
  updated_at: string;
  created_by?: string;
  is_active?: boolean;
}
```

### Enforcement Result
```typescript
interface EnforcementResult {
  decision: "allow" | "quarantine" | "deny";
  reason: string;
  violations: number;
  request_id: string;
  processing_time_ms?: number;
}
```

### RulePack Structure
```yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: string
  version: string
  description?: string
rules:
  - id: string
    name: string
    level: 1 | 2 | 3  # keyword | regex | semantic
    severity: "LOW" | "MEDIUM" | "HIGH" | "CRITICAL"
    keywords?: string[]
    patterns?: 
      - regex: string
    semantic?:
      model: string
      analysis_prompt: string
    response:
      action: "allow" | "warn" | "quarantine" | "deny"
      message: string
```

## Technical Requirements

### State Management
- Policy list state with real-time updates
- Active policy tracking
- Test result history
- System health status

### Real-Time Features
- Policy activation should immediately affect `/v1/check` responses
- WebSocket/SSE for live policy status updates (if implemented)
- Real-time test result streaming for large content

### Error Handling
- API timeout handling (2-second enforcement timeout)
- Authentication failures (401)
- Policy validation errors
- Network connectivity issues

### Performance Considerations
- Lazy loading for large policy lists  
- Debounced search/filter inputs
- Caching for frequently accessed policies
- Optimistic UI updates for policy activation

### Security
- Secure token storage and rotation
- Input sanitization for policy content
- CSP headers for XSS protection
- Audit logging for admin actions

## User Experience Notes

### Key Workflows
1. **Quick Test**: Homepage → Enter content → See immediate result
2. **Policy Creation**: Policies → New → Template → Edit → Save → Activate
3. **Emergency Response**: Dashboard → Deactivate problematic policy → Verify with test

### Visual Hierarchy
- **Green**: Allowed content, healthy systems, active policies
- **Yellow**: Quarantined content, warnings, pending states  
- **Red**: Denied content, errors, inactive critical policies
- **Blue**: Information, navigation, primary actions

### Responsive Design
- Mobile-friendly policy management
- Tablet-optimized YAML editing
- Desktop-focused monitoring dashboards

This specification assumes standard web development knowledge and focuses on PromptShield-specific requirements and data structures.