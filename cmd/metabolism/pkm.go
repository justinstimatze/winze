package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// pkmEntity is a generated entity, ready to render into a corpus file.
type pkmEntity struct {
	varName    string // Go variable name (PascalCase)
	roleType   string // Person, Concept, Hypothesis
	id         string // kebab-case ID
	name       string // display name
	kind       string // person, concept, hypothesis
	brief      string // one-line description
	sourceNote string // which note introduced this entity
}

// pkmClaim is a generated claim for code output.
type pkmClaim struct {
	varName    string // Go variable name
	predicate  string // Authored, BelongsTo, Proposes, etc.
	subject    string // subject entity varName
	object     string // object entity varName
	provVar    string // provenance variable name
	sourceNote string // which note generated this claim
}

// collectExistingEntityNames scans the KB for existing entity variable names and IDs.
func collectExistingEntityNames(dir string) map[string]string {
	existing := map[string]string{} // varName -> id
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return existing
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		// Skip pkm_ files (our own output)
		if strings.HasPrefix(e.Name(), "pkm_") {
			continue
		}

		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
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
				for i, nameIdent := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					// Check if this is a role-typed entity (Person{...}, Concept{...}, etc.)
					cl, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					typeName := ""
					switch t := cl.Type.(type) {
					case *ast.Ident:
						typeName = t.Name
					}
					roleTypes := map[string]bool{
						"Person": true, "Organization": true, "Place": true,
						"Event": true, "Facility": true, "Substance": true,
						"Instrument": true, "Hypothesis": true, "Concept": true,
					}
					if !roleTypes[typeName] {
						continue
					}
					// Extract ID
					id := extractEntityID(cl)
					existing[nameIdent.Name] = id
				}
			}
		}
	}
	return existing
}

// extractEntityID pulls the ID field from a role-typed entity literal.
func extractEntityID(cl *ast.CompositeLit) string {
	// Role types wrap &Entity{...} — the first element is a unary expression
	for _, elt := range cl.Elts {
		ue, ok := elt.(*ast.UnaryExpr)
		if !ok {
			continue
		}
		inner, ok := ue.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, field := range inner.Elts {
			kv, ok := field.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "ID" {
				return resolveStringExpr(kv.Value)
			}
		}
	}
	return ""
}

// extractPKMContent generates entities and claims from parsed notes.

type pkmProvenance struct {
	origin string
	quote  string
}

// generatePKMFile writes a .go file for one vault subdirectory.

func slugify(s string) string {
	var result []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, r)
		} else if len(result) > 0 && result[len(result)-1] != '-' {
			result = append(result, '-')
		}
	}
	return strings.Trim(string(result), "-")
}

func noteDir(sourceNote string) string {
	d := filepath.Dir(sourceNote)
	if d == "." {
		return "root"
	}
	return d
}

func sanitizeDir(dir string) string {
	return strings.ReplaceAll(strings.ReplaceAll(dir, "/", "_"), ".", "root")
}

// goIdentifier converts a string to a valid Go identifier (PascalCase, alphanumeric only).
func goIdentifier(s string) string {
	// Split on any non-alphanumeric character
	var words []string
	current := ""
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current += string(r)
		} else {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		}
	}
	if current != "" {
		words = append(words, current)
	}

	var parts []string
	for _, w := range words {
		if len(w) > 0 {
			// Capitalize first letter, lowercase rest
			first := strings.ToUpper(w[:1])
			rest := ""
			if len(w) > 1 {
				rest = strings.ToLower(w[1:])
			}
			parts = append(parts, first+rest)
		}
	}

	result := strings.Join(parts, "")
	// Ensure it doesn't start with a digit
	if len(result) > 0 && unicode.IsDigit(rune(result[0])) {
		result = "N" + result
	}
	return result
}

func cleanGoString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
