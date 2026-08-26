package permission

import (
	"fmt"
	"strings"
)

const DocumentVersion = 1

type Group struct {
	Parents     []string        `json:"parents,omitempty"`
	Permissions map[string]bool `json:"permissions,omitempty"`
}

type User struct {
	Groups      []string        `json:"groups,omitempty"`
	Permissions map[string]bool `json:"permissions,omitempty"`
}

type Document struct {
	Version int              `json:"version"`
	Groups  map[string]Group `json:"groups"`
	Users   map[string]User  `json:"users"`
}

func DefaultDocument() Document {
	return Document{
		Version: DocumentVersion,
		Groups: map[string]Group{
			"default": {},
			"admin":   {Parents: []string{"default"}, Permissions: map[string]bool{"*": true}},
		},
		Users: map[string]User{},
	}
}

func Normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (d Document) Validate() error {
	if d.Version != DocumentVersion {
		return fmt.Errorf("unsupported permission document version %d", d.Version)
	}
	if _, ok := d.Groups["default"]; !ok {
		return fmt.Errorf("default group is required")
	}
	for name, group := range d.Groups {
		if name == "" || name != Normalize(name) || strings.ContainsAny(name, " \t\r\n") {
			return fmt.Errorf("group name %q must be normalized", name)
		}
		for _, parent := range group.Parents {
			if _, ok := d.Groups[Normalize(parent)]; !ok {
				return fmt.Errorf("group %q has unknown parent %q", name, parent)
			}
		}
		if err := validateNodes(group.Permissions); err != nil {
			return fmt.Errorf("group %q: %w", name, err)
		}
	}
	for name, user := range d.Users {
		if name == "" || name != Normalize(name) {
			return fmt.Errorf("user name %q must be normalized", name)
		}
		for _, group := range user.Groups {
			if _, ok := d.Groups[Normalize(group)]; !ok {
				return fmt.Errorf("user %q has unknown group %q", name, group)
			}
		}
		if err := validateNodes(user.Permissions); err != nil {
			return fmt.Errorf("user %q: %w", name, err)
		}
	}
	return nil
}

func validateNodes(nodes map[string]bool) error {
	for node := range nodes {
		if node == "" || node != Normalize(node) || strings.ContainsAny(node, " \t\r\n") {
			return fmt.Errorf("invalid permission node %q", node)
		}
	}
	return nil
}
