package command

import (
	"testing"
)

// A tree that survives a trip through the wire and back is a tree the host will
// read as the build meant it. Round-tripping is the only check worth making:
// asserting bytes would pin the encoding rather than the meaning.
func TestEncodeTreeRoundTrips(t *testing.T) {
	minimum, maximum := int64(1), int64(64)
	floor := 0.01
	root := Root{Children: []Node{Literal{
		Name: "shop", Permission: "shop.use", Exec: 1, Children: []Node{
			Literal{Name: "sell", Children: []Node{
				Argument{Name: "price", Type: ArgDecimal, DecimalMin: &floor, Exec: 2},
			}},
			Literal{Name: "give", Children: []Node{
				Argument{Name: "player", Type: ArgPlayer, Children: []Node{
					Argument{
						Name: "count", Type: ArgInteger,
						IntegerMin: &minimum, IntegerMax: &maximum, Exec: 3,
					},
				}},
			}},
			Literal{Name: "mode", Children: []Node{
				Argument{Name: "value", Type: ArgEnum, Enum: []string{"buy", "sell"}, Exec: 4},
			}},
			Literal{Name: "home", Children: []Node{
				Argument{Name: "which", Type: ArgCustom, CustomType: "fr.oreo/home", Exec: 5},
			}},
		},
	}}}

	encoded, err := EncodeTree(root)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeTree(encoded)
	if err != nil {
		t.Fatal(err)
	}

	shop := decoded.Children[0].(Literal)
	if shop.Permission != "shop.use" || shop.Exec != 1 {
		t.Fatalf("shop = %+v", shop)
	}
	price := shop.Children[0].(Literal).Children[0].(Argument)
	if price.DecimalMin == nil || *price.DecimalMin != 0.01 || price.DecimalMax != nil {
		t.Fatalf("price bounds = %+v", price)
	}
	count := shop.Children[1].(Literal).Children[0].(Argument).Children[0].(Argument)
	if count.IntegerMin == nil || *count.IntegerMin != 1 || *count.IntegerMax != 64 {
		t.Fatalf("count bounds = %+v", count)
	}
	mode := shop.Children[2].(Literal).Children[0].(Argument)
	if len(mode.Enum) != 2 || mode.Enum[0] != "buy" {
		t.Fatalf("mode enum = %+v", mode)
	}
	home := shop.Children[3].(Literal).Children[0].(Argument)
	if home.CustomType != "fr.oreo/home" {
		t.Fatalf("home custom type = %q", home.CustomType)
	}
}

// A build that could ship a tree the host refuses would move the failure from
// the machine that has the source to the one that does not.
func TestEncodeTreeRefusesWhatTheHostWould(t *testing.T) {
	cases := map[string]Root{
		"leaf runs nothing": {Children: []Node{Literal{Name: "shop", Children: []Node{
			Literal{Name: "sell"},
		}}}},
		"argument at the top": {Children: []Node{Argument{Name: "price", Type: ArgDecimal, Exec: 1}}},
		"duplicate sibling": {Children: []Node{Literal{Name: "shop", Children: []Node{
			Literal{Name: "sell", Exec: 1}, Literal{Name: "sell", Exec: 2},
		}}}},
		"empty": {},
	}
	for name, root := range cases {
		if _, err := EncodeTree(root); err == nil {
			t.Fatalf("%s: encoded", name)
		}
	}
}
