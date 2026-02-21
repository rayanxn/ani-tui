package nyaa

import (
	"sort"
	"strings"
	"unicode"
)

// parseTitleZones splits a torrent title into semantic zones:
// leading [group] tags, the core anime title, and trailing technical metadata.
func parseTitleZones(raw string) (groupTags []string, coreTitle string, techTags string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, "", ""
	}

	// Extract leading [...] as group tags.
	for strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end == -1 {
			break
		}
		tag := strings.ToLower(strings.TrimSpace(s[1:end]))
		if tag != "" {
			groupTags = append(groupTags, tag)
		}
		s = strings.TrimSpace(s[end+1:])
	}

	// Extract trailing [...] and (...) as tech tags (walk backwards).
	// We find the boundary where trailing metadata starts.
	trailingStart := len(s)
	for trailingStart > 0 {
		trimmed := strings.TrimSpace(s[:trailingStart])
		if trimmed == "" {
			break
		}
		last := trimmed[len(trimmed)-1]
		if last == ']' {
			open := strings.LastIndex(trimmed, "[")
			if open == -1 {
				break
			}
			trailingStart = open
		} else if last == ')' {
			open := strings.LastIndex(trimmed, "(")
			if open == -1 {
				break
			}
			trailingStart = open
		} else {
			break
		}
	}

	coreTitle = strings.TrimSpace(s[:trailingStart])
	techTags = strings.TrimSpace(s[trailingStart:])
	return groupTags, coreTitle, techTags
}

// tokenize lowercases a string and splits it into alphanumeric tokens.
func tokenize(s string) []string {
	lower := strings.ToLower(s)
	parts := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			tokens = append(tokens, p)
		}
	}
	return tokens
}

// scoreTitleMatch scores how well a torrent title matches the given alt titles.
// Returns a normalized score; higher is better.
func scoreTitleMatch(torrentTitle string, altTitles []string) float64 {
	groupTags, core, _ := parseTitleZones(torrentTitle)
	coreTokens := tokenize(core)

	coreSet := make(map[string]bool, len(coreTokens))
	for _, t := range coreTokens {
		coreSet[t] = true
	}

	groupSet := make(map[string]bool)
	for _, g := range groupTags {
		for _, t := range tokenize(g) {
			groupSet[t] = true
		}
	}

	var bestScore float64
	var bestCount int

	for _, alt := range altTitles {
		altTokens := tokenize(alt)
		if len(altTokens) == 0 {
			continue
		}

		var raw float64
		for _, at := range altTokens {
			if coreSet[at] {
				raw += 3.0
			} else if groupSet[at] {
				raw -= 5.0
			}
		}

		norm := raw / float64(len(altTokens))
		if norm > bestScore || bestCount == 0 {
			bestScore = norm
			bestCount = len(altTokens)
		}
	}

	// For single-token titles, require a strong match (the token must appear
	// in the core). The threshold of 1.5 alone handles this since a single
	// matched token gives 3.0/1 = 3.0, and a miss gives 0.0.
	// However, for truly short queries we also check that the token is a
	// standalone token in the core (tokenize already ensures word boundaries).

	return bestScore
}

const matchThreshold = 1.5

// FilterByTitle filters torrent items by relevance to the given alternative titles.
// Items whose core title doesn't sufficiently match any alt title are removed.
// If altTitles is empty, all items are returned unfiltered.
func FilterByTitle(items []Item, altTitles []string) []Item {
	if len(altTitles) == 0 {
		return items
	}

	// Remove empty alt titles.
	cleaned := make([]string, 0, len(altTitles))
	for _, t := range altTitles {
		if strings.TrimSpace(t) != "" {
			cleaned = append(cleaned, t)
		}
	}
	if len(cleaned) == 0 {
		return items
	}

	type scored struct {
		item  Item
		score float64
	}

	var kept []scored
	for _, it := range items {
		s := scoreTitleMatch(it.Title, cleaned)
		if s >= matchThreshold {
			kept = append(kept, scored{item: it, score: s})
		}
	}

	// Sort by score desc, then seeders desc, then downloads desc.
	sort.SliceStable(kept, func(i, j int) bool {
		if kept[i].score != kept[j].score {
			return kept[i].score > kept[j].score
		}
		if kept[i].item.Seeders != kept[j].item.Seeders {
			return kept[i].item.Seeders > kept[j].item.Seeders
		}
		return kept[i].item.Downloads > kept[j].item.Downloads
	})

	result := make([]Item, len(kept))
	for i, k := range kept {
		result[i] = k.item
	}
	return result
}
