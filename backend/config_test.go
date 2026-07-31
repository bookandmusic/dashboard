package main

import (
	"encoding/json"
	"testing"
)

const validYAML = `
title: 内部工具导航
desc: 一台 NAS，两套链路
theme: dark
networkMode: intranet
groups:
  - code: DEV
    name: 研发工具
    sites:
      - name: 代码仓库
        desc: Git 代码托管
        icon: git-branch
        addresses:
          - net: intranet
            label: 内网
            url: http://git.intra.example
          - net: internet
            label: 外网
            url: https://git.example.com
      - name: 制品仓库
        addresses:
          - net: intranet
            label: 内网
            url: http://artifacts.intra.example
`

func TestParseConfig(t *testing.T) {
	cfg, err := parseConfig([]byte(validYAML))
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.Title != "内部工具导航" {
		t.Errorf("Title = %q", cfg.Title)
	}
	if cfg.Desc != "一台 NAS，两套链路" {
		t.Errorf("Desc = %q", cfg.Desc)
	}
	if cfg.Theme != "dark" {
		t.Errorf("Theme = %q", cfg.Theme)
	}
	if cfg.NetworkMode != "intranet" {
		t.Errorf("NetworkMode = %q", cfg.NetworkMode)
	}
	if len(cfg.Groups) != 1 {
		t.Fatalf("len(Groups) = %d", len(cfg.Groups))
	}
	g := cfg.Groups[0]
	if g.Code != "DEV" || g.Name != "研发工具" {
		t.Errorf("Group = %+v", g)
	}
	if len(g.Sites) != 2 {
		t.Fatalf("len(Sites) = %d", len(g.Sites))
	}
	s0 := g.Sites[0]
	if s0.Name != "代码仓库" || s0.Desc != "Git 代码托管" || s0.Icon != "git-branch" || len(s0.Addresses) != 2 {
		t.Errorf("Site[0] = %+v", s0)
	}
	a0 := s0.Addresses[0]
	if a0.Net != "intranet" || a0.Label != "内网" || a0.URL != "http://git.intra.example" {
		t.Errorf("Address[0] = %+v", a0)
	}
}

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := parseConfig([]byte("title: t\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - net: intranet\n            label: 内网\n            url: http://x\n"))
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("default Theme = %q, want dark", cfg.Theme)
	}
	if cfg.NetworkMode != "intranet" {
		t.Errorf("default NetworkMode = %q, want intranet", cfg.NetworkMode)
	}
}

func TestParseConfigErrors(t *testing.T) {
	cases := map[string]string{
		"invalid yaml":     ":\t-",
		"missing title":    "groups:\n  - name: g\n    sites: []\n",
		"empty groups":     "title: t\ngroups: []\n",
		"group no name":    "title: t\ngroups:\n  - sites: []\n",
		"site no name":     "title: t\ngroups:\n  - name: g\n    sites:\n      - addresses:\n          - {net: intranet, label: l, url: u}\n",
		"site no address":  "title: t\ngroups:\n  - name: g\n    sites:\n      - name: s\n",
		"address no url":   "title: t\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - {net: intranet, label: l}\n",
		"address bad net":  "title: t\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - {net: lan, label: l, url: u}\n",
	}
	for name, yaml := range cases {
		if _, err := parseConfig([]byte(yaml)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestConfigJSONShape(t *testing.T) {
	cfg, err := parseConfig([]byte(validYAML))
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for _, key := range []string{"title", "theme", "networkMode", "groups"} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON missing key %q: %s", key, data)
		}
	}
	var groups []map[string]json.RawMessage
	json.Unmarshal(m["groups"], &groups)
	var sites []map[string]json.RawMessage
	json.Unmarshal(groups[0]["sites"], &sites)
	var addrs []map[string]json.RawMessage
	json.Unmarshal(sites[0]["addresses"], &addrs)
	for _, key := range []string{"net", "label", "url"} {
		if _, ok := addrs[0][key]; !ok {
			t.Errorf("address JSON missing key %q", key)
		}
	}
}

func TestStoreGetSet(t *testing.T) {
	c1, _ := parseConfig([]byte(validYAML))
	s := NewStore(c1)
	if got := s.Get(); got.Title != "内部工具导航" {
		t.Errorf("Get().Title = %q", got.Title)
	}
	c2, _ := parseConfig([]byte("title: 新标题\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - {net: intranet, label: l, url: u}\n"))
	s.Set(c2)
	if got := s.Get(); got.Title != "新标题" {
		t.Errorf("after Set, Get().Title = %q", got.Title)
	}
}
