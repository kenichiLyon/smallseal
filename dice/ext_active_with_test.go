package dice

import (
	"testing"

	"github.com/sealdice/smallseal/dice/types"
)

// ========== BuildActiveWithGraph Tests ==========

func TestBuildActiveWithGraph_Empty(t *testing.T) {
	graph := BuildActiveWithGraph(nil)
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
}

func TestBuildActiveWithGraph_NoActiveWith(t *testing.T) {
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b"},
	}

	graph := BuildActiveWithGraph(exts)

	// No followers expected
	followers, _ := graph.Load("ext-a")
	if len(followers) != 0 {
		t.Fatalf("expected no followers, got %v", followers)
	}
}

func TestBuildActiveWithGraph_Simple(t *testing.T) {
	// ext-b follows ext-a
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b", ActiveWith: []string{"ext-a"}},
	}

	graph := BuildActiveWithGraph(exts)

	// ext-a should have ext-b as follower
	followers, _ := graph.Load("ext-a")
	if len(followers) != 1 || followers[0] != "ext-b" {
		t.Fatalf("expected [ext-b], got %v", followers)
	}

	// ext-b should have no followers
	followers, _ = graph.Load("ext-b")
	if len(followers) != 0 {
		t.Fatalf("expected no followers for ext-b, got %v", followers)
	}
}

func TestBuildActiveWithGraph_Multiple(t *testing.T) {
	// ext-b and ext-c both follow ext-a
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b", ActiveWith: []string{"ext-a"}},
		{Name: "ext-c", ActiveWith: []string{"ext-a"}},
	}

	graph := BuildActiveWithGraph(exts)

	followers, _ := graph.Load("ext-a")
	if len(followers) != 2 {
		t.Fatalf("expected 2 followers, got %d", len(followers))
	}
}

func TestBuildActiveWithGraph_Chain(t *testing.T) {
	// ext-c follows ext-b, ext-b follows ext-a
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b", ActiveWith: []string{"ext-a"}},
		{Name: "ext-c", ActiveWith: []string{"ext-b"}},
	}

	graph := BuildActiveWithGraph(exts)

	// ext-a -> [ext-b]
	followers, _ := graph.Load("ext-a")
	if len(followers) != 1 || followers[0] != "ext-b" {
		t.Fatalf("expected [ext-b], got %v", followers)
	}

	// ext-b -> [ext-c]
	followers, _ = graph.Load("ext-b")
	if len(followers) != 1 || followers[0] != "ext-c" {
		t.Fatalf("expected [ext-c], got %v", followers)
	}
}

func TestBuildActiveWithGraph_SkipsDeleted(t *testing.T) {
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b", ActiveWith: []string{"ext-a"}, IsDeleted: true},
	}

	graph := BuildActiveWithGraph(exts)

	// ext-a should have no followers (ext-b is deleted)
	followers, _ := graph.Load("ext-a")
	if len(followers) != 0 {
		t.Fatalf("expected no followers (deleted), got %v", followers)
	}
}

// ========== CollectChainedNames Tests ==========

func TestCollectChainedNames_Empty(t *testing.T) {
	names := CollectChainedNames(nil, "ext-a", 10)
	if len(names) != 0 {
		t.Fatalf("expected empty, got %v", names)
	}
}

func TestCollectChainedNames_Simple(t *testing.T) {
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b", ActiveWith: []string{"ext-a"}},
	}
	graph := BuildActiveWithGraph(exts)

	names := CollectChainedNames(graph, "ext-a", 10)
	if len(names) != 1 || names[0] != "ext-b" {
		t.Fatalf("expected [ext-b], got %v", names)
	}
}

func TestCollectChainedNames_Chain(t *testing.T) {
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b", ActiveWith: []string{"ext-a"}},
		{Name: "ext-c", ActiveWith: []string{"ext-b"}},
	}
	graph := BuildActiveWithGraph(exts)

	names := CollectChainedNames(graph, "ext-a", 10)
	// Should include both ext-b and ext-c (in topological order)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %v", names)
	}
}

func TestCollectChainedNames_DepthLimit(t *testing.T) {
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b", ActiveWith: []string{"ext-a"}},
		{Name: "ext-c", ActiveWith: []string{"ext-b"}},
		{Name: "ext-d", ActiveWith: []string{"ext-c"}},
	}
	graph := BuildActiveWithGraph(exts)

	// With depth limit 1, should only get immediate followers
	names := CollectChainedNames(graph, "ext-a", 1)
	// ext-b at depth 1 is included, ext-c at depth 2 should be stopped
	if len(names) > 2 {
		t.Fatalf("expected max 2 due to depth limit, got %v", names)
	}
}

func TestCollectChainedNames_Circular(t *testing.T) {
	// Create circular dependency: a -> b -> c -> a
	exts := []*types.ExtInfo{
		{Name: "ext-a", ActiveWith: []string{"ext-c"}},
		{Name: "ext-b", ActiveWith: []string{"ext-a"}},
		{Name: "ext-c", ActiveWith: []string{"ext-b"}},
	}
	graph := BuildActiveWithGraph(exts)

	// Should not hang due to circular detection
	names := CollectChainedNames(graph, "ext-a", 10)
	// Should have collected some names without infinite loop
	if len(names) > 3 {
		t.Fatalf("circular detection failed, got %v", names)
	}
}

func TestCollectChainedNames_NoFollowers(t *testing.T) {
	exts := []*types.ExtInfo{
		{Name: "ext-a"},
		{Name: "ext-b"},
	}
	graph := BuildActiveWithGraph(exts)

	names := CollectChainedNames(graph, "ext-a", 10)
	if len(names) != 0 {
		t.Fatalf("expected no followers, got %v", names)
	}
}

// ========== Integration Tests ==========

func TestDice_GetFollowerExtensions(t *testing.T) {
	d := NewDice()

	// Register extensions with ActiveWith
	extA := &types.ExtInfo{Name: "test-a"}
	extB := &types.ExtInfo{Name: "test-b", ActiveWith: []string{"test-a"}}

	d.ExtList = append(d.ExtList, extA, extB)
	_ = d.ExtRegistry.Register(extA)
	_ = d.ExtRegistry.Register(extB)

	followers := d.GetFollowerExtensions("test-a")
	if len(followers) != 1 {
		t.Fatalf("expected 1 follower, got %d", len(followers))
	}
	if followers[0].Name != "test-b" {
		t.Fatalf("expected test-b, got %s", followers[0].Name)
	}
}

func TestDice_InvalidateActiveWithGraph(t *testing.T) {
	d := NewDice()

	// Build initial graph
	_ = d.getActiveWithGraph()

	// Should not be nil
	if d.activeWithGraph == nil {
		t.Fatal("expected graph to be built")
	}

	// Invalidate
	d.invalidateActiveWithGraph()

	// Should be nil after invalidation
	if d.activeWithGraph != nil {
		t.Fatal("expected graph to be nil after invalidation")
	}
}
