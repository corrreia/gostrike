// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// Config provides configuration loading for plugins
type Config struct {
	path string
	data map[string]interface{}
}

// LoadConfig loads a JSON configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &Config{
		path: path,
		data: config,
	}, nil
}

// LoadConfigOrDefault loads a config file, returning an empty config if the file doesn't exist
func LoadConfigOrDefault(path string) *Config {
	config, err := LoadConfig(path)
	if err != nil {
		return &Config{
			path: path,
			data: make(map[string]interface{}),
		}
	}
	return config
}

// get retrieves a value by key, supporting dot notation (e.g., "server.name")
func (c *Config) get(key string) (interface{}, bool) {
	if c == nil || c.data == nil {
		return nil, false
	}

	parts := strings.Split(key, ".")
	current := c.data

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil, false
		}

		// Last part - return the value
		if i == len(parts)-1 {
			return val, true
		}

		// Not last part - must be a map
		nested, ok := val.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = nested
	}

	return nil, false
}

// GetString retrieves a string value
func (c *Config) GetString(key string, defaultVal string) string {
	val, ok := c.get(key)
	if !ok {
		return defaultVal
	}

	switch v := val.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return defaultVal
	}
}

// GetInt retrieves an integer value
func (c *Config) GetInt(key string, defaultVal int) int {
	val, ok := c.get(key)
	if !ok {
		return defaultVal
	}

	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return defaultVal
		}
		return i
	default:
		return defaultVal
	}
}

// GetFloat retrieves a float value
func (c *Config) GetFloat(key string, defaultVal float64) float64 {
	val, ok := c.get(key)
	if !ok {
		return defaultVal
	}

	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return defaultVal
		}
		return f
	default:
		return defaultVal
	}
}

// GetBool retrieves a boolean value
func (c *Config) GetBool(key string, defaultVal bool) bool {
	val, ok := c.get(key)
	if !ok {
		return defaultVal
	}

	switch v := val.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return defaultVal
		}
		return b
	case float64:
		return v != 0
	default:
		return defaultVal
	}
}

// GetStringSlice retrieves a string slice
func (c *Config) GetStringSlice(key string) []string {
	val, ok := c.get(key)
	if !ok {
		return nil
	}

	arr, ok := val.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// GetIntSlice retrieves an integer slice
func (c *Config) GetIntSlice(key string) []int {
	val, ok := c.get(key)
	if !ok {
		return nil
	}

	arr, ok := val.([]interface{})
	if !ok {
		return nil
	}

	result := make([]int, 0, len(arr))
	for _, item := range arr {
		if num, ok := item.(float64); ok {
			result = append(result, int(num))
		}
	}
	return result
}

// Has returns true if the key exists
func (c *Config) Has(key string) bool {
	_, ok := c.get(key)
	return ok
}

// GetPath returns the config file path
func (c *Config) GetPath() string {
	return c.path
}

// Save writes the config back to the file
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c.data, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0644)
}

// Set sets a value (does not persist until Save is called)
func (c *Config) Set(key string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}

	parts := strings.Split(key, ".")
	current := c.data

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[part] = next
		}
		current = next
	}
}
