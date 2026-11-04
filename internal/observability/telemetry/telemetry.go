package telemetry

// Registry initialization package for PromptShield unified observability.
// Combines telemetry event collection and distributed tracing in one system:
// - collector.go: Main Collector implementing TelemetryCollector + tracing capabilities
// - provider.go: Unified OpenTelemetry provider setup (meters + tracers, no global state)  
// - exporter.go: Event exporting to files and OTel collectors
// - Helper functions moved to internal/util/tracing/ for span creation

// This unified approach eliminates duplication between telemetry and tracing
// while providing both business analytics and technical debugging capabilities. 
