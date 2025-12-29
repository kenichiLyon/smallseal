package dice

import (
	"strings"
	"testing"

	"github.com/sealdice/smallseal/dice/types"
)

// ========== Basic Registration Tests ==========

func TestExtRegistry_Register(t *testing.T) {
	reg := NewExtRegistry()

	ext := &types.ExtInfo{
		Name:    "test-ext",
		Version: "1.0.0",
		Author:  "tester",
	}

	err := reg.Register(ext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify registration
	found := reg.GetRealExt("test-ext")
	if found == nil {
		t.Fatal("registered extension not found")
	}
	if found.Name != "test-ext" {
		t.Fatalf("expected name 'test-ext', got %q", found.Name)
	}
}

func TestExtRegistry_Register_EmptyName(t *testing.T) {
	reg := NewExtRegistry()

	ext := &types.ExtInfo{
		Name: "",
	}

	err := reg.Register(ext)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestExtRegistry_Register_NilExtension(t *testing.T) {
	reg := NewExtRegistry()

	err := reg.Register(nil)
	if err == nil {
		t.Fatal("expected error for nil extension")
	}
}

func TestExtRegistry_Register_Duplicate(t *testing.T) {
	reg := NewExtRegistry()

	ext1 := &types.ExtInfo{Name: "dup-ext", Version: "1.0"}
	ext2 := &types.ExtInfo{Name: "dup-ext", Version: "2.0"}

	if err := reg.Register(ext1); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	err := reg.Register(ext2)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

// ========== GetRealExt Tests ==========

func TestExtRegistry_GetRealExt_NotFound(t *testing.T) {
	reg := NewExtRegistry()

	ext := reg.GetRealExt("nonexistent")
	if ext != nil {
		t.Fatal("expected nil for non-existent extension")
	}
}

func TestExtRegistry_GetRealExt_Deleted(t *testing.T) {
	reg := NewExtRegistry()

	ext := &types.ExtInfo{Name: "deleted-ext"}
	_ = reg.Register(ext)

	// Mark as deleted
	ext.IsDeleted = true

	found := reg.GetRealExt("deleted-ext")
	if found != nil {
		t.Fatal("expected nil for deleted extension")
	}
}

func TestExtRegistry_GetRealExt_ByAlias(t *testing.T) {
	reg := NewExtRegistry()

	ext := &types.ExtInfo{
		Name:    "main-ext",
		Aliases: []string{"alias1", "alias2"},
	}
	_ = reg.Register(ext)

	// Find by alias
	found := reg.GetRealExt("alias1")
	if found == nil {
		t.Fatal("expected to find extension by alias")
	}
	if found.Name != "main-ext" {
		t.Fatalf("expected name 'main-ext', got %q", found.Name)
	}

	// Find by second alias
	found2 := reg.GetRealExt("alias2")
	if found2 == nil {
		t.Fatal("expected to find extension by second alias")
	}
}

func TestExtRegistry_GetRealExt_CaseInsensitive(t *testing.T) {
	reg := NewExtRegistry()

	ext := &types.ExtInfo{Name: "CamelCase"}
	_ = reg.Register(ext)

	found := reg.GetRealExt("camelcase")
	if found == nil {
		t.Fatal("expected case-insensitive match")
	}
	if found.Name != "CamelCase" {
		t.Fatalf("expected original name 'CamelCase', got %q", found.Name)
	}
}

func TestExtRegistry_GetRealExt_EmptyName(t *testing.T) {
	reg := NewExtRegistry()

	ext := reg.GetRealExt("")
	if ext != nil {
		t.Fatal("expected nil for empty name")
	}
}

// ========== Wrapper Tests ==========

func TestExtRegistry_CreateWrapper(t *testing.T) {
	reg := NewExtRegistry()

	ext := &types.ExtInfo{
		Name:     "real-ext",
		Version:  "1.0",
		Author:   "author",
		Category: types.ExtCategorySystem,
		CmdMap: types.CmdMapCls{
			"cmd1": &types.CmdItemInfo{Name: "cmd1"},
		},
	}
	_ = reg.Register(ext)

	wrapper := reg.CreateWrapper("real-ext")
	if wrapper == nil {
		t.Fatal("failed to create wrapper")
	}
	if !wrapper.IsWrapper {
		t.Fatal("expected IsWrapper to be true")
	}
	if wrapper.WrappedName != "real-ext" {
		t.Fatalf("expected WrappedName 'real-ext', got %q", wrapper.WrappedName)
	}
	// Verify CmdMap is shared (same reference)
	if wrapper.CmdMap["cmd1"] != ext.CmdMap["cmd1"] {
		t.Fatal("expected CmdMap to be shared")
	}
}

func TestExtRegistry_CreateWrapper_NonExistent(t *testing.T) {
	reg := NewExtRegistry()

	wrapper := reg.CreateWrapper("nonexistent")
	if wrapper != nil {
		t.Fatal("expected nil for non-existent extension")
	}
}

func TestExtRegistry_UpdateWrapper(t *testing.T) {
	reg := NewExtRegistry()

	ext := &types.ExtInfo{Name: "ext1", Version: "1.0"}
	_ = reg.Register(ext)
	reg.CreateWrapper("ext1")

	newExt := &types.ExtInfo{
		Name:    "ext1",
		Version: "2.0",
		CmdMap: types.CmdMapCls{
			"newcmd": &types.CmdItemInfo{Name: "newcmd"},
		},
	}

	err := reg.UpdateWrapper("ext1", newExt)
	if err != nil {
		t.Fatalf("update wrapper failed: %v", err)
	}

	// Verify wrapper is updated
	reg.mu.RLock()
	wrapper := reg.wrappers["ext1"]
	reg.mu.RUnlock()

	if wrapper.Version != "2.0" {
		t.Fatalf("expected version '2.0', got %q", wrapper.Version)
	}
	if wrapper.CmdMap["newcmd"] == nil {
		t.Fatal("expected CmdMap to be updated")
	}
}

func TestExtRegistry_UpdateWrapper_NotFound(t *testing.T) {
	reg := NewExtRegistry()

	newExt := &types.ExtInfo{Name: "ext1", Version: "2.0"}
	err := reg.UpdateWrapper("nonexistent", newExt)
	if err == nil {
		t.Fatal("expected error for non-existent wrapper")
	}
}

func TestExtRegistry_GetRealExt_ThroughWrapper(t *testing.T) {
	reg := NewExtRegistry()

	ext := &types.ExtInfo{Name: "wrapped-ext", Version: "1.0"}
	_ = reg.Register(ext)
	reg.CreateWrapper("wrapped-ext")

	// GetRealExt should return the real extension, not wrapper
	found := reg.GetRealExt("wrapped-ext")
	if found == nil {
		t.Fatal("expected to find extension")
	}
	if found.IsWrapper {
		t.Fatal("expected real extension, not wrapper")
	}
	if found.Name != "wrapped-ext" {
		t.Fatalf("expected name 'wrapped-ext', got %q", found.Name)
	}
}

// ========== Conflict Detection Tests ==========

func TestExtRegistry_CheckConflicts_Direct(t *testing.T) {
	reg := NewExtRegistry()

	// Register existing extension
	existing := &types.ExtInfo{Name: "coc7"}
	_ = reg.Register(existing)

	// New extension declares conflict with coc7
	newExt := &types.ExtInfo{
		Name:         "dnd5e",
		ConflictWith: []string{"coc7"},
	}

	conflicts := reg.CheckConflicts(newExt)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0] != "coc7" {
		t.Fatalf("expected conflict with 'coc7', got %q", conflicts[0])
	}
}

func TestExtRegistry_CheckConflicts_Reverse(t *testing.T) {
	reg := NewExtRegistry()

	// Existing extension declares conflict with future extension
	existing := &types.ExtInfo{
		Name:         "coc7",
		ConflictWith: []string{"dnd5e"},
	}
	_ = reg.Register(existing)

	// New extension doesn't declare any conflicts
	newExt := &types.ExtInfo{Name: "dnd5e"}

	conflicts := reg.CheckConflicts(newExt)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 reverse conflict, got %d", len(conflicts))
	}
	if conflicts[0] != "coc7" {
		t.Fatalf("expected conflict from 'coc7', got %q", conflicts[0])
	}
}

func TestExtRegistry_CheckConflicts_NoConflicts(t *testing.T) {
	reg := NewExtRegistry()

	ext1 := &types.ExtInfo{Name: "ext1"}
	_ = reg.Register(ext1)

	ext2 := &types.ExtInfo{Name: "ext2"}
	conflicts := reg.CheckConflicts(ext2)

	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
}

func TestExtRegistry_CheckConflicts_DeletedNotConflict(t *testing.T) {
	reg := NewExtRegistry()

	existing := &types.ExtInfo{Name: "deleted-ext"}
	_ = reg.Register(existing)
	existing.IsDeleted = true

	newExt := &types.ExtInfo{
		Name:         "new-ext",
		ConflictWith: []string{"deleted-ext"},
	}

	conflicts := reg.CheckConflicts(newExt)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts with deleted extension, got %v", conflicts)
	}
}

// ========== Dependency Resolution Tests ==========

func TestExtRegistry_ResolveWithDependencies_Simple(t *testing.T) {
	reg := NewExtRegistry()

	// A depends on B
	extB := &types.ExtInfo{Name: "B", Priority: 100}
	extA := &types.ExtInfo{
		Name:      "A",
		Priority:  50,
		DependsOn: []types.ExtDependency{{Name: "B"}},
	}

	_ = reg.Register(extB)
	_ = reg.Register(extA)

	sorted, err := reg.ResolveWithDependencies([]*types.ExtInfo{extA})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// B should come before A (dependencies first)
	if len(sorted) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(sorted))
	}
	if sorted[0].Name != "B" {
		t.Fatalf("expected B first, got %q", sorted[0].Name)
	}
	if sorted[1].Name != "A" {
		t.Fatalf("expected A second, got %q", sorted[1].Name)
	}
}

func TestExtRegistry_ResolveWithDependencies_Chain(t *testing.T) {
	reg := NewExtRegistry()

	// A -> B -> C (A depends on B, B depends on C)
	extC := &types.ExtInfo{Name: "C"}
	extB := &types.ExtInfo{
		Name:      "B",
		DependsOn: []types.ExtDependency{{Name: "C"}},
	}
	extA := &types.ExtInfo{
		Name:      "A",
		DependsOn: []types.ExtDependency{{Name: "B"}},
	}

	_ = reg.Register(extC)
	_ = reg.Register(extB)
	_ = reg.Register(extA)

	sorted, err := reg.ResolveWithDependencies([]*types.ExtInfo{extA})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Order should be C, B, A
	if len(sorted) != 3 {
		t.Fatalf("expected 3 extensions, got %d", len(sorted))
	}
	if sorted[0].Name != "C" || sorted[1].Name != "B" || sorted[2].Name != "A" {
		t.Fatalf("expected order C->B->A, got %s->%s->%s",
			sorted[0].Name, sorted[1].Name, sorted[2].Name)
	}
}

func TestExtRegistry_ResolveWithDependencies_OptionalMissing(t *testing.T) {
	reg := NewExtRegistry()

	extA := &types.ExtInfo{
		Name: "A",
		DependsOn: []types.ExtDependency{
			{Name: "optional-dep", Optional: true},
		},
	}

	_ = reg.Register(extA)

	sorted, err := reg.ResolveWithDependencies([]*types.ExtInfo{extA})
	if err != nil {
		t.Fatalf("unexpected error for optional missing dependency: %v", err)
	}

	if len(sorted) != 1 {
		t.Fatalf("expected 1 extension, got %d", len(sorted))
	}
}

func TestExtRegistry_ResolveWithDependencies_RequiredMissing(t *testing.T) {
	reg := NewExtRegistry()

	extA := &types.ExtInfo{
		Name: "A",
		DependsOn: []types.ExtDependency{
			{Name: "required-dep", Optional: false},
		},
	}

	_ = reg.Register(extA)

	_, err := reg.ResolveWithDependencies([]*types.ExtInfo{extA})
	if err == nil {
		t.Fatal("expected error for missing required dependency")
	}
	if !strings.Contains(err.Error(), "missing dependency") {
		t.Fatalf("expected 'missing dependency' error, got: %v", err)
	}
}

func TestExtRegistry_ResolveWithDependencies_Circular(t *testing.T) {
	reg := NewExtRegistry()

	// A -> B -> A (circular)
	extA := &types.ExtInfo{
		Name:      "A",
		DependsOn: []types.ExtDependency{{Name: "B"}},
	}
	extB := &types.ExtInfo{
		Name:      "B",
		DependsOn: []types.ExtDependency{{Name: "A"}},
	}

	_ = reg.Register(extA)
	_ = reg.Register(extB)

	_, err := reg.ResolveWithDependencies([]*types.ExtInfo{extA, extB})
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Fatalf("expected 'circular dependency' error, got: %v", err)
	}
}

func TestExtRegistry_ResolveWithDependencies_Priority(t *testing.T) {
	reg := NewExtRegistry()

	// Two independent extensions, Core should come before System
	extCore := &types.ExtInfo{
		Name:     "core-ext",
		Category: types.ExtCategoryCore,
		Priority: 100,
	}
	extSystem := &types.ExtInfo{
		Name:     "system-ext",
		Category: types.ExtCategorySystem,
		Priority: 200,
	}

	_ = reg.Register(extCore)
	_ = reg.Register(extSystem)

	sorted, err := reg.ResolveWithDependencies([]*types.ExtInfo{extSystem, extCore})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Core category should come first (higher priority)
	if len(sorted) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(sorted))
	}
	if sorted[0].Name != "core-ext" {
		t.Fatalf("expected core-ext first (higher category priority), got %q", sorted[0].Name)
	}
}

func TestExtRegistry_ResolveWithDependencies_Empty(t *testing.T) {
	reg := NewExtRegistry()

	sorted, err := reg.ResolveWithDependencies([]*types.ExtInfo{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sorted != nil {
		t.Fatalf("expected nil for empty input, got %v", sorted)
	}
}

// ========== GetActiveDependencies Tests ==========

func TestExtRegistry_GetActiveDependencies(t *testing.T) {
	reg := NewExtRegistry()

	extC := &types.ExtInfo{Name: "C"}
	extB := &types.ExtInfo{
		Name:      "B",
		DependsOn: []types.ExtDependency{{Name: "C"}},
	}
	extA := &types.ExtInfo{
		Name:      "A",
		DependsOn: []types.ExtDependency{{Name: "B"}},
	}

	_ = reg.Register(extC)
	_ = reg.Register(extB)
	_ = reg.Register(extA)

	deps := reg.GetActiveDependencies("A")
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	// Should contain B and C
	names := make(map[string]bool)
	for _, d := range deps {
		names[d.Name] = true
	}
	if !names["B"] || !names["C"] {
		t.Fatalf("expected dependencies B and C, got %v", names)
	}
}

func TestExtRegistry_GetActiveDependencies_NonExistent(t *testing.T) {
	reg := NewExtRegistry()

	deps := reg.GetActiveDependencies("nonexistent")
	if deps != nil {
		t.Fatalf("expected nil for non-existent extension, got %v", deps)
	}
}

// ========== Validate Tests ==========

func TestExtRegistry_Validate(t *testing.T) {
	reg := NewExtRegistry()

	tests := []struct {
		name    string
		ext     *types.ExtInfo
		wantErr bool
	}{
		{"valid", &types.ExtInfo{Name: "valid"}, false},
		{"nil", nil, true},
		{"empty name", &types.ExtInfo{Name: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reg.Validate(tt.ext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ========== GetAllExtensions Tests ==========

func TestExtRegistry_GetAllExtensions(t *testing.T) {
	reg := NewExtRegistry()

	ext1 := &types.ExtInfo{Name: "ext1"}
	ext2 := &types.ExtInfo{Name: "ext2"}
	ext3 := &types.ExtInfo{Name: "ext3"}

	_ = reg.Register(ext1)
	_ = reg.Register(ext2)
	_ = reg.Register(ext3)

	// Delete one
	ext2.IsDeleted = true

	all := reg.GetAllExtensions()
	if len(all) != 2 {
		t.Fatalf("expected 2 active extensions, got %d", len(all))
	}

	names := make(map[string]bool)
	for _, e := range all {
		names[e.Name] = true
	}
	if names["ext2"] {
		t.Fatal("deleted extension should not be returned")
	}
	if !names["ext1"] || !names["ext3"] {
		t.Fatal("expected ext1 and ext3")
	}
}

// ========== DependencyGraph Cache Tests ==========

func TestDependencyGraph_Invalidate(t *testing.T) {
	dg := &DependencyGraph{
		sortedNames: []string{"a", "b"},
		valid:       true,
	}

	if !dg.IsValid() {
		t.Fatal("expected valid before invalidate")
	}

	dg.Invalidate()

	if dg.IsValid() {
		t.Fatal("expected invalid after invalidate")
	}
}

func TestExtRegistry_CacheDependencyOrder(t *testing.T) {
	reg := NewExtRegistry()

	sorted := []string{"core", "coc7", "utils"}
	reg.CacheDependencyOrder(sorted)

	order := reg.GetDependencyOrder()
	if len(order) != 3 {
		t.Fatalf("expected 3 items, got %d", len(order))
	}
	if order[0] != "core" || order[1] != "coc7" || order[2] != "utils" {
		t.Fatalf("unexpected order: %v", order)
	}

	// Modify original slice shouldn't affect cached
	sorted[0] = "modified"
	order2 := reg.GetDependencyOrder()
	if order2[0] != "core" {
		t.Fatal("cache should be independent of original slice")
	}
}

// ========== getPriority Tests ==========

func TestGetPriority(t *testing.T) {
	tests := []struct {
		name     string
		ext      *types.ExtInfo
		wantMore *types.ExtInfo // ext should have higher priority than wantMore
	}{
		{
			"Core > System",
			&types.ExtInfo{Category: types.ExtCategoryCore},
			&types.ExtInfo{Category: types.ExtCategorySystem},
		},
		{
			"System > Utility",
			&types.ExtInfo{Category: types.ExtCategorySystem},
			&types.ExtInfo{Category: types.ExtCategoryUtility},
		},
		{
			"Same category, higher Priority wins",
			&types.ExtInfo{Category: types.ExtCategorySystem, Priority: 200},
			&types.ExtInfo{Category: types.ExtCategorySystem, Priority: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p1 := getPriority(tt.ext)
			p2 := getPriority(tt.wantMore)
			if p1 <= p2 {
				t.Errorf("expected priority %d > %d", p1, p2)
			}
		})
	}
}

func TestGetPriority_Nil(t *testing.T) {
	p := getPriority(nil)
	if p >= 0 {
		t.Errorf("expected negative priority for nil, got %d", p)
	}
}
