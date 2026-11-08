package ctrader

import (
	"context"
	"errors"
	"testing"

	"ila/backend/internal/broker"
)

func TestClient_PlatformAndAccountID(t *testing.T) {
	c := New("acc-123", "client-id", "client-sec")
	if c.Platform() != "ctrader" {
		t.Errorf("got platform %s, want ctrader", c.Platform())
	}
	if c.AccountID() != "acc-123" {
		t.Errorf("got accountID %s, want acc-123", c.AccountID())
	}
	if c.Connected() {
		t.Errorf("expected disconnected initially")
	}
}

func TestClient_Symbol(t *testing.T) {
	c := New("acc-123", "client-id", "client-sec")
	spec, err := c.Symbol(context.Background(), "EURUSD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Digits != 5 {
		t.Errorf("got digits %d, want 5", spec.Digits)
	}

	specGold, err := c.Symbol(context.Background(), "GOLD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if specGold.Symbol != "XAUUSD" {
		t.Errorf("got symbol %s, want XAUUSD", specGold.Symbol)
	}

	_, err = c.Symbol(context.Background(), "UNKNOWN_COIN")
	if !errors.Is(err, broker.ErrUnsupported) {
		t.Errorf("expected ErrUnsupported, got %v", err)
	}
}

func TestClient_Connect_Validation(t *testing.T) {
	c := New("acc-123", "client-id", "client-sec")

	// Missing token
	err := c.Connect(context.Background(), broker.Credentials{Platform: "ctrader"})
	if !errors.Is(err, broker.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for missing token, got %v", err)
	}

	// Missing account ID
	err = c.Connect(context.Background(), broker.Credentials{Platform: "ctrader", Token: "some-token"})
	if !errors.Is(err, broker.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for missing account ID, got %v", err)
	}
}
