// This file contains tests that are executed to verify your solution.
package main

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
)

func Test(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	// Create a new GoChannel pub/sub for testing
	testPubSub := gochannel.NewGoChannel(
		gochannel.Config{
			Persistent: true,
		},
		logger,
	)

	// Set the global pubsub to our test instance
	goPubSub = testPubSub

	originalTopic := uuid.NewString()
	originalMessages, err := testPubSub.Subscribe(context.Background(), originalTopic)
	if err != nil {
		t.Fatal(err)
	}

	var uuids []string
	for i := 0; i < 10; i++ {
		msg := message.NewMessage(watermill.NewUUID(), []byte("{}"))
		msg.Metadata.Set(middleware.ReasonForPoisonedKey, "network down")
		msg.Metadata.Set(middleware.PoisonedTopicKey, originalTopic)
		if err := testPubSub.Publish(PoisonQueueTopic, msg); err != nil {
			t.Fatal(err)
		}
		uuids = append(uuids, msg.UUID)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}

	expectedUUIDs := []string{
		uuids[0],
		uuids[1],
		uuids[2],
		uuids[3],
		uuids[4],
		uuids[5],
		uuids[6],
		uuids[7],
		uuids[8],
		uuids[9],
	}

	assertMessages(t, expectedUUIDs)

	requeue(t, uuids[1])
	requeue(t, uuids[7])

	h, err = NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	err = h.Requeue(context.Background(), uuid.NewString())
	if err == nil {
		t.Fatal("expected to fail when requeuing unknown message ID")
	}

	expectedUUIDs = []string{
		uuids[0],
		uuids[2],
		uuids[3],
		uuids[4],
		uuids[5],
		uuids[6],
		uuids[8],
		uuids[9],
	}

	assertMessages(t, expectedUUIDs)

	for _, expectedUUID := range []string{uuids[1], uuids[7]} {
		select {
		case msg := <-originalMessages:
			if msg.UUID != expectedUUID {
				t.Fatalf("expected message with uuid %s, got %s", expectedUUID, msg.UUID)
			}
			msg.Ack()
		case <-time.After(time.Second):
		}
	}
}

func assertMessages(t *testing.T, expectedUUIDs []string) {
	t.Helper()

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}

	messages, err := h.Preview(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != len(expectedUUIDs) {
		t.Fatalf("expected %v messages, got %d", len(expectedUUIDs), len(messages))
	}

	for _, uuid := range expectedUUIDs {
		found := false
		for _, msg := range messages {
			if msg.ID == uuid {
				if msg.Reason != "network down" {
					t.Fatalf("expected reason to be 'network down', got %s", msg.Reason)
				}
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("expected message with uuid %s, but not found", uuid)
		}
	}
}

func requeue(t *testing.T, id string) {
	t.Helper()

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}

	err = h.Requeue(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
}
