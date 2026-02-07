// Package ipc provides inter-plugin communication primitives:
// a publish/subscribe event bus and a named service registry.
package ipc

import (
	"sync"
	"sync/atomic"

	"github.com/corrreia/gostrike/internal/shared"
)

type subscription struct {
	id       uint64
	slug     string // owning plugin slug (for bulk cleanup)
	callback func(data map[string]any)
}

var (
	subsMu    sync.RWMutex
	subs      = make(map[string][]*subscription) // topic -> subscriptions
	nextSubID uint64
)

// Subscribe registers a callback for a topic. Returns a subscription ID.
func Subscribe(topic string, callback func(data map[string]any)) uint64 {
	return SubscribeForPlugin("", topic, callback)
}

// SubscribeForPlugin registers a callback with an owning plugin slug.
func SubscribeForPlugin(slug, topic string, callback func(data map[string]any)) uint64 {
	id := atomic.AddUint64(&nextSubID, 1)
	s := &subscription{id: id, slug: slug, callback: callback}

	subsMu.Lock()
	subs[topic] = append(subs[topic], s)
	subsMu.Unlock()

	return id
}

// Unsubscribe removes a subscription by ID.
func Unsubscribe(id uint64) {
	subsMu.Lock()
	defer subsMu.Unlock()

	for topic, entries := range subs {
		for i, s := range entries {
			if s.id == id {
				last := len(entries) - 1
				entries[i] = entries[last]
				subs[topic] = entries[:last]
				return
			}
		}
	}
}

// Publish sends data to all subscribers of a topic.
// Callbacks are invoked synchronously with panic recovery.
func Publish(topic string, data map[string]any) {
	subsMu.RLock()
	entries := subs[topic]
	subsMu.RUnlock()

	for _, s := range entries {
		func() {
			defer func() {
				if r := recover(); r != nil {
					shared.LogError("IPC", "Panic in subscriber %d for topic '%s': %v", s.id, topic, r)
				}
			}()
			s.callback(data)
		}()
	}
}

// UnsubscribeAllForPlugin removes all subscriptions owned by a plugin slug.
func UnsubscribeAllForPlugin(slug string) {
	subsMu.Lock()
	defer subsMu.Unlock()

	for topic, entries := range subs {
		filtered := entries[:0]
		for _, s := range entries {
			if s.slug != slug {
				filtered = append(filtered, s)
			}
		}
		subs[topic] = filtered
	}
}
