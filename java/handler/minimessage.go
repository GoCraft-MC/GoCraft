package handler

// MiniMessage parser — converts MiniMessage-formatted strings to Minecraft
// legacy §-coded strings (supported by all modern Java clients).
//
// Supported tags:
//
//	Named colors : <black> <dark_blue> <dark_green> <dark_aqua> <dark_red>
//	               <dark_purple> <gold> <gray> <dark_gray> <blue> <green>
//	               <aqua> <red> <light_purple> <yellow> <white>
//	Hex color    : <#RRGGBB>
//	Formatting   : <bold>/<b>  <italic>/<i>  <underlined>/<u>
//	               <strikethrough>/<st>  <obfuscated>/<obf>
//	Gradient     : <gradient:#RRGGBB:#RRGGBB>text</gradient>
//	               <gradient:#RRGGBB:#RRGGBB:#RRGGBB>text</gradient>  (multi-stop)
//	Reset        : <reset>/<r>
//	Close tags   : </red>  </bold>  etc. — restore previous state
//	Legacy codes : &0-9  &a-f  &k &l &m &n &o &r  (§ synonym)

import (
	"math"
	"strconv"
	"strings"
)

// MMOptions controls optional features of the MiniMessage parser.
type MMOptions struct {
	// Glyphs maps glyph names to their Unicode characters.
	// Enables <glyph:name> tags in templates.
	Glyphs map[string]string
}

// ParseMiniMessage converts a MiniMessage string to a §-coded string.
// Call EscapeMiniMessage on untrusted player input before embedding it
// inside a MiniMessage template.
func ParseMiniMessage(input string) string {
	return ParseMiniMessageWithOptions(input, MMOptions{})
}

// ParseMiniMessageWithOptions is like ParseMiniMessage but accepts extra options
// such as a glyph map for <glyph:name> tags.
func ParseMiniMessageWithOptions(input string, opts MMOptions) string {
	input = expandLegacyCodes(input)
	p := &mmParser{runes: []rune(input), glyphs: opts.Glyphs}
	p.stack = []mmState{{}}
	p.run()
	return p.buf.String()
}

// ParseMiniMessageBedrock parses a MiniMessage string and returns a §-coded
// string that is safe for Bedrock Edition chat. Gradients are collapsed to
// their first stop color; hex <#RRGGBB> colors are mapped to the nearest
// named §-color. Everything else behaves identically to ParseMiniMessage.
func ParseMiniMessageBedrock(input string) string {
	input = expandLegacyCodes(input)
	p := &mmParser{runes: []rune(input), bedrockSafe: true}
	p.stack = []mmState{{}}
	p.run()
	return p.buf.String()
}

// EscapeMiniMessage escapes '<' so the string is treated as plain text when
// embedded inside a MiniMessage template (prevents tag injection).
func EscapeMiniMessage(s string) string {
	return strings.ReplaceAll(s, "<", "\\<")
}

// ── state ─────────────────────────────────────────────────────────────────────

type mmState struct {
	color         string // §X or §x§... hex sequence, or ""
	bold          bool
	italic        bool
	underlined    bool
	strikethrough bool
	obfuscated    bool
}

// codes emits §r followed by all active formatting codes for this state.
func (s mmState) codes() string {
	var b strings.Builder
	b.WriteString("§r")
	if s.color != "" {
		b.WriteString(s.color)
	}
	if s.bold {
		b.WriteString("§l")
	}
	if s.italic {
		b.WriteString("§o")
	}
	if s.underlined {
		b.WriteString("§n")
	}
	if s.strikethrough {
		b.WriteString("§m")
	}
	if s.obfuscated {
		b.WriteString("§k")
	}
	return b.String()
}

// ── parser ────────────────────────────────────────────────────────────────────

type mmParser struct {
	runes       []rune
	pos         int
	stack       []mmState
	buf         strings.Builder
	glyphs      map[string]string
	bedrockSafe bool // when true: collapse gradients to first color, map hex to nearest named color
}

func (p *mmParser) cur() mmState {
	if len(p.stack) == 0 {
		return mmState{}
	}
	return p.stack[len(p.stack)-1]
}

func (p *mmParser) push(s mmState) { p.stack = append(p.stack, s) }

func (p *mmParser) pop() {
	if len(p.stack) > 1 {
		p.stack = p.stack[:len(p.stack)-1]
	}
}

func (p *mmParser) run() {
	for p.pos < len(p.runes) {
		r := p.runes[p.pos]
		// Escaped angle bracket
		if r == '\\' && p.pos+1 < len(p.runes) && p.runes[p.pos+1] == '<' {
			p.buf.WriteRune('<')
			p.pos += 2
			continue
		}
		if r == '<' {
			if tag, end, ok := p.peekTag(); ok {
				p.pos = end
				p.handleTag(tag)
				continue
			}
		}
		p.buf.WriteRune(r)
		p.pos++
	}
}

// peekTag reads a tag starting at p.pos.  Returns tag content (no brackets),
// position after '>', and true on success.
func (p *mmParser) peekTag() (string, int, bool) {
	if p.pos >= len(p.runes) || p.runes[p.pos] != '<' {
		return "", 0, false
	}
	end := p.pos + 1
	for end < len(p.runes) && p.runes[end] != '>' && p.runes[end] != '\n' {
		end++
	}
	if end >= len(p.runes) || p.runes[end] != '>' {
		return "", 0, false
	}
	tag := string(p.runes[p.pos+1 : end])
	if tag == "" {
		return "", 0, false
	}
	return tag, end + 1, true
}

func (p *mmParser) handleTag(raw string) {
	lower := strings.ToLower(strings.TrimSpace(raw))

	// ── close tag ─────────────────────────────────────────────────────────────
	if strings.HasPrefix(lower, "/") {
		p.pop()
		p.buf.WriteString(p.cur().codes())
		return
	}

	// ── glyph ─────────────────────────────────────────────────────────────────
	if strings.HasPrefix(lower, "glyph:") {
		name := strings.TrimPrefix(lower, "glyph:")
		if p.glyphs != nil {
			if ch, ok := p.glyphs[name]; ok {
				p.buf.WriteString(ch)
			}
		}
		return
	}

	// ── gradient ──────────────────────────────────────────────────────────────
	if strings.HasPrefix(lower, "gradient:") {
		stops := strings.Split(lower[9:], ":")
		colors := make([][3]uint8, 0, len(stops))
		for _, s := range stops {
			if rgb, ok := parseHexColor(s); ok {
				colors = append(colors, rgb)
			}
		}
		if len(colors) >= 2 {
			inner := p.collectUntil("/gradient")
			if p.bedrockSafe {
				// Collapse to the nearest named color for the first stop.
				code := nearestNamedColor(colors[0])
				next := p.cur()
				next.color = code
				p.push(next)
				p.buf.WriteString(code)
				p.buf.WriteString(inner)
				p.pop()
				p.buf.WriteString(p.cur().codes())
			} else {
				p.emitGradient(colors, inner)
			}
			return
		}
		// bad gradient: treat as unknown
		return
	}

	// ── reset ─────────────────────────────────────────────────────────────────
	if lower == "reset" || lower == "r" {
		p.push(mmState{})
		p.buf.WriteString("§r")
		return
	}

	// ── formatting ────────────────────────────────────────────────────────────
	next := p.cur()
	switch lower {
	case "bold", "b":
		next.bold = true
		p.push(next)
		p.buf.WriteString("§l")
	case "italic", "i", "em":
		next.italic = true
		p.push(next)
		p.buf.WriteString("§o")
	case "underlined", "u":
		next.underlined = true
		p.push(next)
		p.buf.WriteString("§n")
	case "strikethrough", "st":
		next.strikethrough = true
		p.push(next)
		p.buf.WriteString("§m")
	case "obfuscated", "obf":
		next.obfuscated = true
		p.push(next)
		p.buf.WriteString("§k")
	default:
		if code, ok := resolveMMColor(lower); ok {
			if p.bedrockSafe && strings.HasPrefix(lower, "#") {
				if rgb, ok2 := parseHexColor(lower); ok2 {
					code = nearestNamedColor(rgb)
				}
			}
			next.color = code
			p.push(next)
			p.buf.WriteString(code)
		}
		// unknown tag: skip silently
	}
}

// collectUntil reads runes from the current position until it encounters the
// given closing tag name (e.g. "/gradient"), then advances past the close tag.
func (p *mmParser) collectUntil(closeTag string) string {
	start := p.pos
	closeTag = strings.ToLower(closeTag)
	for p.pos < len(p.runes) {
		if p.runes[p.pos] == '<' {
			if tag, end, ok := p.peekTag(); ok {
				if strings.ToLower(strings.TrimSpace(tag)) == closeTag {
					text := string(p.runes[start:p.pos])
					p.pos = end
					return text
				}
			}
		}
		p.pos++
	}
	return string(p.runes[start:])
}

// ── gradient ──────────────────────────────────────────────────────────────────

func (p *mmParser) emitGradient(stops [][3]uint8, text string) {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return
	}
	segments := len(stops) - 1
	for i, r := range runes {
		t := float64(i) / math.Max(float64(n-1), 1)
		// Which segment does t fall in?
		seg := int(t * float64(segments))
		if seg >= segments {
			seg = segments - 1
		}
		local := t*float64(segments) - float64(seg)
		s0, s1 := stops[seg], stops[seg+1]
		cr := lerp(s0[0], s1[0], local)
		cg := lerp(s0[1], s1[1], local)
		cb := lerp(s0[2], s1[2], local)
		p.buf.WriteString(hexColorCode(cr, cg, cb))
		p.buf.WriteRune(r)
	}
}

func lerp(a, b uint8, t float64) uint8 {
	return uint8(math.Round(float64(a)*(1-t) + float64(b)*t))
}

// ── color helpers ──────────────────────────────────────────────────────────────

// resolveMMColor converts a MiniMessage color name or <#RRGGBB> to §-codes.
func resolveMMColor(s string) (string, bool) {
	switch s {
	case "black":
		return "§0", true
	case "dark_blue":
		return "§1", true
	case "dark_green":
		return "§2", true
	case "dark_aqua":
		return "§3", true
	case "dark_red":
		return "§4", true
	case "dark_purple":
		return "§5", true
	case "gold":
		return "§6", true
	case "gray":
		return "§7", true
	case "dark_gray":
		return "§8", true
	case "blue":
		return "§9", true
	case "green":
		return "§a", true
	case "aqua":
		return "§b", true
	case "red":
		return "§c", true
	case "light_purple":
		return "§d", true
	case "yellow":
		return "§e", true
	case "white":
		return "§f", true
	}
	if strings.HasPrefix(s, "#") {
		if rgb, ok := parseHexColor(s); ok {
			return hexColorCode(rgb[0], rgb[1], rgb[2]), true
		}
	}
	return "", false
}

// hexColorCode produces the §x§r₁§r₂§g₁§g₂§b₁§b₂ sequence (Minecraft 1.16+).
func hexColorCode(r, g, b uint8) string {
	h := make([]byte, 6)
	const digits = "0123456789abcdef"
	h[0] = digits[r>>4]
	h[1] = digits[r&0xF]
	h[2] = digits[g>>4]
	h[3] = digits[g&0xF]
	h[4] = digits[b>>4]
	h[5] = digits[b&0xF]
	return "§x§" + string(h[0]) + "§" + string(h[1]) +
		"§" + string(h[2]) + "§" + string(h[3]) +
		"§" + string(h[4]) + "§" + string(h[5])
}

func parseHexColor(s string) ([3]uint8, bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return [3]uint8{}, false
	}
	rv, e1 := strconv.ParseUint(s[0:2], 16, 8)
	gv, e2 := strconv.ParseUint(s[2:4], 16, 8)
	bv, e3 := strconv.ParseUint(s[4:6], 16, 8)
	if e1 != nil || e2 != nil || e3 != nil {
		return [3]uint8{}, false
	}
	return [3]uint8{uint8(rv), uint8(gv), uint8(bv)}, true
}

// nearestNamedColor maps an RGB value to the closest Minecraft named §-color
// code using Euclidean distance in RGB space. Used by the Bedrock-safe parser
// to replace hex and gradient colors with colors Bedrock can actually render.
func nearestNamedColor(rgb [3]uint8) string {
	named := [][4]int{
		{0x00, 0x00, 0x00, '0'}, // black
		{0x00, 0x00, 0xAA, '1'}, // dark_blue
		{0x00, 0xAA, 0x00, '2'}, // dark_green
		{0x00, 0xAA, 0xAA, '3'}, // dark_aqua
		{0xAA, 0x00, 0x00, '4'}, // dark_red
		{0xAA, 0x00, 0xAA, '5'}, // dark_purple
		{0xFF, 0xAA, 0x00, '6'}, // gold
		{0xAA, 0xAA, 0xAA, '7'}, // gray
		{0x55, 0x55, 0x55, '8'}, // dark_gray
		{0x55, 0x55, 0xFF, '9'}, // blue
		{0x55, 0xFF, 0x55, 'a'}, // green
		{0x55, 0xFF, 0xFF, 'b'}, // aqua
		{0xFF, 0x55, 0x55, 'c'}, // red
		{0xFF, 0x55, 0xFF, 'd'}, // light_purple
		{0xFF, 0xFF, 0x55, 'e'}, // yellow
		{0xFF, 0xFF, 0xFF, 'f'}, // white
	}
	best, bestDist := '7', int(1<<31-1)
	for _, c := range named {
		dr := int(rgb[0]) - c[0]
		dg := int(rgb[1]) - c[1]
		db := int(rgb[2]) - c[2]
		dist := dr*dr + dg*dg + db*db
		if dist < bestDist {
			bestDist = dist
			best = rune(c[3])
		}
	}
	return "§" + string(best)
}

// ── legacy & codes ────────────────────────────────────────────────────────────

func expandLegacyCodes(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		if runes[i] == '&' && i+1 < len(runes) && isLegacyCodeChar(runes[i+1]) {
			b.WriteRune('§')
			b.WriteRune(runes[i+1])
			i++
			continue
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}

func isLegacyCodeChar(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F') ||
		r == 'k' || r == 'K' ||
		r == 'l' || r == 'L' ||
		r == 'm' || r == 'M' ||
		r == 'n' || r == 'N' ||
		r == 'o' || r == 'O' ||
		r == 'r' || r == 'R'
}
