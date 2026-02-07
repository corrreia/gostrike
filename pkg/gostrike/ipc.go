// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides inter-plugin communication (IPC) primitives.
package gostrike

import (
	"github.com/corrreia/gostrike/internal/ipc"
	"github.com/corrreia/gostrike/internal/scope"
)

// Subscribe registers a callback for an IPC topic.
// Returns a subscription ID that can be used to unsubscribe.
func Subscribe(topic string, callback func(data map[string]any)) uint64 {
	id := ipc.Subscribe(topic, callback)
	if s := scope.GetActive(); s != nil {
		s.TrackEventSub(id)
	}
	return id
}

// Unsubscribe removes an IPC subscription by ID.
func Unsubscribe(id uint64) {
	ipc.Unsubscribe(id)
}

// Publish sends data to all subscribers of an IPC topic.
func Publish(topic string, data map[string]any) {
	ipc.Publish(topic, data)
}

// RegisterService registers a named service for other plugins to use.
func RegisterService(name string, service interface{}) error {
	err := ipc.RegisterService(name, service)
	if err == nil {
		if s := scope.GetActive(); s != nil {
			s.TrackService(name)
		}
	}
	return err
}

// GetService retrieves a named service registered by another plugin.
// Returns nil if the service is not found.
func GetService(name string) interface{} {
	return ipc.GetService(name)
}
