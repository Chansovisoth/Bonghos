package websocket

import (
	"testing"
	"time"
)

func testClient(topic string, interval time.Duration) *client {
	return &client{
		send:           make(chan []byte, 8),
		topics:         map[string]bool{topic: true},
		topicIntervals: map[string]time.Duration{topic: interval},
		lastTopicSent:  map[string]time.Time{},
	}
}

func TestMinimumIntervalUsesFastestSubscriber(t *testing.T) {
	h := NewHub()
	h.clients[testClient("performance", 5*time.Second)] = true
	h.clients[testClient("performance", 2*time.Second)] = true
	h.clients[testClient("overview", time.Second)] = true

	if got := h.MinimumInterval("performance", 10*time.Second); got != 2*time.Second {
		t.Fatalf("MinimumInterval() = %v, want 2s", got)
	}
}

func TestBroadcastDueRespectsPerClientInterval(t *testing.T) {
	h := NewHub()
	fast := testClient("performance", 2*time.Second)
	slow := testClient("performance", 5*time.Second)
	h.clients[fast] = true
	h.clients[slow] = true

	start := time.Unix(100, 0)
	h.broadcastDueAt("performance", "sample", map[string]int{"value": 1}, 10*time.Second, start)
	h.broadcastDueAt("performance", "sample", map[string]int{"value": 2}, 10*time.Second, start.Add(2*time.Second))

	if got := len(fast.send); got != 2 {
		t.Fatalf("fast subscriber received %d events, want 2", got)
	}
	if got := len(slow.send); got != 1 {
		t.Fatalf("slow subscriber received %d events, want 1", got)
	}

	h.broadcastDueAt("performance", "sample", map[string]int{"value": 3}, 10*time.Second, start.Add(5*time.Second))
	if got := len(slow.send); got != 2 {
		t.Fatalf("slow subscriber received %d events after 5s, want 2", got)
	}
}
