package version

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestDefaultHTTPClientSingleton(t *testing.T) {
	t.Helper()

	first := defaultHTTPClient()
	second := defaultHTTPClient()
	if first == nil {
		t.Fatalf("expected non-nil default client")
	}
	if first != second {
		t.Fatalf("expected default client to be reused")
	}
}

func TestDocsHTTPClientSingleton(t *testing.T) {
	t.Helper()

	first := docsHTTPClient()
	second := docsHTTPClient()
	if first == nil {
		t.Fatalf("expected non-nil docs client")
	}
	if first != second {
		t.Fatalf("expected docs client to be reused")
	}
}

func TestContextWithHTTPClientOverride(t *testing.T) {
	t.Helper()

	custom := &http.Client{}
	ctx := ContextWithHTTPClient(context.Background(), custom)
	got := httpClientFromContext(ctx, httpClientKeyGitHub, defaultHTTPClient)
	if got != custom {
		t.Fatalf("expected context override to be used")
	}
}

func TestContextWithDocsHTTPClientOverride(t *testing.T) {
	t.Helper()

	custom := &http.Client{}
	ctx := ContextWithDocsHTTPClient(context.Background(), custom)
	got := httpClientFromContext(ctx, httpClientKeyDocs, docsHTTPClient)
	if got != custom {
		t.Fatalf("expected docs context override to be used")
	}
}

func TestContextWithHTTPClientNilFallsBack(t *testing.T) {
	t.Helper()

	ctx := ContextWithHTTPClient(nil, nil)
	def := defaultHTTPClient()
	got := httpClientFromContext(ctx, httpClientKeyGitHub, defaultHTTPClient)
	if got != def {
		t.Fatalf("expected fallback client when override is nil")
	}
}

func TestSetDefaultHTTPClientOverride(t *testing.T) {
	t.Helper()

	original := defaultHTTPClient()
	override := &http.Client{}
	SetDefaultHTTPClient(override)
	t.Cleanup(func() { SetDefaultHTTPClient(original) })

	got := defaultHTTPClient()
	if got != override {
		t.Fatalf("expected default HTTP client override to be used")
	}
}

func TestSetDocsHTTPClientOverride(t *testing.T) {
	t.Helper()

	original := docsHTTPClient()
	override := &http.Client{}
	SetDocsHTTPClient(override)
	t.Cleanup(func() { SetDocsHTTPClient(original) })

	got := docsHTTPClient()
	if got != override {
		t.Fatalf("expected docs HTTP client override to be used")
	}
}

func TestWithTargetTimeoutDefault(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	derived, cancel := withTargetTimeout(ctx, httpTimeoutKeyGitHub)
	if cancel == nil {
		t.Fatalf("expected cancel func when default timeout applies")
	}
	defer cancel()
	if _, ok := derived.Deadline(); !ok {
		t.Fatalf("expected default timeout to set deadline")
	}
}

func TestContextWithHTTPTimeoutOverride(t *testing.T) {
	t.Helper()

	ctx := ContextWithHTTPTimeout(context.Background(), 2*time.Second)
	derived, cancel := withTargetTimeout(ctx, httpTimeoutKeyGitHub)
	if cancel == nil {
		t.Fatalf("expected cancel func for override timeout")
	}
	defer cancel()
	deadline, ok := derived.Deadline()
	if !ok {
		t.Fatalf("expected deadline when override timeout is set")
	}
	delta := time.Until(deadline)
	if delta > 3*time.Second || delta < time.Second {
		t.Fatalf("expected deadline roughly 2s away, got %v", delta)
	}
}

func TestContextWithHTTPTimeoutDisable(t *testing.T) {
	t.Helper()

	ctx := ContextWithHTTPTimeout(context.Background(), 0)
	derived, cancel := withTargetTimeout(ctx, httpTimeoutKeyGitHub)
	if cancel != nil {
		t.Fatalf("expected no cancel func when timeout disabled")
	}
	if _, ok := derived.Deadline(); ok {
		t.Fatalf("expected no deadline when timeout disabled")
	}
}

func TestWithTargetTimeoutRespectsExistingDeadline(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	derived, derivedCancel := withTargetTimeout(ctx, httpTimeoutKeyGitHub)
	if derivedCancel != nil {
		t.Fatalf("expected nil cancel when deadline already set")
	}
	deadline, ok := derived.Deadline()
	if !ok {
		t.Fatalf("expected existing deadline to remain")
	}
	if time.Until(deadline) <= 0 {
		t.Fatalf("expected remaining time on deadline")
	}
}

func TestDefaultClientKeepsHandshakeTimeoutWhenConnectTimeoutDisabled(t *testing.T) {
	t.Helper()

	originalConnect := HTTPConnectTimeout
	originalClient := defaultHTTPClient()
	SetDefaultHTTPClient(originalClient)

	originalTransport, _ := originalClient.Transport.(*http.Transport)
	originalDefault := http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout
	if originalTransport != nil && originalTransport.TLSHandshakeTimeout == 0 && originalDefault == 0 {
		t.Skip("default transport handshake timeout is zero; nothing to assert")
	}

	SetDefaultHTTPClient(nil)
	HTTPConnectTimeout = 0
	t.Cleanup(func() {
		HTTPConnectTimeout = originalConnect
		SetDefaultHTTPClient(originalClient)
	})

	client := defaultHTTPClient()
	transport, _ := client.Transport.(*http.Transport)
	if transport == nil {
		t.Fatal("expected transport on default HTTP client")
	}

	if got := transport.TLSHandshakeTimeout; got != originalDefault {
		t.Fatalf("expected handshake timeout %v, got %v", originalDefault, got)
	}
}
