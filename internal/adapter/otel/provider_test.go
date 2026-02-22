package otel_test

import (
	"context"
	"testing"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/otel"
)

func TestSetup_StdoutExporter(t *testing.T) {
	providers, err := adapter.Setup(context.Background(), adapter.Config{
		ServiceName:    "test",
		ServiceVersion: "0.0.1",
		Environment:    "test",
		Exporter:       "stdout",
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if err := providers.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestSetup_InvalidExporter(t *testing.T) {
	_, err := adapter.Setup(context.Background(), adapter.Config{
		ServiceName:    "test",
		ServiceVersion: "0.0.1",
		Environment:    "test",
		Exporter:       "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid exporter")
	}
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	cfg := adapter.ConfigFromEnv()

	if cfg.ServiceName != "tenantiq" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "tenantiq")
	}
	if cfg.ServiceVersion != "0.1.0" {
		t.Errorf("ServiceVersion = %q, want %q", cfg.ServiceVersion, "0.1.0")
	}
	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "development")
	}
	if cfg.Exporter != "stdout" {
		t.Errorf("Exporter = %q, want %q", cfg.Exporter, "stdout")
	}
}

func TestConfigFromEnv_CustomValues(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "custom-service")
	t.Setenv("OTEL_SERVICE_VERSION", "1.0.0")
	t.Setenv("OTEL_ENVIRONMENT", "production")
	t.Setenv("OTEL_EXPORTER", "otlp")

	cfg := adapter.ConfigFromEnv()

	if cfg.ServiceName != "custom-service" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "custom-service")
	}
	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("ServiceVersion = %q, want %q", cfg.ServiceVersion, "1.0.0")
	}
	if cfg.Environment != "production" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "production")
	}
	if cfg.Exporter != "otlp" {
		t.Errorf("Exporter = %q, want %q", cfg.Exporter, "otlp")
	}
}
