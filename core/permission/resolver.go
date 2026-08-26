package permission

import "strings"

// Allowed resolves a command permission for one player. Operators retain a
// wildcard bypass for compatibility with vanilla ops.json administration.
func (m *Manager) Allowed(username, node string, operator, defaultAllowed bool) bool {
	if operator {
		return true
	}
	username, node = Normalize(username), Normalize(node)
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.document.Users[username]
	if value, _, found := permissionDecision(user.Permissions, node); exists && found {
		return value
	}
	groups := append([]string{"default"}, user.Groups...)
	bestValue, bestScore, found := false, -1, false
	for _, group := range groups {
		value, score, ok := m.groupDecision(Normalize(group), node, 50, map[string]bool{})
		if ok && (score > bestScore || score == bestScore && !value) {
			bestValue, bestScore, found = value, score, true
		}
	}
	if found {
		return bestValue
	}
	return defaultAllowed
}

func (m *Manager) groupDecision(name, node string, depth int, visiting map[string]bool) (bool, int, bool) {
	if visiting[name] {
		return false, 0, false
	}
	group, ok := m.document.Groups[name]
	if !ok {
		return false, 0, false
	}
	visiting[name] = true
	defer delete(visiting, name)
	bestValue, bestScore, found := permissionDecision(group.Permissions, node)
	if found {
		bestScore = bestScore*100 + max(depth, 0)
	}
	for _, parent := range group.Parents {
		value, score, parentFound := m.groupDecision(Normalize(parent), node, depth-1, visiting)
		if parentFound && (score > bestScore || score == bestScore && !value) {
			bestValue, bestScore, found = value, score, true
		}
	}
	return bestValue, bestScore, found
}

func permissionDecision(rules map[string]bool, node string) (bool, int, bool) {
	bestValue, bestScore, found := false, -1, false
	for pattern, value := range rules {
		score, matches := permissionMatch(Normalize(pattern), node)
		if matches && (score > bestScore || score == bestScore && !value) {
			bestValue, bestScore, found = value, score, true
		}
	}
	return bestValue, bestScore, found
}

func permissionMatch(pattern, node string) (int, bool) {
	switch {
	case pattern == node:
		return 1000 + len(pattern), true
	case pattern == "*":
		return 1, true
	case strings.HasSuffix(pattern, ".*"):
		prefix := strings.TrimSuffix(pattern, ".*")
		if node == prefix || strings.HasPrefix(node, prefix+".") {
			return 2 + len(prefix), true
		}
	}
	return 0, false
}
