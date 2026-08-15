// Command genvanillatrades converts Mojang's decompiled 1.21.4
// VillagerTrades.java into GoCraft's deterministic profession/level catalogue.
//
// It intentionally selects the first two usable entries from every vanilla
// level pool. Vanilla selects two random entries when the villager acquires a
// level; deterministic selection gives GoCraft stable offers until entity NBT
// persistence includes the complete MerchantOffers list.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type offer struct {
	in1, in2, out      string
	in1Count, in2Count int
	outCount, max, xp  int
	price              float64
	tier               int
}

var itemPattern = regexp.MustCompile(`(?:Items|Blocks)\.([A-Z0-9_]+)`)

func main() {
	source := flag.String("source", "", "decompiled Mojang VillagerTrades.java")
	output := flag.String("output", "java/handler/trade_catalog_generated.go", "generated Go output")
	flag.Parse()
	if *source == "" {
		fatalf("-source is required")
	}
	raw, err := os.ReadFile(*source)
	if err != nil {
		fatalf("read source: %v", err)
	}
	catalog := parseCatalog(string(raw))
	formatted, err := format.Source(render(catalog))
	if err != nil {
		fatalf("format output: %v", err)
	}
	if err := os.WriteFile(*output, formatted, 0o644); err != nil {
		fatalf("write output: %v", err)
	}
}

func parseCatalog(source string) map[string]map[int][]offer {
	professionPattern := regexp.MustCompile(`\$\$0\.put\(VillagerProfession\.([A-Z]+),`)
	matches := professionPattern.FindAllStringSubmatchIndex(source, -1)
	result := make(map[string]map[int][]offer, len(matches))
	for index, match := range matches {
		profession := source[match[2]:match[3]]
		end := len(source)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		segment := source[match[1]:end]
		levels := make(map[int][]offer, 5)
		for level := 1; level <= 5; level++ {
			marker := fmt.Sprintf("(Object)%d, (Object)new ItemListing[]{", level)
			at := strings.Index(segment, marker)
			if at < 0 {
				continue
			}
			open := at + len(marker) - 1
			close := matching(segment, open, '{', '}')
			if close < 0 {
				fatalf("unclosed level %d pool for %s", level, profession)
			}
			for _, expression := range splitTopLevel(segment[open+1 : close]) {
				parsed, ok := parseOffer(expression, level-1)
				if ok {
					levels[level] = append(levels[level], parsed)
					if len(levels[level]) == 2 {
						break
					}
				}
			}
		}
		result[profession] = levels
	}
	return result
}

func parseOffer(expression string, tier int) (offer, bool) {
	expression = strings.TrimSpace(expression)
	if !strings.HasPrefix(expression, "new ") {
		return offer{}, false
	}
	nameEnd := strings.IndexByte(expression, '(')
	if nameEnd < 0 {
		return offer{}, false
	}
	class := strings.TrimSpace(strings.TrimPrefix(expression[:nameEnd], "new "))
	close := matching(expression, nameEnd, '(', ')')
	if close < 0 {
		return offer{}, false
	}
	args := splitTopLevel(expression[nameEnd+1 : close])
	base := offer{max: 12, price: 0.05, tier: tier}
	switch class {
	case "EmeraldForItems":
		if len(args) < 4 {
			return offer{}, false
		}
		base.in1, base.in1Count = item(args[0]), integer(args[1])
		base.out, base.outCount = "minecraft:emerald", 1
		base.max, base.xp = integer(args[2]), integer(args[3])
		if len(args) >= 5 {
			base.outCount = integer(args[4])
		}
	case "ItemsForEmeralds":
		if len(args) < 4 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:emerald", integer(args[1])
		base.out, base.outCount = item(args[0]), integer(args[2])
		if len(args) == 4 {
			base.xp = integer(args[3])
		} else {
			base.max, base.xp = integer(args[3]), integer(args[4])
		}
		if len(args) >= 6 {
			base.price = number(args[5])
		}
	case "EnchantedItemForEmeralds":
		if len(args) < 4 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:emerald", integer(args[1])
		base.out, base.outCount = item(args[0]), 1
		base.max, base.xp = integer(args[2]), integer(args[3])
		if len(args) >= 5 {
			base.price = number(args[4])
		}
	case "DyedArmorForEmeralds":
		if len(args) < 2 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:emerald", integer(args[1])
		base.out, base.outCount = item(args[0]), 1
		base.max, base.xp, base.price = 12, 1, 0.2
		if len(args) >= 4 {
			base.max, base.xp = integer(args[2]), integer(args[3])
		}
	case "ItemsAndEmeraldsToItems":
		if len(args) < 8 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:emerald", integer(args[2])
		base.in2, base.in2Count = item(args[0]), integer(args[1])
		base.out, base.outCount = item(args[3]), integer(args[4])
		base.max, base.xp, base.price = integer(args[5]), integer(args[6]), number(args[7])
	case "SuspiciousStewForEmerald":
		if len(args) < 3 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:emerald", 1
		base.out, base.outCount = "minecraft:suspicious_stew", 1
		base.xp = integer(args[len(args)-1])
	case "EnchantBookForEmeralds":
		if len(args) < 1 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:emerald", 10
		base.in2, base.in2Count = "minecraft:book", 1
		base.out, base.outCount = "minecraft:enchanted_book", 1
		base.xp, base.price = integer(args[0]), 0.2
	case "TreasureMapForEmeralds":
		if len(args) < 6 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:emerald", integer(args[0])
		base.in2, base.in2Count = "minecraft:compass", 1
		base.out, base.outCount = "minecraft:filled_map", 1
		base.max, base.xp, base.price = integer(args[4]), integer(args[5]), 0.2
	case "TippedArrowForItemsAndEmeralds":
		if len(args) < 7 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:emerald", integer(args[4])
		base.in2, base.in2Count = item(args[0]), integer(args[1])
		base.out, base.outCount = item(args[2]), integer(args[3])
		base.max, base.xp = integer(args[5]), integer(args[6])
	case "EmeraldsForVillagerTypeItem":
		if len(args) < 3 {
			return offer{}, false
		}
		base.in1, base.in1Count = "minecraft:oak_boat", integer(args[0])
		base.out, base.outCount = "minecraft:emerald", 1
		base.max, base.xp = integer(args[1]), integer(args[2])
	default:
		return offer{}, false
	}
	if base.in1 == "" || base.out == "" || base.in1Count <= 0 || base.outCount <= 0 || base.max <= 0 {
		return offer{}, false
	}
	return base, true
}

func splitTopLevel(value string) []string {
	var values []string
	start, parens, braces, brackets := 0, 0, 0, 0
	for index, current := range value {
		switch current {
		case '(':
			parens++
		case ')':
			parens--
		case '{':
			braces++
		case '}':
			braces--
		case '[':
			brackets++
		case ']':
			brackets--
		case ',':
			if parens == 0 && braces == 0 && brackets == 0 {
				values = append(values, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	if tail := strings.TrimSpace(value[start:]); tail != "" {
		values = append(values, tail)
	}
	return values
}

func matching(value string, open int, left, right byte) int {
	depth := 0
	for index := open; index < len(value); index++ {
		switch value[index] {
		case left:
			depth++
		case right:
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func item(value string) string {
	match := itemPattern.FindStringSubmatch(value)
	if len(match) != 2 {
		return ""
	}
	return "minecraft:" + strings.ToLower(match[1])
}

func integer(value string) int {
	match := regexp.MustCompile(`-?\d+`).FindString(value)
	parsed, _ := strconv.Atoi(match)
	return parsed
}

func number(value string) float64 {
	match := regexp.MustCompile(`-?(?:\d+\.?\d*|\.\d+)`).FindString(value)
	parsed, _ := strconv.ParseFloat(match, 64)
	return parsed
}

func render(catalog map[string]map[int][]offer) []byte {
	var output bytes.Buffer
	output.WriteString("// Code generated by genvanillatrades from Mojang 1.21.4 VillagerTrades; DO NOT EDIT.\n")
	output.WriteString("package handler\n\nimport corentity \"GoCraft/core/entity\"\n\n")
	output.WriteString("var vanillaVillagerTradeCatalog = map[corentity.VillagerProfession]map[int32][]tradeOffer{\n")
	professions := make([]string, 0, len(catalog))
	for profession := range catalog {
		professions = append(professions, profession)
	}
	sort.Strings(professions)
	for _, profession := range professions {
		fmt.Fprintf(&output, "corentity.VillagerProfession%s: {\n", goName(profession))
		for level := 1; level <= 5; level++ {
			fmt.Fprintf(&output, "%d: {\n", level)
			for _, current := range catalog[profession][level] {
				fmt.Fprintf(&output, "{input1: tradeItem{%q, %d}, input2: tradeItem{%q, %d}, output: tradeItem{%q, %d}, maxUses: %d, xpPerTrade: %d, tier: %d, priceMultiplier: %g},\n",
					current.in1, current.in1Count, current.in2, current.in2Count, current.out, current.outCount,
					current.max, current.xp, current.tier, current.price)
			}
			output.WriteString("},\n")
		}
		output.WriteString("},\n")
	}
	output.WriteString("}\n")
	return output.Bytes()
}

func goName(value string) string {
	parts := strings.Split(strings.ToLower(value), "_")
	for index := range parts {
		parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
	}
	return strings.Join(parts, "")
}

func fatalf(format string, values ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}
