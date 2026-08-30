package server

import (
	"fmt"
	"strconv"
	"strings"

	corepermission "GoCraft/core/permission"
	"GoCraft/java/handler"
)

func (s *Server) registerPermissionCommands() {
	s.cmds.RegisterOperator("gocraft", func(ctx handler.CommandContext) error {
		message, err := s.executePermissionCommand(ctx.Args)
		if err != nil {
			return err
		}
		if len(ctx.Args) == 1 && strings.EqualFold(ctx.Args[0], "peditor") && ctx.Reply == nil {
			link := strings.TrimPrefix(message, "Open the permission editor: ")
			return ctx.ReplyLink(message, link)
		}
		return commandReply(ctx, message)
	})
}

func (s *Server) executePermissionCommand(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf(gocraftUsage)
	}

	switch strings.ToLower(args[0]) {

	// ── /gocraft peditor ────────────────────────────────────────────────────────
	case "peditor":
		if s.permissionEditor == nil {
			return "", fmt.Errorf("permission editor is disabled in server.yml")
		}
		link, err := s.permissionEditor.create(s.cmds.CommandPermissions())
		if err != nil {
			return "", fmt.Errorf("creating permission editor: %w", err)
		}
		return "Open the permission editor: " + link, nil

	// ── /gocraft applyedits <key> ───────────────────────────────────────────────
	case "applyedits":
		if s.permissionEditor == nil {
			return "", fmt.Errorf("permission editor is disabled in server.yml")
		}
		if len(args) < 2 {
			return "", fmt.Errorf("usage: /gocraft applyedits <link-or-code>")
		}
		if err := s.permissionEditor.apply(args[1]); err != nil {
			return "", fmt.Errorf("applying permission edits: %w", err)
		}
		s.syncCommandPermissionsToAll()
		return "Permission edits applied and saved to permissions.json", nil

	// ── /gocraft user … ─────────────────────────────────────────────────────────
	case "user":
		return s.executeUserCommand(args[1:])

	// ── /gocraft group … ────────────────────────────────────────────────────────
	case "group":
		return s.executeGroupCommand(args[1:])

	default:
		return "", fmt.Errorf(gocraftUsage)
	}
}

// ── user sub-commands ──────────────────────────────────────────────────────────

func (s *Server) executeUserCommand(args []string) (string, error) {
	if len(args) < 3 || !strings.EqualFold(args[1], "group") {
		return "", fmt.Errorf("usage: /gocraft user <player> group <set|remove|list> [group]")
	}
	username := corepermission.Normalize(args[0])
	sub := strings.ToLower(args[2])

	doc := s.permissions.Snapshot()
	user := doc.Users[username]

	switch sub {
	case "set":
		if len(args) < 4 {
			return "", fmt.Errorf("usage: /gocraft user <player> group set <group>")
		}
		groupName := corepermission.Normalize(args[3])
		if _, ok := doc.Groups[groupName]; !ok {
			return "", fmt.Errorf("group %q does not exist", groupName)
		}
		if !containsString(user.Groups, groupName) {
			user.Groups = append(user.Groups, groupName)
		}
		doc.Users[username] = user
		if err := s.permissions.Replace(doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Added %s to group %s", username, groupName), nil

	case "remove":
		if len(args) < 4 {
			return "", fmt.Errorf("usage: /gocraft user <player> group remove <group>")
		}
		groupName := corepermission.Normalize(args[3])
		user.Groups = removeString(user.Groups, groupName)
		doc.Users[username] = user
		if err := s.permissions.Replace(doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Removed %s from group %s", username, groupName), nil

	case "list":
		if len(user.Groups) == 0 {
			return fmt.Sprintf("%s is in: default (only)", username), nil
		}
		return fmt.Sprintf("%s is in: default, %s", username, strings.Join(user.Groups, ", ")), nil

	default:
		return "", fmt.Errorf("usage: /gocraft user <player> group <set|remove|list> [group]")
	}
}

// ── group sub-commands ─────────────────────────────────────────────────────────

func (s *Server) executeGroupCommand(args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: /gocraft group <name> <create|delete|setprefix|setweight|addparent|removeparent>")
	}
	groupName := corepermission.Normalize(args[0])
	sub := strings.ToLower(args[1])

	doc := s.permissions.Snapshot()

	switch sub {
	case "create":
		if _, ok := doc.Groups[groupName]; ok {
			return "", fmt.Errorf("group %q already exists", groupName)
		}
		doc.Groups[groupName] = corepermission.Group{
			Parents:     []string{"default"},
			Permissions: map[string]bool{},
		}
		if err := s.permissions.Replace(doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Created group %s", groupName), nil

	case "delete":
		if groupName == "default" {
			return "", fmt.Errorf("cannot delete the default group")
		}
		if _, ok := doc.Groups[groupName]; !ok {
			return "", fmt.Errorf("group %q does not exist", groupName)
		}
		delete(doc.Groups, groupName)
		// Remove the deleted group from every user's group list.
		for uname, u := range doc.Users {
			u.Groups = removeString(u.Groups, groupName)
			doc.Users[uname] = u
		}
		// Remove the deleted group from every other group's parents.
		for gname, g := range doc.Groups {
			g.Parents = removeString(g.Parents, groupName)
			doc.Groups[gname] = g
		}
		if err := s.permissions.Replace(doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted group %s", groupName), nil

	case "setprefix":
		if _, ok := doc.Groups[groupName]; !ok {
			return "", fmt.Errorf("group %q does not exist", groupName)
		}
		prefix := ""
		if len(args) >= 3 {
			prefix = strings.Join(args[2:], " ")
		}
		g := doc.Groups[groupName]
		g.Prefix = prefix
		doc.Groups[groupName] = g
		if err := s.permissions.Replace(doc); err != nil {
			return "", err
		}
		if prefix == "" {
			return fmt.Sprintf("Cleared prefix for group %s", groupName), nil
		}
		return fmt.Sprintf("Set prefix for group %s to: %s", groupName, prefix), nil

	case "setweight":
		if len(args) < 3 {
			return "", fmt.Errorf("usage: /gocraft group <name> setweight <number>")
		}
		if _, ok := doc.Groups[groupName]; !ok {
			return "", fmt.Errorf("group %q does not exist", groupName)
		}
		w, err := strconv.Atoi(args[2])
		if err != nil {
			return "", fmt.Errorf("weight must be an integer, got %q", args[2])
		}
		g := doc.Groups[groupName]
		g.Weight = w
		doc.Groups[groupName] = g
		if err := s.permissions.Replace(doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Set weight for group %s to %d", groupName, w), nil

	case "addparent":
		if len(args) < 3 {
			return "", fmt.Errorf("usage: /gocraft group <name> addparent <parent>")
		}
		if _, ok := doc.Groups[groupName]; !ok {
			return "", fmt.Errorf("group %q does not exist", groupName)
		}
		parent := corepermission.Normalize(args[2])
		if _, ok := doc.Groups[parent]; !ok {
			return "", fmt.Errorf("parent group %q does not exist", parent)
		}
		g := doc.Groups[groupName]
		if !containsString(g.Parents, parent) {
			g.Parents = append(g.Parents, parent)
		}
		doc.Groups[groupName] = g
		if err := s.permissions.Replace(doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Added parent %s to group %s", parent, groupName), nil

	case "removeparent":
		if len(args) < 3 {
			return "", fmt.Errorf("usage: /gocraft group <name> removeparent <parent>")
		}
		if _, ok := doc.Groups[groupName]; !ok {
			return "", fmt.Errorf("group %q does not exist", groupName)
		}
		parent := corepermission.Normalize(args[2])
		g := doc.Groups[groupName]
		g.Parents = removeString(g.Parents, parent)
		doc.Groups[groupName] = g
		if err := s.permissions.Replace(doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("Removed parent %s from group %s", parent, groupName), nil

	default:
		return "", fmt.Errorf("usage: /gocraft group <name> <create|delete|setprefix|setweight|addparent|removeparent>")
	}
}

// ── helpers ────────────────────────────────────────────────────────────────────

const gocraftUsage = "usage: /gocraft <peditor|applyedits|user|group>"

func (s *Server) syncCommandPermissionsToAll() {
	if s.sessions == nil {
		return
	}
	for _, online := range s.sessions.SnapshotAll() {
		_ = handler.SyncCommandPermissions(online.Conn, online.Player, s.cmds)
	}
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	out := slice[:0:0]
	for _, v := range slice {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}
