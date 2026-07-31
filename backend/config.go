package main

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type Address struct {
	Net   string `yaml:"net" json:"net"`
	Label string `yaml:"label" json:"label"`
	URL   string `yaml:"url" json:"url"`
}

type Site struct {
	Name      string    `yaml:"name" json:"name"`
	Desc      string    `yaml:"desc" json:"desc"`
	Icon      string    `yaml:"icon" json:"icon,omitempty"`
	Addresses []Address `yaml:"addresses" json:"addresses"`
}

type Group struct {
	Code  string `yaml:"code" json:"code"`
	Name  string `yaml:"name" json:"name"`
	Sites []Site `yaml:"sites" json:"sites"`
}

type Config struct {
	Title       string  `yaml:"title" json:"title"`
	Desc        string  `yaml:"desc" json:"desc"`
	Theme       string  `yaml:"theme" json:"theme"`
	NetworkMode string  `yaml:"networkMode" json:"networkMode"`
	Groups      []Group `yaml:"groups" json:"groups"`
}

func parseConfig(data []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if c.Title == "" {
		return nil, fmt.Errorf("parse config: title is required")
	}
	if len(c.Groups) == 0 {
		return nil, fmt.Errorf("parse config: groups must not be empty")
	}
	for gi, g := range c.Groups {
		if g.Name == "" {
			return nil, fmt.Errorf("parse config: groups[%d].name is required", gi)
		}
		for si, s := range g.Sites {
			if s.Name == "" {
				return nil, fmt.Errorf("parse config: groups[%d].sites[%d].name is required", gi, si)
			}
			if len(s.Addresses) == 0 {
				return nil, fmt.Errorf("parse config: site %q must have at least one address", s.Name)
			}
			for ai, a := range s.Addresses {
				if a.Net == "" || a.Label == "" || a.URL == "" {
					return nil, fmt.Errorf("parse config: site %q addresses[%d] requires net, label and url", s.Name, ai)
				}
				if a.Net != "intranet" && a.Net != "internet" {
					return nil, fmt.Errorf("parse config: site %q addresses[%d].net must be intranet or internet, got %q", s.Name, ai, a.Net)
				}
			}
		}
	}
	if c.Theme == "" {
		c.Theme = "dark"
	}
	if c.NetworkMode == "" {
		c.NetworkMode = "intranet"
	}
	return &c, nil
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseConfig(data)
}

type Store struct {
	mu  sync.RWMutex
	cfg *Config
}

func NewStore(c *Config) *Store { return &Store{cfg: c} }

func (s *Store) Get() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Store) Set(c *Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = c
}
