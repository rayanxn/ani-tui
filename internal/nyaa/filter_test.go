package nyaa

import (
	"testing"
)

func TestParseTitleZones(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		wantGroup []string
		wantCore  string
		wantTech  string
	}{
		{
			name:      "full format with group and tech tags",
			raw:       "[Exiled-Destiny] Persona 4 The Animation 01-26 (Dual Audio) [BD 720p 8bit]",
			wantGroup: []string{"exiled-destiny"},
			wantCore:  "Persona 4 The Animation 01-26",
			wantTech:  "(Dual Audio) [BD 720p 8bit]",
		},
		{
			name:      "subsplease format",
			raw:       "[SubsPlease] Takt Op. Destiny - 01 (1080p) [hash123]",
			wantGroup: []string{"subsplease"},
			wantCore:  "Takt Op. Destiny - 01",
			wantTech:  "(1080p) [hash123]",
		},
		{
			name:      "no group tags",
			raw:       "My Anime Title (720p)",
			wantGroup: nil,
			wantCore:  "My Anime Title",
			wantTech:  "(720p)",
		},
		{
			name:      "no tech tags",
			raw:       "[Group] My Anime Title",
			wantGroup: []string{"group"},
			wantCore:  "My Anime Title",
			wantTech:  "",
		},
		{
			name:      "multiple group tags",
			raw:       "[Group1] [Group2] Title Here (1080p)",
			wantGroup: []string{"group1", "group2"},
			wantCore:  "Title Here",
			wantTech:  "(1080p)",
		},
		{
			name:      "empty string",
			raw:       "",
			wantGroup: nil,
			wantCore:  "",
			wantTech:  "",
		},
		{
			name:      "title only",
			raw:       "Just A Title",
			wantGroup: nil,
			wantCore:  "Just A Title",
			wantTech:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotGroup, gotCore, gotTech := parseTitleZones(tt.raw)

			if len(gotGroup) != len(tt.wantGroup) {
				t.Errorf("groupTags: got %v, want %v", gotGroup, tt.wantGroup)
			} else {
				for i := range gotGroup {
					if gotGroup[i] != tt.wantGroup[i] {
						t.Errorf("groupTags[%d]: got %q, want %q", i, gotGroup[i], tt.wantGroup[i])
					}
				}
			}

			if gotCore != tt.wantCore {
				t.Errorf("coreTitle: got %q, want %q", gotCore, tt.wantCore)
			}
			if gotTech != tt.wantTech {
				t.Errorf("techTags: got %q, want %q", gotTech, tt.wantTech)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{"normal title", "Takt Op. Destiny", []string{"takt", "op", "destiny"}},
		{"with numbers", "86 - Eighty Six", []string{"86", "eighty", "six"}},
		{"single char", "K", []string{"k"}},
		{"empty", "", nil},
		{"special chars", "Re:Zero - Starting Life", []string{"re", "zero", "starting", "life"}},
		{"mixed case", "ONE PIECE", []string{"one", "piece"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tokenize(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
			for i := range got {
				if got[i] != tt.expect[i] {
					t.Errorf("token[%d]: got %q, want %q", i, got[i], tt.expect[i])
				}
			}
		})
	}
}

func TestFilterByTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		altTitles  []string
		items      []Item
		wantTitles []string // expected titles in result, in order
	}{
		{
			name:      "destiny bug - false positive excluded",
			altTitles: []string{"Destiny", "Takt Op. Destiny"},
			items: []Item{
				{Title: "[Exiled-Destiny] Persona 4 The Animation 01-26 (Dual Audio) [BD 720p 8bit]", Seeders: 10},
				{Title: "[SubsPlease] Takt Op. Destiny - 01 (1080p) [ABC123]", Seeders: 50},
			},
			wantTitles: []string{"[SubsPlease] Takt Op. Destiny - 01 (1080p) [ABC123]"},
		},
		{
			name:      "english and romaji match",
			altTitles: []string{"Attack on Titan", "Shingeki no Kyojin"},
			items: []Item{
				{Title: "[SubsPlease] Shingeki no Kyojin - 05 (1080p) [hash]", Seeders: 100},
			},
			wantTitles: []string{"[SubsPlease] Shingeki no Kyojin - 05 (1080p) [hash]"},
		},
		{
			name:      "case insensitive",
			altTitles: []string{"One Piece"},
			items: []Item{
				{Title: "[Erai-raws] ONE PIECE - 1080 (1080p) [hash]", Seeders: 200},
			},
			wantTitles: []string{"[Erai-raws] ONE PIECE - 1080 (1080p) [hash]"},
		},
		{
			name:      "no substring false hit - K vs KonoSuba",
			altTitles: []string{"K"},
			items: []Item{
				{Title: "[Group] KonoSuba 01 (720p)", Seeders: 50},
			},
			wantTitles: nil,
		},
		{
			name:      "short title exact match",
			altTitles: []string{"K"},
			items: []Item{
				{Title: "[SubsPlease] K - 01 (1080p)", Seeders: 30},
			},
			wantTitles: []string{"[SubsPlease] K - 01 (1080p)"},
		},
		{
			name:      "short title 86",
			altTitles: []string{"86"},
			items: []Item{
				{Title: "[SubsPlease] 86 - Eighty Six - 03 (1080p) [hash]", Seeders: 80},
			},
			wantTitles: []string{"[SubsPlease] 86 - Eighty Six - 03 (1080p) [hash]"},
		},
		{
			name:      "empty alt titles - no filtering",
			altTitles: []string{},
			items: []Item{
				{Title: "[Group] Anything 01 (720p)", Seeders: 10},
				{Title: "[Group] Something Else 02 (1080p)", Seeders: 20},
			},
			wantTitles: []string{"[Group] Anything 01 (720p)", "[Group] Something Else 02 (1080p)"},
		},
		{
			name:      "no results pass - empty return",
			altTitles: []string{"Nonexistent"},
			items: []Item{
				{Title: "[Group] Something Else 01 (720p)", Seeders: 10},
			},
			wantTitles: nil,
		},
		{
			name:      "sorted by score then seeders",
			altTitles: []string{"Attack on Titan", "Shingeki no Kyojin"},
			items: []Item{
				{Title: "[GroupA] Shingeki no Kyojin - 05 (720p)", Seeders: 50},
				{Title: "[GroupB] Shingeki no Kyojin - 05 (1080p)", Seeders: 100},
			},
			wantTitles: []string{
				"[GroupB] Shingeki no Kyojin - 05 (1080p)",
				"[GroupA] Shingeki no Kyojin - 05 (720p)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FilterByTitle(tt.items, tt.altTitles)

			if len(got) != len(tt.wantTitles) {
				titles := make([]string, len(got))
				for i, g := range got {
					titles[i] = g.Title
				}
				t.Fatalf("got %d results %v, want %d %v", len(got), titles, len(tt.wantTitles), tt.wantTitles)
			}
			for i, g := range got {
				if g.Title != tt.wantTitles[i] {
					t.Errorf("result[%d]: got %q, want %q", i, g.Title, tt.wantTitles[i])
				}
			}
		})
	}
}
