package dice

import (
	"fmt"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/sealdice/smallseal/dice/types"
)

// ExtRegistry 扩展注册表
type ExtRegistry struct {
	mu              sync.RWMutex
	extensions      map[string]*types.ExtInfo // name -> 真实ExtInfo
	wrappers        map[string]*types.ExtInfo // name -> Wrapper
	dependencyGraph *DependencyGraph          // 依赖图缓存

	// 索引优化
	aliasIndex map[string]string // alias -> name
}

// DependencyGraph 依赖图缓存
type DependencyGraph struct {
	mu          sync.RWMutex
	sortedNames []string // 拓扑排序结果
	valid       bool     // 缓存是否有效
}

// NewExtRegistry 创建新的扩展注册表
func NewExtRegistry() *ExtRegistry {
	return &ExtRegistry{
		extensions:      make(map[string]*types.ExtInfo),
		wrappers:        make(map[string]*types.ExtInfo),
		aliasIndex:      make(map[string]string),
		dependencyGraph: &DependencyGraph{},
	}
}

// GetRealExt 解析真实扩展（处理Wrapper和别名）
func (reg *ExtRegistry) GetRealExt(name string) *types.ExtInfo {
	if name == "" {
		return nil
	}

	reg.mu.RLock()
	defer reg.mu.RUnlock()

	return reg.getRealExtLocked(name)
}

// getRealExtLocked 内部查找方法（调用方需持有读锁）
func (reg *ExtRegistry) getRealExtLocked(name string) *types.ExtInfo {
	// 1. 从主表查找
	if ext, ok := reg.extensions[name]; ok {
		if !ext.IsDeleted {
			return ext
		}
	}

	// 2. 如果是Wrapper，解析真实扩展
	if wrapper, ok := reg.wrappers[name]; ok {
		if !wrapper.IsDeleted {
			if realExt, ok := reg.extensions[wrapper.WrappedName]; ok && !realExt.IsDeleted {
				return realExt
			}
		}
	}

	// 3. 检查别名索引（递归查找）
	if realName, ok := reg.aliasIndex[name]; ok {
		return reg.getRealExtLocked(realName)
	}

	// 4. 不区分大小写查找
	nameLower := strings.ToLower(name)
	for extName, ext := range reg.extensions {
		if !ext.IsDeleted && strings.ToLower(extName) == nameLower {
			return ext
		}
	}

	return nil
}

// Register 注册扩展到注册表
func (reg *ExtRegistry) Register(ext *types.ExtInfo) error {
	if ext == nil {
		return fmt.Errorf("extension is nil")
	}
	if ext.Name == "" {
		return fmt.Errorf("extension name is empty")
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	// 检查是否已存在
	if existing, exists := reg.extensions[ext.Name]; exists && !existing.IsDeleted {
		return fmt.Errorf("extension %s already registered", ext.Name)
	}

	// 注册扩展
	reg.extensions[ext.Name] = ext

	// 重建索引
	reg.rebuildIndicesLocked()

	// 使依赖图失效
	reg.dependencyGraph.Invalidate()

	return nil
}

// CreateWrapper 为扩展创建Wrapper
func (reg *ExtRegistry) CreateWrapper(name string) *types.ExtInfo {
	reg.mu.RLock()
	realExt, exists := reg.extensions[name]
	reg.mu.RUnlock()

	if !exists || realExt == nil {
		return nil
	}

	wrapper := &types.ExtInfo{
		Name:        name,
		Aliases:     realExt.Aliases,
		Version:     realExt.Version,
		IsWrapper:   true,
		WrappedName: name,
		CmdMap:      realExt.CmdMap, // 共享CmdMap，零拷贝
		Brief:       realExt.Brief,
		Author:      realExt.Author,
		Official:    realExt.Official,
		Category:    realExt.Category,
	}

	reg.mu.Lock()
	reg.wrappers[name] = wrapper
	reg.mu.Unlock()

	return wrapper
}

// UpdateWrapper 更新Wrapper指向的真实扩展
func (reg *ExtRegistry) UpdateWrapper(name string, newExt *types.ExtInfo) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	wrapper, ok := reg.wrappers[name]
	if !ok {
		return fmt.Errorf("wrapper %s not found", name)
	}

	// 更新Wrapper的关键字段
	wrapper.CmdMap = newExt.CmdMap
	wrapper.Version = newExt.Version
	wrapper.IsDeleted = false

	return nil
}

// CheckConflicts 检查扩展冲突
func (reg *ExtRegistry) CheckConflicts(ext *types.ExtInfo) []string {
	if ext == nil {
		return nil
	}

	reg.mu.RLock()
	defer reg.mu.RUnlock()

	var conflicts []string

	// 1. 检查ConflictWith声明
	for _, conflictName := range ext.ConflictWith {
		if existing := reg.getRealExtInternal(conflictName); existing != nil && !existing.IsDeleted {
			conflicts = append(conflicts, conflictName)
		}
	}

	// 2. 反向检查（其他扩展声明与当前扩展冲突）
	for name, other := range reg.extensions {
		if other.IsDeleted {
			continue
		}
		for _, conflictName := range other.ConflictWith {
			if strings.EqualFold(conflictName, ext.Name) {
				conflicts = append(conflicts, name)
			}
		}
	}

	// 3. 去重返回
	return lo.Uniq(conflicts)
}

// getRealExtInternal 内部使用，不加锁（调用方需持有锁）
// 注意：这是 getRealExtLocked 的别名，保持向后兼容
func (reg *ExtRegistry) getRealExtInternal(name string) *types.ExtInfo {
	return reg.getRealExtLocked(name)
}

// Replace 替换已存在的扩展（用于热重载）
func (reg *ExtRegistry) Replace(name string, newExt *types.ExtInfo) error {
	if newExt == nil {
		return fmt.Errorf("newExt is nil")
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	if _, exists := reg.extensions[name]; !exists {
		return fmt.Errorf("extension %s not found", name)
	}

	reg.extensions[name] = newExt
	reg.rebuildIndicesLocked()
	reg.dependencyGraph.Invalidate()

	return nil
}

// Validate 验证扩展信息
func (reg *ExtRegistry) Validate(ext *types.ExtInfo) error {
	if ext == nil {
		return fmt.Errorf("extension is nil")
	}
	if ext.Name == "" {
		return fmt.Errorf("extension name is empty")
	}
	// 可以添加更多验证逻辑
	return nil
}

// rebuildIndicesLocked 重建索引（调用方需持有写锁）
func (reg *ExtRegistry) rebuildIndicesLocked() {
	// 重建别名索引
	reg.aliasIndex = make(map[string]string)
	for name, ext := range reg.extensions {
		if ext.IsDeleted {
			continue
		}
		for _, alias := range ext.Aliases {
			reg.aliasIndex[alias] = name
		}
	}
}

// GetAllExtensions 获取所有扩展（不包括已删除的）
func (reg *ExtRegistry) GetAllExtensions() []*types.ExtInfo {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	var exts []*types.ExtInfo
	for _, ext := range reg.extensions {
		if !ext.IsDeleted {
			exts = append(exts, ext)
		}
	}
	return exts
}

// InvalidateDependencyGraph 使依赖图缓存失效
func (reg *ExtRegistry) InvalidateDependencyGraph() {
	reg.dependencyGraph.Invalidate()
}

// Invalidate 使依赖图缓存失效
func (dg *DependencyGraph) Invalidate() {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	dg.valid = false
}

// IsValid 检查依赖图缓存是否有效
func (dg *DependencyGraph) IsValid() bool {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	return dg.valid
}
