package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/resetfixture\n\ngo 1.26\n")
	writeTestFile(t, root, "sample/sample.go", `package sample

import "example.com/resetfixture/other"

// generate:reset
type Child struct { Count int }

type Plain struct { Count int; Names []string; Values map[string]int }

// generate:reset
type Record struct {
	Count int
	Name string
	Ready bool
	Names []string
	Values map[string]int
	CountP *int
	NamesP *[]string
	ValuesP *map[string]int
	Child Child
	ChildP *Child
	EmptyP *Child
	PlainP *Plain
	Other other.Value
}

type Untouched struct { Count int }

// generate:reset
type Box[T any] struct { Value T; Items []T }
`)
	writeTestFile(t, root, "sample/sample_test.go", `package sample

import (
	"testing"
	"example.com/resetfixture/other"
)

func TestReset(t *testing.T) {
	n := 4
	names := []string{"a", "b"}
	values := map[string]int{"a": 1}
	r := &Record{Count: 3, Name: "x", Ready: true, Names: []string{"x"}, Values: map[string]int{"x": 1}, CountP: &n, NamesP: &names, ValuesP: &values, Child: Child{Count: 1}, ChildP: &Child{Count: 2}, PlainP: &Plain{Count: 3, Names: []string{"x"}, Values: map[string]int{"x": 1}}, Other: other.Value{Flag: true}}
	namesData, mapData := r.Names, r.Values
	plainNames := r.PlainP.Names
	r.Reset()
	if r.Count != 0 || r.Name != "" || r.Ready || n != 0 || r.Child.Count != 0 || r.ChildP.Count != 0 || r.EmptyP != nil || r.Other.Flag { t.Fatal("scalar or nested value was not reset") }
	if len(r.Names) != 0 || r.Names == nil || &r.Names[:cap(r.Names)][0] != &namesData[0] { t.Fatal("slice backing storage was lost") }
	if len(names) != 0 || names == nil || len(r.Values) != 0 || r.Values == nil || len(values) != 0 { t.Fatal("collection was not retained and cleared") }
	if r.PlainP.Count != 0 || len(r.PlainP.Names) != 0 || r.PlainP.Names == nil || &r.PlainP.Names[:cap(r.PlainP.Names)][0] != &plainNames[0] || len(r.PlainP.Values) != 0 || r.PlainP.Values == nil { t.Fatal("plain nested struct was not reset recursively") }
	mapData["new"] = 2
	if r.Values["new"] != 2 { t.Fatal("map identity was lost") }
	var nilRecord *Record
	nilRecord.Reset()
	box := &Box[int]{Value: 3, Items: []int{1}}
	box.Reset()
	if box.Value != 0 || box.Items == nil || len(box.Items) != 0 { t.Fatal("generic struct was not reset") }
}
`)
	writeTestFile(t, root, "other/other.go", "package other\n\n// generate:reset\ntype Value struct { Flag bool }\n")
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sample/reset.gen.go", "other/reset.gen.go"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.ReadFile(filepath.Join(root, "sample/reset.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(root, "sample/reset.gen.go"))
	if err != nil || string(before) != string(after) {
		t.Fatalf("generation is not idempotent: %v", err)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated code failed: %v\n%s\n%s", err, output, before)
	}
	if strings.Contains(string(before), "Untouched) Reset") {
		t.Fatal("generated a method for an unmarked struct")
	}
}

func TestRegenerateAfterFieldChange(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/resetfixture\n\ngo 1.26\n")
	writeTestFile(t, root, "sample/sample.go", "package sample\n\n// generate:reset\ntype Value struct { Old int }\n")
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "sample/sample.go", "package sample\n\n// generate:reset\ntype Value struct { New int }\n")
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join(root, "sample/reset.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "value.New") || strings.Contains(string(generated), "value.Old") {
		t.Fatalf("outdated generated code: %s", generated)
	}
}

func TestRegenerateWhenSourceCallsReset(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/resetfixture\n\ngo 1.26\n")
	writeTestFile(t, root, "sample.go", "package sample\n\n// generate:reset\ntype Value struct { Count int }\n")
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "sample.go", "package sample\n\n// generate:reset\ntype Value struct { Count int }\n\nfunc Use(v *Value) { v.Reset() }\n")
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	testGeneratedPackage(t, root)
}

func TestGenerateWhenSourceCallsReset(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/resetfixture\n\ngo 1.26\n")
	writeTestFile(t, root, "sample.go", "package sample\n\n// generate:reset\ntype Value struct { Count int }\n\nfunc Use(v *Value) { v.Reset() }\n")
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	testGeneratedPackage(t, root)
}

func TestRemoveGeneratedFileAfterTypeDeletion(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/resetfixture\n\ngo 1.26\n")
	writeTestFile(t, root, "sample.go", "package sample\n\n// generate:reset\ntype Value struct { Count int }\n")
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "sample.go", "package sample\n\ntype Other struct { Count int }\n")
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "reset.gen.go")); !os.IsNotExist(err) {
		t.Fatalf("stale generated file remains: %v", err)
	}
	testGeneratedPackage(t, root)
}

func testGeneratedPackage(t *testing.T, root string) {
	t.Helper()
	command := exec.Command("go", "test", "./...")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated package failed: %v\n%s", err, output)
	}
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
