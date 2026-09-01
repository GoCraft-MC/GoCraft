package command

import (
	"math"
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
	integerMin, integerMax := int64(10), int64(1)
	nan := math.NaN()
	tests := []struct {
		name string
		root *Root
		want string
	}{
		{
			name: "argument at root",
			root: &Root{Children: []Node{
				Argument{Name: "target", Type: ArgPlayer, Exec: 1},
			}},
			want: "root commands must be literals",
		},
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
		{
			name: "duplicate enum",
			root: &Root{Children: []Node{Literal{Name: "mode", Children: []Node{
				Argument{Name: "value", Type: ArgEnum, Enum: []string{"one", "one"}, Exec: 1},
			}}}},
			want: "duplicate value",
		},
		{
			name: "reversed integer range",
			root: &Root{Children: []Node{Literal{Name: "count", Children: []Node{
				Argument{Name: "value", Type: ArgInteger, IntegerMin: &integerMin, IntegerMax: &integerMax, Exec: 1},
			}}}},
			want: "minimum exceeds maximum",
		},
		{
			name: "nan decimal range",
			root: &Root{Children: []Node{Literal{Name: "price", Children: []Node{
				Argument{Name: "value", Type: ArgDecimal, DecimalMin: &nan, Exec: 1},
			}}}},
			want: "contains NaN",
		},
		{
			name: "missing custom type",
			root: &Root{Children: []Node{Literal{Name: "home", Children: []Node{
				Argument{Name: "name", Type: ArgCustom, Exec: 1},
			}}}},
			want: "has no type id",
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
