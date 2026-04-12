package main

import (
	"encoding/xml"
	"testing"
)

func TestFilterNewPapers(t *testing.T) {
	cases := []struct {
		name    string
		papers  []Paper
		seen    map[string]bool
		minYear int
		want    int
	}{
		{
			name:    "empty input",
			papers:  nil,
			seen:    map[string]bool{},
			minYear: 2024,
			want:    0,
		},
		{
			name: "filters empty PaperID",
			papers: []Paper{
				{PaperID: "", Title: "No ID", Year: 2025},
				{PaperID: "a1", Title: "Has ID", Year: 2025},
			},
			seen:    map[string]bool{},
			minYear: 2024,
			want:    1,
		},
		{
			name: "filters already seen",
			papers: []Paper{
				{PaperID: "a1", Title: "Seen", Year: 2025},
				{PaperID: "a2", Title: "New", Year: 2025},
			},
			seen:    map[string]bool{"a1": true},
			minYear: 2024,
			want:    1,
		},
		{
			name: "filters below minYear",
			papers: []Paper{
				{PaperID: "a1", Title: "Old", Year: 2023},
				{PaperID: "a2", Title: "New", Year: 2024},
				{PaperID: "a3", Title: "Newer", Year: 2025},
			},
			seen:    map[string]bool{},
			minYear: 2024,
			want:    2,
		},
		{
			name: "mixed filters",
			papers: []Paper{
				{PaperID: "", Title: "No ID", Year: 2025},
				{PaperID: "seen1", Title: "Already seen", Year: 2025},
				{PaperID: "old1", Title: "Too old", Year: 2020},
				{PaperID: "good1", Title: "Valid", Year: 2025},
			},
			seen:    map[string]bool{"seen1": true},
			minYear: 2024,
			want:    1,
		},
		{
			name: "does not mutate seen map",
			papers: []Paper{
				{PaperID: "new1", Title: "New paper", Year: 2025},
			},
			seen:    map[string]bool{},
			minYear: 2024,
			want:    1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterNewPapers(tc.papers, tc.seen, tc.minYear)
			if len(got) != tc.want {
				t.Errorf("filterNewPapers returned %d papers, want %d", len(got), tc.want)
			}
		})
	}

	// Verify seen map is not mutated
	t.Run("seen map immutability", func(t *testing.T) {
		seen := map[string]bool{}
		papers := []Paper{{PaperID: "x", Year: 2025}}
		filterNewPapers(papers, seen, 2024)
		if seen["x"] {
			t.Error("filterNewPapers mutated the seen map")
		}
	})
}

func TestArxivXMLDecode(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.00001v1</id>
    <title>Test Paper on Consciousness</title>
    <summary>This paper studies awareness.</summary>
    <published>2024-01-15T00:00:00Z</published>
    <author><name>Jane Smith</name></author>
    <link href="http://arxiv.org/abs/2401.00001v1" type="text/html"/>
  </entry>
</feed>`

	var feed arxivFeed
	if err := xml.Unmarshal([]byte(xmlData), &feed); err != nil {
		t.Fatalf("xml decode: %v", err)
	}
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}
	e := feed.Entries[0]
	if e.Title != "Test Paper on Consciousness" {
		t.Errorf("title = %q", e.Title)
	}
	if len(e.Authors) != 1 || e.Authors[0].Name != "Jane Smith" {
		t.Errorf("authors = %v", e.Authors)
	}
	if e.Published != "2024-01-15T00:00:00Z" {
		t.Errorf("published = %q", e.Published)
	}
}
