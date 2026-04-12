// Dream --fix: autonomous maintenance engine for the KB.
//
// Fixes dream cycle findings without human intervention:
//   - Missing Briefs: LLM generates from entity Name + claim context
//   - Overlong Briefs: LLM tightens to 100-200 chars (--tighten)
//
// Source modification uses byte-offset splicing: parse the file with go/ast
// to find exact positions, replace text at those positions, then run
// go/format to normalize. Quality gates (build + vet + lint) validate
// before committing.
package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// briefTarget is an entity whose Brief needs fixing.
type briefTarget struct {
	file     string // absolute path
	entity   string // variable name
	roleType string
	brief    string // current Brief text ("" if missing)
	// AST positions for byte-offset splicing
	briefStart int // byte offset of Brief value start (or insertion point)
	briefEnd   int // byte offset of Brief value end
	isMissing  bool
}

func runDreamFix(dir string, tightenOverlong bool, dryRun bool, jsonOut bool) {
	// Collect targets
	targets := collectBriefTargets(dir, tightenOverlong)

	if len(targets) == 0 {
		fmt.Println("[dream-fix] no Brief fixes needed")
		return
	}

	// Separate by type for reporting
	var missing, overlong []briefTarget
	for _, t := range targets {
		if t.isMissing {
			missing = append(missing, t)
		} else {
			overlong = append(overlong, t)
		}
	}

	fmt.Printf("[dream-fix] found %d targets: %d missing, %d overlong\n",
		len(targets), len(missing), len(overlong))

	if dryRun {
		fmt.Println("\n  dry run — would fix:")
		for _, t := range targets {
			kind := "tighten"
			if t.isMissing {
				kind = "generate"
			}
			fmt.Printf("    %-42s %s (%s, %d chars)\n",
				t.entity, filepath.Base(t.file), kind, len(t.brief))
		}
		return
	}

	// Need LLM for tightening
	if tightenOverlong && len(overlong) > 0 {
		loadDotEnv(dir)
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			fmt.Fprintf(os.Stderr, "[dream-fix] ANTHROPIC_API_KEY required for --tighten\n")
			os.Exit(1)
		}
		client := anthropic.NewClient(option.WithAPIKey(apiKey))

		// Backup original files
		backups := map[string][]byte{}
		for _, t := range overlong {
			if _, ok := backups[t.file]; !ok {
				data, err := os.ReadFile(t.file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[dream-fix] error reading %s: %v\n", t.file, err)
					os.Exit(1)
				}
				backups[t.file] = data
			}
		}

		// Process files
		filesFixed := 0
		entitiesFixed := 0
		for file, original := range backups {
			// Collect all targets for this file
			var fileTargets []briefTarget
			for _, t := range overlong {
				if t.file == file {
					fileTargets = append(fileTargets, t)
				}
			}

			modified, n, err := fixBriefsInFile(file, original, fileTargets, &client)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[dream-fix] error fixing %s: %v\n", filepath.Base(file), err)
				// Restore backup
				_ = os.WriteFile(file, original, 0644)
				continue
			}

			if n > 0 {
				if err := os.WriteFile(file, modified, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "[dream-fix] error writing %s: %v\n", filepath.Base(file), err)
					_ = os.WriteFile(file, original, 0644)
					continue
				}
				filesFixed++
				entitiesFixed += n
				fmt.Printf("  fixed %d entities in %s\n", n, filepath.Base(file))
			}
		}

		fmt.Printf("\n[dream-fix] tightened %d entities across %d files\n", entitiesFixed, filesFixed)

		// Quality gates
		if entitiesFixed > 0 {
			fmt.Println("\n[dream-fix] running quality gates...")
			if !runDreamGate(dir, "go", "build", "./...") {
				fmt.Println("[dream-fix] FAIL: go build — reverting all changes")
				for file, original := range backups {
					_ = os.WriteFile(file, original, 0644)
				}
				os.Exit(2)
			}
			if !runDreamGate(dir, "go", "vet", "./...") {
				fmt.Println("[dream-fix] FAIL: go vet — reverting all changes")
				for file, original := range backups {
					_ = os.WriteFile(file, original, 0644)
				}
				os.Exit(2)
			}
			if !runDreamGate(dir, "go", "run", "./cmd/lint", dir) {
				fmt.Println("[dream-fix] FAIL: lint — reverting all changes")
				for file, original := range backups {
					_ = os.WriteFile(file, original, 0644)
				}
				os.Exit(2)
			}
			fmt.Println("[dream-fix] all quality gates passed")
		}
	}
}

// fixBriefsInFile processes all Brief fixes for a single file.
// Returns the modified file content and the number of fixes applied.
func fixBriefsInFile(filePath string, original []byte, targets []briefTarget, client *anthropic.Client) ([]byte, int, error) {
	// Sort targets by byte offset descending — process from end of file
	// so earlier fixes don't shift later offsets.
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].briefStart > targets[j].briefStart
	})

	content := make([]byte, len(original))
	copy(content, original)
	fixed := 0

	for _, t := range targets {
		// Generate tightened Brief via LLM
		newBrief, err := tightenBrief(*client, t.entity, t.roleType, t.brief)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    skip %s: LLM error: %v\n", t.entity, err)
			continue
		}

		// Escape for Go source
		escaped := escapeGoString(newBrief)
		replacement := fmt.Sprintf(`"%s"`, escaped)

		// Splice: replace bytes from briefStart to briefEnd
		var buf []byte
		buf = append(buf, content[:t.briefStart]...)
		buf = append(buf, []byte(replacement)...)
		buf = append(buf, content[t.briefEnd:]...)
		content = buf
		fixed++

		fmt.Printf("    %s: %d → %d chars\n", t.entity, len(t.brief), len(newBrief))
	}

	if fixed == 0 {
		return original, 0, nil
	}

	// Format the result
	formatted, err := format.Source(content)
	if err != nil {
		return nil, 0, fmt.Errorf("go/format: %w", err)
	}

	return formatted, fixed, nil
}

// collectBriefTargets finds all entities needing Brief fixes.
func collectBriefTargets(dir string, includeOverlong bool) []briefTarget {
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// Collect role types
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), ".go") && !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil
	}
	roleTypes := collectDreamRoleTypes(pkgs)

	var targets []briefTarget

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		if isInfraFile(e.Name()) {
			continue
		}

		filePath := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Values) == 0 {
					continue
				}
				cl, ok := vs.Values[0].(*ast.CompositeLit)
				if !ok {
					continue
				}
				typeName := compositeTypeName(cl)
				if !roleTypes[typeName] {
					continue
				}

				varName := vs.Names[0].Name

				// Find the Brief field and its position
				briefExpr, briefText := findBriefExpr(cl)
				if briefExpr == nil {
					// Truly missing Brief — would need insertion
					// Skip for now (none exist after parser fix)
					continue
				}

				if briefText == "" {
					targets = append(targets, briefTarget{
						file:       filePath,
						entity:     varName,
						roleType:   typeName,
						brief:      "",
						briefStart: fset.Position(briefExpr.Pos()).Offset,
						briefEnd:   fset.Position(briefExpr.End()).Offset,
						isMissing:  true,
					})
				} else if includeOverlong && len(briefText) > 300 {
					targets = append(targets, briefTarget{
						file:       filePath,
						entity:     varName,
						roleType:   typeName,
						brief:      briefText,
						briefStart: fset.Position(briefExpr.Pos()).Offset,
						briefEnd:   fset.Position(briefExpr.End()).Offset,
						isMissing:  false,
					})
				}
			}
		}
	}

	return targets
}

// findBriefExpr locates the Brief field's value expression in an entity literal.
// Returns the expression node and the resolved string value.
func findBriefExpr(cl *ast.CompositeLit) (ast.Expr, string) {
	// Check direct KV fields
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if ok {
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Brief" {
				return kv.Value, resolveStringExpr(kv.Value)
			}
			continue
		}
		// RoleType{&Entity{...}} pattern
		ue, ok := elt.(*ast.UnaryExpr)
		if !ok {
			continue
		}
		inner, ok := ue.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, innerElt := range inner.Elts {
			kv, ok := innerElt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Brief" {
				return kv.Value, resolveStringExpr(kv.Value)
			}
		}
	}
	return nil, ""
}

// tightenBrief calls the LLM to generate a shorter Brief.
func tightenBrief(client anthropic.Client, entityName, roleType, currentBrief string) (string, error) {
	prompt := fmt.Sprintf(`Tighten this knowledge base entity Brief to 100-200 characters. One or two sentences maximum.

Rules:
1. Preserve the core factual identity — what this entity IS
2. Remove narrative context, parenthetical asides, cross-references to other entities
3. Do not add information not present in the original
4. Do not mention file names, schema details, or KB metadata
5. Start with the entity's role (e.g., "Austrian logician who..." not "A person named...")

Entity: %s (%s)
Current Brief (%d chars):
%s

Respond with ONLY the tightened Brief text. No quotes, no explanation, no prefix.`, entityName, roleType, len(currentBrief), currentBrief)

	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("API error: %w", err)
	}

	for _, block := range resp.Content {
		if block.Type == "text" {
			result := strings.TrimSpace(block.Text)
			// Strip quotes if LLM wrapped them
			result = strings.Trim(result, "\"'`")
			// Clean LLM artifacts (smart quotes, escaped newlines) before
			// the result is spliced into Go source code.
			result = cleanLLMString(result)
			return result, nil
		}
	}
	return "", fmt.Errorf("no text in response")
}

// escapeGoString escapes a string for inclusion in Go source as a quoted literal.
func escapeGoString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// runDreamGate runs a quality gate command. Returns true if it passes.
func runDreamGate(dir string, name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}
