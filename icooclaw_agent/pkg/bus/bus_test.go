package bus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMessageBus_BasicOperations(t *testing.T) {
	cfg := Config{
		InboundCapacity:  10,
		OutboundCapacity: 10,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Publish and Consume Inbound", func(t *testing.T) {
		msg := InboundMessage{
			Channel:   "test",
			SessionID: "session-1",
			Text:      "Hello",
		}

		if err := bus.PublishInbound(ctx, msg); err != nil {
			t.Fatalf("PublishInbound failed: %v", err)
		}

		consumed, ok := bus.ConsumeInbound(ctx)
		if !ok {
			t.Fatal("ConsumeInbound returned false")
		}

		if consumed.Text != msg.Text {
			t.Errorf("consumed.Text = %q, want %q", consumed.Text, msg.Text)
		}
		if consumed.Channel != msg.Channel {
			t.Errorf("consumed.Channel = %q, want %q", consumed.Channel, msg.Channel)
		}
	})

	t.Run("Publish and Consume Outbound", func(t *testing.T) {
		msg := OutboundMessage{
			Channel:   "test",
			SessionID: "session-1",
			Text:      "World",
		}

		if err := bus.PublishOutbound(ctx, msg); err != nil {
			t.Fatalf("PublishOutbound failed: %v", err)
		}

		consumed, ok := bus.ConsumeOutbound(ctx)
		if !ok {
			t.Fatal("ConsumeOutbound returned false")
		}

		if consumed.Text != msg.Text {
			t.Errorf("consumed.Text = %q, want %q", consumed.Text, msg.Text)
		}
	})

	t.Run("Close stops the bus", func(t *testing.T) {
		bus.Close()
		if !bus.IsClosed() {
			t.Error("IsClosed should return true after Close")
		}
	})
}

func TestMessageBus_Subscribe(t *testing.T) {
	cfg := Config{
		InboundCapacity:  10,
		OutboundCapacity: 10,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Subscribe to Inbound", func(t *testing.T) {
		sub := bus.SubscribeInbound("test-sub", 5)

		msg := InboundMessage{
			Channel:   "test",
			SessionID: "session-1",
			Text:      "Subscriber test",
		}

		if err := bus.PublishInbound(ctx, msg); err != nil {
			t.Fatalf("PublishInbound failed: %v", err)
		}

		select {
		case received := <-sub:
			if received.Text != msg.Text {
				t.Errorf("received.Text = %q, want %q", received.Text, msg.Text)
			}
		case <-time.After(time.Second):
			t.Error("timeout waiting for subscriber message")
		}

		bus.UnsubscribeInbound("test-sub")
	})

	t.Run("Subscribe to Outbound", func(t *testing.T) {
		sub := bus.SubscribeOutbound("test-sub-out", 5)

		msg := OutboundMessage{
			Channel:   "test",
			SessionID: "session-1",
			Text:      "Outbound subscriber test",
		}

		if err := bus.PublishOutbound(ctx, msg); err != nil {
			t.Fatalf("PublishOutbound failed: %v", err)
		}

		select {
		case received := <-sub:
			if received.Text != msg.Text {
				t.Errorf("received.Text = %q, want %q", received.Text, msg.Text)
			}
		case <-time.After(time.Second):
			t.Error("timeout waiting for subscriber message")
		}

		bus.UnsubscribeOutbound("test-sub-out")
	})
}

func TestMessageBus_DropCount(t *testing.T) {
	cfg := Config{
		InboundCapacity:  2,
		OutboundCapacity: 2,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Drop count on full subscriber buffer", func(t *testing.T) {
		_ = bus.SubscribeInbound("slow-sub", 1)

		for i := 0; i < 5; i++ {
			msg := InboundMessage{
				Channel:   "test",
				SessionID: "session-1",
				Text:      "Message",
			}
			bus.PublishInbound(ctx, msg)
		}

		time.Sleep(100 * time.Millisecond)

		dropCount := bus.DropCount()
		if dropCount == 0 {
			t.Error("expected drop count > 0, got 0")
		}

		bus.UnsubscribeInbound("slow-sub")
	})
}

func TestMessageBus_GetMetrics(t *testing.T) {
	cfg := Config{
		InboundCapacity:  100,
		OutboundCapacity: 100,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Metrics reflect queue state", func(t *testing.T) {
		metrics := bus.GetMetrics()

		if metrics.InboundCapacity != 100 {
			t.Errorf("metrics.InboundCapacity = %d, want 100", metrics.InboundCapacity)
		}
		if metrics.OutboundCapacity != 100 {
			t.Errorf("metrics.OutboundCapacity = %d, want 100", metrics.OutboundCapacity)
		}
		if metrics.InboundQueueSize != 0 {
			t.Errorf("metrics.InboundQueueSize = %d, want 0", metrics.InboundQueueSize)
		}

		for i := 0; i < 50; i++ {
			msg := InboundMessage{
				Channel:   "test",
				SessionID: "session-1",
				Text:      "Test",
			}
			bus.PublishInbound(ctx, msg)
		}

		metrics = bus.GetMetrics()
		if metrics.InboundQueueSize != 50 {
			t.Errorf("metrics.InboundQueueSize = %d, want 50", metrics.InboundQueueSize)
		}
		if metrics.InboundUtilization != 50 {
			t.Errorf("metrics.InboundUtilization = %d%%, want 50%%", metrics.InboundUtilization)
		}
	})
}

func TestMessageBus_AlertHandler(t *testing.T) {
	cfg := Config{
		InboundCapacity:  10,
		OutboundCapacity: 10,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("AlertHandler called on drops", func(t *testing.T) {
		var alerts []Alert
		var mu sync.Mutex

		bus.SetAlertHandler(func(a Alert) {
			mu.Lock()
			alerts = append(alerts, a)
			mu.Unlock()
		})

		_ = bus.SubscribeInbound("slow-sub", 1)

		for i := 0; i < 10; i++ {
			msg := InboundMessage{
				Channel:   "test",
				SessionID: "session-1",
				Text:      "Test",
			}
			bus.PublishInbound(ctx, msg)
		}

		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		hasDropAlert := false
		for _, a := range alerts {
			if a.Type == AlertDropDetected {
				hasDropAlert = true
				break
			}
		}
		mu.Unlock()

		if !hasDropAlert {
			t.Error("expected AlertDropDetected alert")
		}

		bus.UnsubscribeInbound("slow-sub")
	})

	bus.Close()
}

func TestMessageBus_ConcurrentAccess(t *testing.T) {
	cfg := Config{
		InboundCapacity:  100,
		OutboundCapacity: 100,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	t.Run("Concurrent publish", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					msg := InboundMessage{
						Channel:   "test",
						SessionID: "session-1",
						Text:      "Message",
					}
					bus.PublishInbound(ctx, msg)
				}
			}(i)
		}
		wg.Wait()

		metrics := bus.GetMetrics()
		if metrics.InboundQueueSize == 0 {
			t.Error("expected some messages in queue")
		}
	})

	bus.Close()
}

func TestMessageBus_ReplacSubscription(t *testing.T) {
	cfg := Config{
		InboundCapacity:  10,
		OutboundCapacity: 10,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Replace existing subscription", func(t *testing.T) {
		sub1 := bus.SubscribeInbound("same-sub", 5)
		sub2 := bus.SubscribeInbound("same-sub", 5)

		if sub1 == sub2 {
			t.Error("expected different channels after replace")
		}

		msg := InboundMessage{
			Channel:   "test",
			SessionID: "session-1",
			Text:      "Test",
		}
		bus.PublishInbound(ctx, msg)

		select {
		case <-sub1:
			t.Error("old subscription should not receive messages")
		case <-sub2:
		case <-time.After(time.Second):
			t.Error("timeout waiting for message on new subscription")
		}
	})

	bus.Close()
}

func TestMessageBus_PublishOutboundMedia(t *testing.T) {
	cfg := Config{
		InboundCapacity:  10,
		OutboundCapacity: 10,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Publish and consume media message", func(t *testing.T) {
		msg := OutboundMediaMessage{
			Channel: "test",
			Media:   []string{"image1.jpg", "image2.jpg"},
			Caption: "Test images",
		}

		if err := bus.PublishOutboundMedia(ctx, msg); err != nil {
			t.Fatalf("PublishOutboundMedia failed: %v", err)
		}

		consumed, ok := bus.ConsumeOutboundMedia(ctx)
		if !ok {
			t.Fatal("ConsumeOutboundMedia returned false")
		}

		if len(consumed.Media) != 2 {
			t.Errorf("consumed.Media len = %d, want 2", len(consumed.Media))
		}
		if consumed.Caption != msg.Caption {
			t.Errorf("consumed.Caption = %q, want %q", consumed.Caption, msg.Caption)
		}
	})

	bus.Close()
}

func TestMessageBus_ContextCancellation(t *testing.T) {
	cfg := Config{
		InboundCapacity:  1,
		OutboundCapacity: 1,
	}
	bus := NewMessageBus(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	_ = bus.PublishInbound(ctx, InboundMessage{Channel: "test", Text: "msg1"})
	_ = bus.PublishInbound(ctx, InboundMessage{Channel: "test", Text: "msg2"})

	cancel()

	time.Sleep(100 * time.Millisecond)

	_, ok := bus.ConsumeInbound(ctx)
	if ok {
		t.Error("expected ConsumeInbound to return false after context cancel")
	}
}

func TestMessageBus_ClosedBus(t *testing.T) {
	cfg := Config{
		InboundCapacity:  10,
		OutboundCapacity: 10,
	}
	bus := NewMessageBus(cfg)
	bus.Close()

	ctx := context.Background()

	t.Run("PublishInbound after close", func(t *testing.T) {
		err := bus.PublishInbound(ctx, InboundMessage{Channel: "test", Text: "msg"})
		if err == nil {
			t.Error("expected error on PublishInbound after close")
		}
	})

	t.Run("PublishOutbound after close", func(t *testing.T) {
		err := bus.PublishOutbound(ctx, OutboundMessage{Channel: "test", Text: "msg"})
		if err == nil {
			t.Error("expected error on PublishOutbound after close")
		}
	})

	t.Run("ConsumeInbound after close", func(t *testing.T) {
		_, ok := bus.ConsumeInbound(ctx)
		if ok {
			t.Error("expected ConsumeInbound to return false after close")
		}
	})
}
