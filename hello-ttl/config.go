package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type serverConfig struct {
	mu           sync.RWMutex
	Language     *string        `json:"language"`
	ConsulAddr   *string        `json:"consul_addr"`
	KVPath       *string        `json:"kv_path"`
	ServiceName  *string        `json:"service_name"`
	TTLEndpoint  *string        `json:"ttl_endpoint"`
	TTLID        *string        `json:"ttl_id"`
	TTLInterval  *time.Duration `json:"ttl_interval"`
	EnableChecks *bool          `json:"enable_checks"`
	DebugMode    *bool          `json:"debug_mode"`
	ToWatch      *[]string      `json:"keys_to_watch"`
}

func (c *serverConfig) finalize() *serverConfig {
	def := defaultConfig()

	if c == nil {
		return &def
	}
	if c.Language == nil {
		c.Language = def.Language
	}
	if c.ConsulAddr == nil {
		c.ConsulAddr = def.ConsulAddr
	}
	if c.KVPath == nil {
		c.KVPath = def.KVPath
	}
	if c.ServiceName == nil {
		c.ServiceName = def.ServiceName
	}
	if c.TTLEndpoint == nil {
		c.TTLEndpoint = def.TTLEndpoint
	}
	if c.TTLID == nil {
		c.TTLID = def.TTLID
	}
	if c.TTLInterval == nil {
		c.TTLInterval = def.TTLInterval
	}
	if c.EnableChecks == nil {
		c.EnableChecks = def.EnableChecks
	}
	if c.DebugMode == nil {
		c.DebugMode = def.DebugMode
	}
	if c.ToWatch == nil {
		c.ToWatch = def.ToWatch
	}
	return c
}

func defaultConfig() serverConfig {
	return serverConfig{
		Language:     StringPtr("english"),
		ConsulAddr:   StringPtr("http://localhost:8500"),
		KVPath:       StringPtr("/v1/kv/service/hello/"),
		ServiceName:  StringPtr("hello-ttl/"),
		TTLEndpoint:  StringPtr("/v1/agent/check/pass/"),
		TTLInterval:  DurationPtr(5 * time.Second),
		TTLID:        StringPtr("hello_ttl"),
		EnableChecks: BoolPtr(true),
		DebugMode:    BoolPtr(false),
		ToWatch:      SlicePtr([]string{"hello-ttl/enable_checks"}),
	}
}

func loadConfig(filename string) (*serverConfig, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open '%s': %v", filename, err)
	}
	defer f.Close()

	body, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read '%s': %v", filename, err)
	}

	var cfg serverConfig
	json.Unmarshal(body, &cfg)

	return &cfg, nil
}

// BoolPtr returns a pointer to the given bool.
func BoolPtr(b bool) *bool {
	return &b
}

// BoolVal returns the value of the boolean at the pointer, or false if the
// pointer is nil.
func BoolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string {
	return &s
}

// StringVal returns the value of the string at the pointer, or "" if the
// pointer is nil.
func StringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// DurationPtr returns a pointer to the given time.Duration.
func DurationPtr(t time.Duration) *time.Duration {
	return &t
}

// DurationVal returns the value of the string at the pointer, or 0 if the
// pointer is nil.
func DurationVal(t *time.Duration) time.Duration {
	if t == nil {
		return time.Duration(0)
	}
	return *t
}

// SlicePtr returns a pointer to the given string slice.
func SlicePtr(s []string) *[]string {
	return &s
}

// SliceVal returns the value of the slice at the pointer, or an empty
// slice if the pointer is nil
func SliceVal(s *[]string) []string {
	if s == nil {
		return []string{}
	}
	return *s
}
