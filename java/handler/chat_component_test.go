package handler

import (
	"bytes"
	"testing"
)

func TestLinkComponentContainsJava1214OpenURLStyle(t *testing.T) {
	link := "https://permissions.example/permissions/token"
	component := nbtLinkComponent("Open editor", link)
	for _, expected := range []string{"text", "underlined", "clickEvent", "action", "open_url", "value", link} {
		if !bytes.Contains(component, []byte(expected)) {
			t.Errorf("link component is missing %q", expected)
		}
	}
	if component[0] != 0x0a || component[len(component)-1] != 0x00 {
		t.Fatalf("link component has invalid root compound framing: %x", component)
	}
}
