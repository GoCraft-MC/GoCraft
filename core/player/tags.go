package player

import (
	"sort"
	"strings"
)

func (p *Player) AddTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	if tag == "" || len(tag) > 1024 {
		return false
	}
	p.tagsMu.Lock()
	defer p.tagsMu.Unlock()
	if p.tags == nil {
		p.tags = make(map[string]struct{})
	}
	if _, exists := p.tags[tag]; exists {
		return false
	}
	p.tags[tag] = struct{}{}
	return true
}

func (p *Player) RemoveTag(tag string) bool {
	p.tagsMu.Lock()
	defer p.tagsMu.Unlock()
	if _, exists := p.tags[tag]; !exists {
		return false
	}
	delete(p.tags, tag)
	return true
}

func (p *Player) Tags() []string {
	p.tagsMu.RLock()
	tags := make([]string, 0, len(p.tags))
	for tag := range p.tags {
		tags = append(tags, tag)
	}
	p.tagsMu.RUnlock()
	sort.Strings(tags)
	return tags
}
