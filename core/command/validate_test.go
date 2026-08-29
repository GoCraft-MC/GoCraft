package command

import (
	"strings"
	"testing"
)

func TestValidateAcceptsTypedCommandTree(t *testing.T) {
	root := &Root{Children: []Node{
		Literal{Name: "shop", Permission: "shop.use", Children: []Node{
			Literal{Name: "sell", Children: []Node{
				Argument{Name: "price", Type: ArgDecimal, Exec: 1},
			}},
		}},
		Literal{Name: "list", Exec: 2},
	}}
	if err := Validate(root); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsInvalidTrees(t *testing.T) {
	tests := []struct {
		name string
		root *Root
		want string
	}{
		{
			name: "duplicate literal",
			root: &Root{Children: []Node{
				Literal{Name: "shop", Exec: 1}, Literal{Name: "shop", Exec: 2},
			}},
			want: "duplicate literal",
		},
		{
			name: "greedy child",
			root: &Root{Children: []Node{Literal{Name: "say", Children: []Node{
				Argument{Name: "message", Type: ArgGreedy, Children: []Node{Literal{Name: "later", Exec: 1}}},
			}}}},
			want: "greedy argument must be last",
		},
		{
			name: "executable parent",
			root: &Root{Children: []Node{Literal{Name: "shop", Exec: 1, Children: []Node{
				Literal{Name: "sell", Exec: 2},
			}}}},
			want: "executable node has children",
		},
		{
			name: "empty leaf",
			root: &Root{Children: []Node{Literal{Name: "shop"}}},
			want: "leaf has no executor",
		},
		{
			name: "empty enum",
			root: &Root{Children: []Node{Literal{Name: "mode", Children: []Node{
				Argument{Name: "value", Type: ArgEnum, Exec: 1},
			}}}},
			want: "enum has no values",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
		})
	}
}
