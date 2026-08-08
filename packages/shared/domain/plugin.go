package domain

import (
	"encoding/json"
	"time"
)

// PluginType defines whether the plugin runs on a probe, on a customer agent,
// or as a collector that scrapes external endpoints.
type PluginType string

const (
	PluginTypeProbe     PluginType = "probe"
	PluginTypeAgent     PluginType = "agent"
	PluginTypeCollector PluginType = "collector"
)

type Plugin struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Slug                string          `json:"slug"`
	Description         string          `json:"description"`
	Type                PluginType      `json:"type"`
	Version             string          `json:"version"`
	Icon                string          `json:"icon"`
	Category            string          `json:"category"`
	ConfigurationSchema json.RawMessage `json:"configuration_schema"`
	Enabled             bool            `json:"enabled"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type PluginInput struct {
	Name                string          `json:"name"`
	Slug                string          `json:"slug"`
	Description         string          `json:"description"`
	Type                PluginType      `json:"type"`
	Version             string          `json:"version"`
	Icon                string          `json:"icon"`
	Category            string          `json:"category"`
	ConfigurationSchema json.RawMessage `json:"configuration_schema"`
	Enabled             *bool           `json:"enabled,omitempty"`
}
