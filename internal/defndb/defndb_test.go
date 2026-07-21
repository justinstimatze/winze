package defndb_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/justinstimatze/winze/internal/astutil"
	"github.com/justinstimatze/winze/internal/defndb"
)

func rootDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// newClient opens a Client over the corpus root. The backing store is the
// corpus source itself, so unlike the previous defn-backed implementation
// there is no external database that can be missing — these tests always run
// rather than skipping into a false green.
func newClient(t *testing.T) *defndb.Client {
	t.Helper()
	client, err := defndb.New(rootDir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestConcordance_RoleTypes(t *testing.T) {
	client := newClient(t)
	dir := rootDir(t)

	// AST path
	pkgs, _, err := astutil.ParseCorpus(dir)
	if err != nil {
		t.Fatal(err)
	}
	astRoles := astutil.CollectRoleTypes(pkgs)

	// defn path
	defnRoles, err := client.RoleTypeSet()
	if err != nil {
		t.Fatal(err)
	}

	for name := range astRoles {
		if !defnRoles[name] {
			t.Errorf("AST found role type %q but defn did not", name)
		}
	}
	for name := range defnRoles {
		if !astRoles[name] {
			t.Errorf("defn found role type %q but AST did not", name)
		}
	}
}

func TestConcordance_ClaimFields(t *testing.T) {
	client := newClient(t)

	fields, err := client.ClaimFields()
	if err != nil {
		t.Fatal(err)
	}

	subjects := 0
	objects := 0
	for _, f := range fields {
		switch f.FieldName {
		case "Subject":
			subjects++
		case "Object":
			objects++
		}
	}

	if subjects == 0 {
		t.Error("expected Subject fields, got 0")
	}
	if objects == 0 {
		t.Error("expected Object fields, got 0")
	}
	// Subject count >= Object count (unary claims have Subject but no Object)
	if subjects < objects {
		t.Errorf("expected subjects(%d) >= objects(%d)", subjects, objects)
	}
}

func TestConcordance_Pragmas(t *testing.T) {
	client := newClient(t)

	pragmas, err := client.Pragmas("winze:")
	if err != nil {
		t.Fatal(err)
	}

	contested := 0
	functional := 0
	for _, p := range pragmas {
		switch p.Key {
		case "winze:contested":
			contested++
		case "winze:functional":
			functional++
		}
	}

	if contested == 0 {
		t.Error("expected at least one winze:contested pragma")
	}
	if functional == 0 {
		t.Error("expected at least one winze:functional pragma")
	}
}

func TestConcordance_EntityFields(t *testing.T) {
	client := newClient(t)

	fields, err := client.EntityFields()
	if err != nil {
		t.Fatal(err)
	}

	names := 0
	briefs := 0
	for _, f := range fields {
		switch f.FieldName {
		case "Name":
			names++
		case "Brief":
			briefs++
		}
	}

	if names == 0 {
		t.Error("expected Name fields, got 0")
	}
	if briefs == 0 {
		t.Error("expected Brief fields, got 0")
	}
}
