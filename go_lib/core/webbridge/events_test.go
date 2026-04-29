package webbridge

import (
	"testing"
	"time"
)

func TestEventHubPublishDeliversToSubscribedClients(t *testing.T) {
	hub := NewEventHub()
	ch, unsubscribe := hub.subscribe()
	defer unsubscribe()

	hub.Publish(`{"type":"log","payload":{"message":"ok"}}`)

	select {
	case got := <-ch:
		if got != `{"type":"log","payload":{"message":"ok"}}` {
			t.Fatalf("expected published message, got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected subscribed client to receive published message")
	}
}

func TestEventHubPublishKeepsLatestMessageForSlowClient(t *testing.T) {
	hub := NewEventHub()
	ch, unsubscribe := hub.subscribe()
	defer unsubscribe()

	for i := 0; i < cap(ch); i++ {
		ch <- "old"
	}

	done := make(chan struct{})
	go func() {
		hub.Publish("latest")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish should not block on a full client channel")
	}

	foundLatest := false
	for len(ch) > 0 {
		if <-ch == "latest" {
			foundLatest = true
			break
		}
	}
	if !foundLatest {
		t.Fatal("expected slow client buffer to keep the latest message")
	}
}

func TestEventHubUnsubscribeRemovesAndClosesClient(t *testing.T) {
	hub := NewEventHub()
	ch, unsubscribe := hub.subscribe()

	unsubscribe()

	if _, ok := <-ch; ok {
		t.Fatal("expected unsubscribed channel to be closed")
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if len(hub.clients) != 0 {
		t.Fatalf("expected no subscribed clients, got %d", len(hub.clients))
	}
}
