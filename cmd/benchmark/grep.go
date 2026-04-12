package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true,
	"at": true, "be": true, "but": true, "by": true, "did": true,
	"do": true, "does": true, "for": true, "from": true, "had": true,
	"has": true, "have": true, "he": true, "her": true, "his": true,
	"i": true, "if": true, "in": true, "is": true, "it": true,
	"its": true, "me": true, "my": true, "not": true, "of": true,
	"on": true, "or": true, "our": true, "she": true, "so": true,
	"that": true, "the": true, "their": true, "them": true, "they": true,
	"this": true, "to": true, "was": true, "we": true, "were": true,
	"what": true, "when": true, "where": true, "which": true, "who": true,
	"will": true, "with": true, "you": true, "your": true, "about": true,
	"how": true, "many": true, "list": true, "all": true, "exist": true,
}

var tokenRe = regexp.MustCompile(`[A-Za-z0-9']+`)

func extractKeywords(query string) []string {
	raw := tokenRe.FindAllString(query, -1)
	var out []string
	for _, t := range raw {
		t = strings.ToLower(t)
		if len(t) > 2 && !stopwords[t] {
			out = append(out, t)
		}
	}
	return out
}

type varBlock struct {
	name string
	text string
}

func collectVarBlocks(dir string) ([]varBlock, error) {
	fset := token.NewFileSet()
	var out []varBlock

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, nameIdent := range vs.Names {
					start := fset.Position(vs.Pos()).Offset
					end := fset.Position(vs.End()).Offset
					if end > len(src) {
						end = len(src)
					}
					blockText := string(src[start:end])
					out = append(out, varBlock{
						name: nameIdent.Name,
						text: blockText,
					})
				}
			}
		}
	}
	return out, nil
}

type scoredBlock struct {
	name          string
	uniqueKeyHits int
	totalHits     int
}

func grepRetrieve(dir string, query string, k int) []string {
	keywords := extractKeywords(query)
	if len(keywords) == 0 {
		return nil
	}

	blocks, err := collectVarBlocks(dir)
	if err != nil {
		return nil
	}

	var scored []scoredBlock
	for _, b := range blocks {
		lower := strings.ToLower(b.text)
		var uniqueHits int
		var totalHits int
		for _, kw := range keywords {
			count := strings.Count(lower, kw)
			if count > 0 {
				uniqueHits++
				totalHits += count
			}
		}
		if uniqueHits > 0 {
			scored = append(scored, scoredBlock{
				name:          b.name,
				uniqueKeyHits: uniqueHits,
				totalHits:     totalHits,
			})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].uniqueKeyHits != scored[j].uniqueKeyHits {
			return scored[i].uniqueKeyHits > scored[j].uniqueKeyHits
		}
		return scored[i].totalHits > scored[j].totalHits
	})

	var out []string
	for i, s := range scored {
		if i >= k {
			break
		}
		out = append(out, s.name)
	}
	return out
}
