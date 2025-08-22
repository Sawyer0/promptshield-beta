# PromptShield Dashboard Design

## Executive Summary

A minimal, modern, enterprise-grade dashboard for PromptShield that provides security teams with immediate situational awareness and actionable insights for LLM threat protection.

## Design Philosophy

- **Minimal**: Only essential information on the main dashboard
- **Modern**: Clean aesthetics with smooth interactions
- **Enterprise-Grade**: Professional, reliable, and scalable
- **Data-Rich**: Maximum insight with minimum cognitive load

## Dashboard Layout

### 1. Header Section
```
┌─────────────────────────────────────────────────────────────────────────┐
│ 🛡️ PromptShield  |  System Status: ● Operational  |  Quick Test  |  ⚙️  │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2. Key Metrics Cards
Four primary metrics displayed prominently at the top:

```
┌─────────────────────┬─────────────────────┬─────────────────────┬─────────────────────┐
│  THREATS BLOCKED    │  ACTIVE POLICIES    │  API LATENCY        │  SYSTEM HEALTH      │
│  24.3K              │  12 Active          │  <1ms P50           │  99.98%             │
│  ↑ 15% (24h)        │  3 Modified Today   │  <10ms P95          │  All Systems Go     │
└─────────────────────┴─────────────────────┴─────────────────────┴─────────────────────┘
```

**Metrics Explained:**
- **Threats Blocked**: Total threats prevented with 24h trend
- **Active Policies**: Currently enforced policies with recent changes
- **API Latency**: P50 and P95 response times
- **System Health**: Overall system availability

### 3. Real-Time Activity Feed (Left Panel - 60% Width)

Live streaming threat detection with contextual information:

```
┌─────────────────────────────────────────────────────────────────┐
│ LIVE THREAT DETECTION                              [Auto-refresh]│
├─────────────────────────────────────────────────────────────────┤
│ 🔴 HIGH   | 14:32:01 | Prompt Injection | Policy: PI-001       │
│           | "Ignore previous instructions and..."               │
│           | Source: API Gateway | User: anonymous                │
│                                                                 │
│ 🟡 MEDIUM | 14:31:45 | PII Detection | Policy: PII-REDACT      │
│           | SSN pattern detected and redacted                   │
│           | Source: Chat Interface | User: user_1234            │
│                                                                 │
│ 🔴 HIGH   | 14:31:22 | Jailbreak Attempt | Policy: JB-BLOCK   │
│           | Semantic analysis flagged manipulation              │
│           | Source: Production API | User: app_service          │
└─────────────────────────────────────────────────────────────────┘
```

**Feed Features:**
- Color-coded severity (🔴 High, 🟡 Medium, 🟢 Low)
- Timestamp with millisecond precision
- Threat category and triggering policy
- Preview of blocked content (truncated)
- Source and user attribution

### 4. Performance Insights (Right Panel - 40% Width)

Real-time scanning performance by detection level:

```
┌─────────────────────────────────────────────────────────────────┐
│ SCANNING PERFORMANCE                                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Level 1 (Keywords)    ████████████████░░░░  80% (<1ms avg)    │
│                       12,432 scans/min                         │
│                                                                 │
│ Level 2 (Regex)       ██████████░░░░░░░░░░  50% (<10ms avg)   │
│                       6,123 scans/min                          │
│                                                                 │
│ Level 3 (Semantic)    ████░░░░░░░░░░░░░░░░  20% (<100ms avg)  │
│                       1,245 scans/min                          │
│                                                                 │
│ ─────────────────────────────────────────                      │
│ Cache Hit Rate: 78% ▲                                          │
│ Queue Depth: 12 requests                                       │
│ Token Usage: 45K/100K (45%)                                    │
└─────────────────────────────────────────────────────────────────┘
```

### 5. Policy Effectiveness Chart (Bottom Left - 60% Width)

Visual representation of most triggered policies:

```
┌─────────────────────────────────────────────────────────────────┐
│ TOP TRIGGERED POLICIES (Last 7 Days)                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ prompt-injection     ████████████████████████  2,341 (32%)    │
│ pii-detection       ███████████████         1,892 (26%)       │
│ jailbreak-attempts  ██████████             1,023 (14%)        │
│ hate-speech         ████████                 876 (12%)        │
│ data-exfiltration   ███                      234 (3%)         │
│ hallucination       ██                       187 (2%)         │
│ Other               ████████                 823 (11%)        │
│                                                                 │
│ Total Blocks: 7,376              [View All Policies →]         │
└─────────────────────────────────────────────────────────────────┘
```

### 6. Quick Test Widget (Bottom Right - 40% Width)

Instant policy testing interface:

```
┌─────────────────────────────────────────────────────────────────┐
│ QUICK CONTENT TEST                                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ ┌───────────────────────────────────────────────────┐         │
│ │ Enter text to test against active policies...     │         │
│ │                                                    │         │
│ │                                                    │         │
│ └───────────────────────────────────────────────────┘         │
│                                                                 │
│ Policy Set: [All Active ▼]  [Test Content] [Upload]           │
│                                                                 │
│ Recent Tests:                                                  │
│ ✅ SAFE     | "Hello, how can I help?" (2s ago)              │
│ 🔴 BLOCKED  | "Ignore all instructions" (45s ago)            │
└─────────────────────────────────────────────────────────────────┘
```

## Visual Design Specifications

### Color Palette

| Element | Color | Hex Code | Usage |
|---------|-------|----------|-------|
| Primary | Deep Blue | #1e40af | Headers, primary actions |
| Success | Green | #10b981 | Safe status, positive trends |
| Warning | Amber | #f59e0b | Medium severity, caution states |
| Danger | Red | #ef4444 | High severity, critical alerts |
| Background | Light Gray | #f9fafb | Page background |
| Card Background | White | #ffffff | Content cards |
| Text Primary | Dark Gray | #111827 | Main text |
| Text Secondary | Medium Gray | #6b7280 | Supporting text |

### Typography

- **Headers**: Inter, SF Pro Display, or system font
- **Metrics**: Monospace or tabular numbers for alignment
- **Body Text**: System UI font stack for optimal rendering
- **Font Sizes**:
  - Large metrics: 32px
  - Card headers: 14px (uppercase, letter-spacing)
  - Body text: 14px
  - Small text: 12px

### Spacing System

8px grid system:
- Card padding: 24px
- Element spacing: 16px
- Inline spacing: 8px
- Section margins: 32px

## Interactive Features

### Real-Time Updates
- **WebSocket Connection**: Live threat feed
- **Polling Intervals**:
  - Metrics cards: 5 seconds
  - Performance charts: 10 seconds
  - Policy effectiveness: 30 seconds
  - System health: 10 seconds

### User Interactions
1. **Hover Effects**:
   - Cards elevate with subtle shadow
   - Metrics show detailed tooltips
   - Trend sparklines on hover

2. **Click Actions**:
   - Metric cards → Detailed analytics page
   - Threat items → Full incident report
   - Policy bars → Policy configuration
   - Test results → Detailed analysis

3. **Animations**:
   - Smooth number transitions
   - Fade in/out for feed items
   - Progress bar animations
   - Pulse effect for live indicators

## Data Sources

### API Endpoints
```
GET  /metrics                 # Prometheus metrics
GET  /healthz                 # System health
GET  /readyz                  # Readiness status
GET  /v1/admin/policies       # Active policies
POST /v1/check                # Content testing
WS   /v1/stream/threats       # Live threat feed
```

### Metrics Calculation
- **Threats Blocked**: Sum of all violations with action=block
- **API Latency**: Histogram percentiles from ps_gateway_request_duration
- **System Health**: Composite of health checks and error rates
- **Cache Hit Rate**: cache_hits / (cache_hits + cache_misses)

## Responsive Design

### Desktop (1920px+)
- Full layout as designed
- All panels visible
- Maximum data density

### Laptop (1366px - 1919px)
- Slightly compressed cards
- Feed shows 3-4 items
- Charts remain full

### Tablet (768px - 1365px)
- Two-column layout
- Stack performance under feed
- Horizontal scroll for charts

### Mobile (< 768px)
- Single column
- Collapsible sections
- Essential metrics only
- Simplified feed (last 5)

## Navigation Structure

Each dashboard element links to detailed views:

| Dashboard Element | Links To | Purpose |
|-------------------|----------|---------|
| Threats Blocked | `/analytics/threats` | Historical threat analysis |
| Active Policies | `/policies` | Policy management |
| API Latency | `/analytics/performance` | Performance deep-dive |
| System Health | `/system/status` | System diagnostics |
| Feed Items | `/incidents/{id}` | Full incident details |
| Policy Bars | `/policies/{id}` | Individual policy config |
| Test Results | `/test/results/{id}` | Detailed test analysis |

## Implementation Technologies

### Frontend Stack
- **Framework**: React/Vue.js for reactivity
- **Styling**: Tailwind CSS for utility-first design
- **Charts**: Chart.js or D3.js for visualizations
- **Real-time**: Socket.io for WebSocket management
- **State**: Redux/Vuex for centralized state

### Performance Optimizations
- Virtual scrolling for long feeds
- Debounced API calls
- Memoized calculations
- Lazy loading for charts
- Progressive data loading

## Accessibility Features

- **WCAG 2.1 AA Compliance**
- Keyboard navigation support
- Screen reader announcements for alerts
- High contrast mode support
- Focus indicators
- ARIA labels for all interactive elements

## Security Considerations

- **Authentication**: JWT tokens with refresh
- **Rate Limiting**: API call throttling
- **Data Sanitization**: XSS prevention
- **Secure WebSocket**: WSS protocol
- **Content Security Policy**: Strict CSP headers

## Future Enhancements

### Phase 2
- Customizable dashboard layouts
- Advanced filtering for threat feed
- Exportable reports
- Team collaboration features
- Mobile native apps

### Phase 3
- AI-powered threat predictions
- Automated policy recommendations
- Integration with SIEM systems
- Custom metric definitions
- Multi-tenant support

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Page Load Time | < 2 seconds | Performance monitoring |
| Time to First Insight | < 5 seconds | User observation |
| Dashboard Engagement | > 70% daily active | Analytics tracking |
| Alert Response Time | < 30 seconds | Incident tracking |
| User Satisfaction | > 4.5/5 | Regular surveys |

## Conclusion

This dashboard design provides security teams with:
- **Immediate situational awareness** without information overload
- **Actionable insights** for rapid response
- **Clear visual hierarchy** for priority focus
- **Seamless navigation** to detailed analysis
- **Enterprise reliability** with modern aesthetics

The minimal design ensures that users can quickly identify threats, understand system performance, and take action without cognitive overload, while the modern interface provides a professional experience worthy of enterprise deployment.