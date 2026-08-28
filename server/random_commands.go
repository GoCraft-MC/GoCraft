package server

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"GoCraft/java/handler"
)

func parseRandomRange(value string) (int64, int64, error) {
	parts := strings.Split(value, "..")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("range must use <minimum>..<maximum>")
	}
	minimum, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minimum: %s", parts[0])
	}
	maximum, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil || maximum < minimum {
		return 0, 0, fmt.Errorf("maximum must be at least the minimum")
	}
	return minimum, maximum, nil
}

func (s *Server) registerRandomCommand() {
	s.cmds.RegisterOperator("random", func(ctx handler.CommandContext) error {
		if len(ctx.Args) != 2 {
			return fmt.Errorf("usage: /random <value|roll> <minimum>..<maximum>")
		}
		mode := strings.ToLower(ctx.Args[0])
		if mode != "value" && mode != "roll" {
			return fmt.Errorf("usage: /random <value|roll> <minimum>..<maximum>")
		}
		minimum, maximum, err := parseRandomRange(ctx.Args[1])
		if err != nil {
			return err
		}
		choice, err := rand.Int(rand.Reader, big.NewInt(maximum-minimum+1))
		if err != nil {
			return fmt.Errorf("generating random value: %w", err)
		}
		result := choice.Int64() + minimum
		if mode == "roll" {
			name := "Server"
			if ctx.Player != nil {
				name = ctx.Player.Username
			}
			s.broadcastMessage(fmt.Sprintf("%s rolled %d (%d..%d)", name, result, minimum, maximum))
			return nil
		}
		return commandReply(ctx, fmt.Sprintf("Random value: %d", result))
	})
}
