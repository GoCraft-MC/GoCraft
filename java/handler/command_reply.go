package handler

import "GoCraft/core/player"

// fillFeedback completes the callbacks a handler uses to answer, from the
// edition-neutral bridges the dispatcher holds.
//
// A callback the caller already supplied is kept. That is what lets a test
// drive a command with its own recorder and no server at all, and it is why
// every one of these is checked for nil rather than overwritten.
func fillFeedback(
	ctx *CommandContext,
	messenger func(*player.Player, string) error,
	linkMessenger func(*player.Player, string, string) error,
	syncAbilities func(*player.Player),
) {
	if ctx.Reply == nil && messenger != nil {
		issuer := ctx.Player
		ctx.Reply = func(text string) error { return messenger(issuer, text) }
	}
	if ctx.ReplyTo == nil && messenger != nil {
		ctx.ReplyTo = messenger
	}
	if ctx.ReplyLink == nil {
		ctx.ReplyLink = linkReplyFor(ctx, linkMessenger)
	}
	if ctx.SyncAbilities == nil && syncAbilities != nil {
		ctx.SyncAbilities = syncAbilities
	}
}

// linkReplyFor prefers the link bridge and falls back to plain text.
//
// The fallback matters more than it looks: a player who cannot be sent a
// clickable component must still be told the URL, and the alternative — sending
// nothing — is how the permission editor became invisible on Bedrock.
func linkReplyFor(
	ctx *CommandContext,
	linkMessenger func(*player.Player, string, string) error,
) func(string, string) error {
	issuer := ctx.Player
	if linkMessenger != nil {
		return func(text, link string) error { return linkMessenger(issuer, text, link) }
	}
	reply := ctx.Reply
	if reply == nil {
		return nil
	}
	return func(text, link string) error { return reply(text + " " + link) }
}

// sendCommandMessage is the one way a command answers its issuer.
//
// Handlers must not reach past it for a connection: on Bedrock there is none,
// and every Java send helper silently ignores a nil one, so a command that did
// would appear to work while telling that player nothing.
func sendCommandMessage(ctx CommandContext, text string) error {
	if ctx.Reply != nil {
		return ctx.Reply(text)
	}
	// Kept until every caller stops supplying a connection, so this commit
	// changes what is available to a handler without changing what any of
	// them currently do.
	return sendSystemMessage(ctx.Conn, text)
}

// sendCommandMessageTo answers someone other than the issuer — the target of a
// /heal or a /tphere, who may be on the other edition.
func sendCommandMessageTo(ctx CommandContext, target *player.Player, text string) error {
	if ctx.ReplyTo == nil || target == nil {
		return nil
	}
	return ctx.ReplyTo(target, text)
}

// syncPlayerState republishes the issuing player's game mode, flight and speeds
// after a command changed them.
//
// Java sends a Game Event plus Player Abilities, Bedrock sends
// SetPlayerGameType plus UpdateAbilities; neither is reachable from a command
// handler, and a handler that picked one silently did nothing for half the
// players on the server.
func syncPlayerState(ctx CommandContext) {
	if ctx.SyncAbilities == nil || ctx.Player == nil {
		return
	}
	ctx.SyncAbilities(ctx.Player)
}
