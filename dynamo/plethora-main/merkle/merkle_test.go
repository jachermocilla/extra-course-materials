package merkle

import (
	"crypto/md5"
	"testing"
)

func h(s string) [16]byte {
	return md5.Sum([]byte(s))
}

func TestBuildNil(t *testing.T) {
	if Build(nil) != nil {
		t.Fatal("expected nil tree for empty input")
	}
}

func TestBuildSingleEntry(t *testing.T) {
	tree := Build([]KeyHash{{Key: "a", Hash: h("a")}})
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if tree.Key != "a" {
		t.Fatalf("expected leaf key 'a', got %q", tree.Key)
	}
	if tree.Hash != h("a") {
		t.Fatal("leaf hash mismatch")
	}
}

func TestBuildDeterministic(t *testing.T) {
	entries := []KeyHash{
		{Key: "c", Hash: h("c")},
		{Key: "a", Hash: h("a")},
		{Key: "b", Hash: h("b")},
	}
	tree1 := Build(entries)

	// reverse order, should produce the same root hash after sorting
	entries2 := []KeyHash{
		{Key: "b", Hash: h("b")},
		{Key: "a", Hash: h("a")},
		{Key: "c", Hash: h("c")},
	}
	tree2 := Build(entries2)

	if tree1.Hash != tree2.Hash {
		t.Fatal("trees built from same data in different order should have the same root hash")
	}
}

func TestBuildPowerOfTwo(t *testing.T) {
	entries := []KeyHash{
		{Key: "a", Hash: h("a")},
		{Key: "b", Hash: h("b")},
		{Key: "c", Hash: h("c")},
		{Key: "d", Hash: h("d")},
	}
	tree := Build(entries)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if tree.Left == nil || tree.Right == nil {
		t.Fatal("root should have two children for 4 entries")
	}
}

func TestDiffIdenticalTrees(t *testing.T) {
	entries := []KeyHash{
		{Key: "a", Hash: h("a")},
		{Key: "b", Hash: h("b")},
		{Key: "c", Hash: h("c")},
	}
	tree1 := Build(entries)
	tree2 := Build(entries)

	diff := Diff(tree1, tree2)
	if len(diff) != 0 {
		t.Fatalf("expected no diff for identical trees, got %v", diff)
	}
}

func TestDiffOneKeyDiffers(t *testing.T) {
	local := []KeyHash{
		{Key: "a", Hash: h("a-v1")},
		{Key: "b", Hash: h("b-v1")},
		{Key: "c", Hash: h("c-v1")},
	}
	remote := []KeyHash{
		{Key: "a", Hash: h("a-v1")},
		{Key: "b", Hash: h("b-v2")}, // different
		{Key: "c", Hash: h("c-v1")},
	}

	diff := Diff(Build(local), Build(remote))
	if len(diff) != 1 || diff[0] != "b" {
		t.Fatalf("expected diff [b], got %v", diff)
	}
}

func TestDiffMultipleKeysDiffer(t *testing.T) {
	local := []KeyHash{
		{Key: "a", Hash: h("a-v1")},
		{Key: "b", Hash: h("b-v1")},
		{Key: "c", Hash: h("c-v1")},
		{Key: "d", Hash: h("d-v1")},
	}
	remote := []KeyHash{
		{Key: "a", Hash: h("a-v2")},
		{Key: "b", Hash: h("b-v1")},
		{Key: "c", Hash: h("c-v1")},
		{Key: "d", Hash: h("d-v2")},
	}

	diff := Diff(Build(local), Build(remote))
	if len(diff) != 2 {
		t.Fatalf("expected 2 diffs, got %v", diff)
	}
	diffSet := map[string]bool{}
	for _, k := range diff {
		diffSet[k] = true
	}
	if !diffSet["a"] || !diffSet["d"] {
		t.Fatalf("expected diffs for 'a' and 'd', got %v", diff)
	}
}

func TestDiffBothNil(t *testing.T) {
	diff := Diff(nil, nil)
	if len(diff) != 0 {
		t.Fatalf("expected no diff for two nil trees, got %v", diff)
	}
}

func TestDiffOneNil(t *testing.T) {
	entries := []KeyHash{
		{Key: "a", Hash: h("a")},
		{Key: "b", Hash: h("b")},
	}
	tree := Build(entries)

	diff := Diff(tree, nil)
	if len(diff) != 2 {
		t.Fatalf("expected 2 keys from non-nil side, got %v", diff)
	}

	diff = Diff(nil, tree)
	if len(diff) != 2 {
		t.Fatalf("expected 2 keys from non-nil side, got %v", diff)
	}
}

func TestDiffAllKeysDiffer(t *testing.T) {
	local := []KeyHash{
		{Key: "x", Hash: h("x-old")},
		{Key: "y", Hash: h("y-old")},
	}
	remote := []KeyHash{
		{Key: "x", Hash: h("x-new")},
		{Key: "y", Hash: h("y-new")},
	}

	diff := Diff(Build(local), Build(remote))
	if len(diff) != 2 {
		t.Fatalf("expected 2 diffs, got %v", diff)
	}
}
