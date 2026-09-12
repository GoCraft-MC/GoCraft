package nbt

import (
	"bytes"
	"testing"
)

func TestLinkComponentContainsJava1214OpenURLStyle(t *testing.T) {
	link := "https://permissions.example/permissions/token"
	c := DefaultTextLinkComponent("Open editor", link).
		Bytes()

	for _, expected := range []string{"text", "underlined", "clickEvent", "action", "open_url", "value", link} {
		if !bytes.Contains(c, []byte(expected)) {
			t.Errorf("link component is missing %q", expected)
		}
	}

	if c[0] != 0x0a || c[len(c)-1] != 0x00 {
		t.Fatalf("link component has invalid root compound framing: %x", c)
	}
}
