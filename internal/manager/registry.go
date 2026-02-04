// Package manager provides plugin lifecycle management for GoStrike.
// This file contains the plugin registry for static plugin registration.
package manager

// RegisteredPlugins holds plugins registered at init time
// This allows plugins to be registered via init() before the manager starts
var RegisteredPlugins []interface{}
var RegisteredFactories []func() interface{}

// QueuePlugin queues a plugin for registration
// Called from plugin init() functions before manager starts
func QueuePlugin(p interface{}) {
	RegisteredPlugins = append(RegisteredPlugins, p)
}

// QueuePluginFactory queues a plugin factory for registration
func QueuePluginFactory(factory func() interface{}) {
	RegisteredFactories = append(RegisteredFactories, factory)
}

// ProcessQueue processes all queued registrations
// Called by Init() after the manager is ready
func ProcessQueue() {
	for _, p := range RegisteredPlugins {
		RegisterPlugin(p)
	}
	for _, f := range RegisteredFactories {
		RegisterPluginFunc(f)
	}

	// Clear queues
	RegisteredPlugins = nil
	RegisteredFactories = nil
}
