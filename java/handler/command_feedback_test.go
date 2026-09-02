package handler

import (
	"strings"
	"testing"

	"GoCraft/core/player"
)

// recorder stands in for both adapters. The bridges the server installs are the
// only thing that knows about editions, so a test needs no session, no socket
// and no listener to prove a command answered.
type recorder struct {
	messages map[string][]string
	links    []string
	synced   []string
}

func newRecorder() *recorder {
	return &recorder{messages: make(map[string][]string)}
}

func (r *recorder) send(target *player.Player, text string) error {
	name := "<console>"
	if target != nil {
		name = target.Username
	}
	r.messages[name] = append(r.messages[name], text)
	return nil
}

func (r *recorder) sendLink(_ *player.Player, text, link string) error {
	r.links = append(r.links, text+" -> "+link)
	return nil
}

func (r *recorder) sync(target *player.Player) {
	r.synced = append(r.synced, target.Username)
}

func (r *recorder) to(name string) string {
	return strings.Join(r.messages[name], "\n")
}

func wiredDispatcher(t *testing.T) (*Dispatcher, *recorder) {
	t.Helper()
	dispatcher := NewDispatcher()
	record := newRecorder()
	dispatcher.SetMessenger(record.send)
	dispatcher.SetLinkMessenger(record.sendLink)
	dispatcher.SetAbilitySync(record.sync)
	return dispatcher, record
}

func bedrockPlayer(name string) *player.Player {
	return player.New([16]byte{1}, name, player.ClientEditionBedrock)
}

// The bug this whole change exists for. These commands used to write straight
// to a *network.ClientConn, which a Bedrock player does not have, and every
// Java send helper treats a nil connection as a no-op — so they ran, changed
// state, and told the player nothing at all.
func TestBuiltinsAnswerABedrockPlayer(t *testing.T) {
	for _, testCase := range []struct {
		command string
		want    string
	}{
		{"/gamemode creative", "creative"},
		{"/walkspeed 0.3", "Walking speed"},
		{"/flyspeed 0.2", "Flying speed"},
		{"/fly", "Flight"},
		{"/help", "Commands:"},
	} {
		t.Run(testCase.command, func(t *testing.T) {
			dispatcher, record := wiredDispatcher(t)
			RegisterBuiltins(dispatcher)
			issuer := bedrockPlayer("steve")
			issuer.Operator = true

			dispatcher.Dispatch(testCase.command, CommandContext{Player: issuer})

			if got := record.to("steve"); !strings.Contains(got, testCase.want) {
				t.Fatalf("%s answered %q, want it to mention %q", testCase.command, got, testCase.want)
			}
		})
	}
}

// A handler that answered twice would be as visible as one that never answered.
// /fly used to hold both a deferred reply for one edition and an inline one for
// the other, and only avoided doubling because whichever edition the player was
// not on left its callback nil.
func TestFlyAnswersExactlyOnce(t *testing.T) {
	dispatcher, record := wiredDispatcher(t)
	RegisterBuiltins(dispatcher)
	issuer := bedrockPlayer("steve")
	issuer.Operator = true

	dispatcher.Dispatch("/fly", CommandContext{Player: issuer})

	if replies := record.messages["steve"]; len(replies) != 1 {
		t.Fatalf("/fly sent %d replies: %v", len(replies), replies)
	}
	if len(record.synced) != 1 {
		t.Fatalf("/fly synced abilities %d times", len(record.synced))
	}
}

// The usage error must not be accompanied by a flight report: nothing was
// toggled, and the deferred reply used to claim otherwise.
func TestFlyRejectsArgumentsWithoutReportingAState(t *testing.T) {
	dispatcher, record := wiredDispatcher(t)
	RegisterBuiltins(dispatcher)
	issuer := bedrockPlayer("steve")
	issuer.Operator = true

	dispatcher.Dispatch("/fly now", CommandContext{Player: issuer})

	got := record.to("steve")
	if !strings.Contains(got, "usage") {
		t.Fatalf("/fly now answered %q, want the usage error", got)
	}
	if strings.Contains(got, "Flight enabled") || strings.Contains(got, "Flight disabled") {
		t.Fatalf("/fly now answered %q, reporting a state it never changed", got)
	}
	if len(record.synced) != 0 {
		t.Fatalf("/fly now synced abilities after refusing to run")
	}
}

// Changing game mode has to republish it. On Java that is a Game Event plus
// Player Abilities, on Bedrock SetPlayerGameType plus UpdateAbilities; the
// handler must know neither.
func TestGameModeRepublishesPlayerState(t *testing.T) {
	dispatcher, record := wiredDispatcher(t)
	RegisterBuiltins(dispatcher)
	issuer := bedrockPlayer("steve")
	issuer.Operator = true

	dispatcher.Dispatch("/gamemode creative", CommandContext{Player: issuer})

	if issuer.GameMode != player.GameModeCreative {
		t.Fatalf("game mode = %v", issuer.GameMode)
	}
	if len(record.synced) != 1 || record.synced[0] != "steve" {
		t.Fatalf("ability sync = %v, want one for the issuer", record.synced)
	}
}

func TestUnknownCommandAndPermissionDenialReachBedrock(t *testing.T) {
	dispatcher, record := wiredDispatcher(t)
	RegisterBuiltins(dispatcher)
	issuer := bedrockPlayer("steve")

	dispatcher.Dispatch("/nosuchthing", CommandContext{Player: issuer})
	if got := record.to("steve"); !strings.Contains(got, "Unknown command") {
		t.Fatalf("unknown command answered %q", got)
	}

	record.messages = make(map[string][]string)
	dispatcher.Dispatch("/gamemode creative", CommandContext{Player: issuer})
	if got := record.to("steve"); !strings.Contains(got, "permission") {
		t.Fatalf("denied command answered %q", got)
	}
}

// A caller that supplied its own callback keeps it. That is what lets a test
// drive one handler with its own recorder, and it is why every field is filled
// only when nil.
func TestDispatchKeepsACallerSuppliedReply(t *testing.T) {
	dispatcher, record := wiredDispatcher(t)
	dispatcher.Register("ping", func(ctx CommandContext) error {
		return sendCommandMessage(ctx, "pong")
	})
	var own string

	dispatcher.Dispatch("/ping", CommandContext{
		Player: bedrockPlayer("steve"),
		Reply:  func(text string) error { own = text; return nil },
	})

	if own != "pong" {
		t.Fatalf("caller reply = %q", own)
	}
	if got := record.to("steve"); got != "" {
		t.Fatalf("bridge also answered %q, want the caller's reply to win", got)
	}
}

// With no link bridge installed the URL must still arrive as text, rather than
// the command appearing to do nothing.
func TestReplyLinkFallsBackToText(t *testing.T) {
	dispatcher := NewDispatcher()
	record := newRecorder()
	dispatcher.SetMessenger(record.send)
	dispatcher.Register("editor", func(ctx CommandContext) error {
		return ctx.ReplyLink("Open the editor:", "https://example.invalid/e")
	})

	dispatcher.Dispatch("/editor", CommandContext{Player: bedrockPlayer("steve")})

	if got := record.to("steve"); !strings.Contains(got, "https://example.invalid/e") {
		t.Fatalf("link fallback answered %q, want the URL in the text", got)
	}
}

func TestReplyLinkPrefersTheLinkBridge(t *testing.T) {
	dispatcher, record := wiredDispatcher(t)
	dispatcher.Register("editor", func(ctx CommandContext) error {
		return ctx.ReplyLink("Open the editor:", "https://example.invalid/e")
	})

	dispatcher.Dispatch("/editor", CommandContext{Player: bedrockPlayer("steve")})

	if len(record.links) != 1 {
		t.Fatalf("links = %v, want the bridge used", record.links)
	}
	if got := record.to("steve"); got != "" {
		t.Fatalf("link also went out as plain text: %q", got)
	}
}

// /heal notifies the player it healed, who may well be on the other edition
// from whoever ran it. That used to go to the target's Java connection, so a
// Bedrock target was healed in silence.
func TestReplyToReachesTheOtherEdition(t *testing.T) {
	dispatcher, record := wiredDispatcher(t)
	target := player.New([16]byte{2}, "alex", player.ClientEditionBedrock)
	dispatcher.Register("notify", func(ctx CommandContext) error {
		return sendCommandMessageTo(ctx, target, "you were healed")
	})

	dispatcher.Dispatch("/notify", CommandContext{Player: bedrockPlayer("steve")})

	if got := record.to("alex"); got != "you were healed" {
		t.Fatalf("target received %q", got)
	}
}
