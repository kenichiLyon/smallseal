package dice

import (
	"errors"
	"fmt"
	"sort"

	"github.com/sealdice/smallseal/dice/types"
)

// ResolveWithDependencies 解析扩展依赖并进行拓扑排序
// 返回按依赖顺序排列的扩展列表（依赖在前，被依赖在后）
func (reg *ExtRegistry) ResolveWithDependencies(exts []*types.ExtInfo) ([]*types.ExtInfo, error) {
	if len(exts) == 0 {
		return nil, nil
	}

	// 1. 构建扩展映射和依赖图
	extMap := make(map[string]*types.ExtInfo)
	graph := make(map[string][]string) // 被依赖者 -> 依赖者列表
	inDegree := make(map[string]int)   // 入度计数

	// 2. 使用队列递归收集所有扩展及其依赖
	toProcess := make([]*types.ExtInfo, 0, len(exts))
	for _, ext := range exts {
		if ext == nil || ext.IsDeleted {
			continue
		}
		toProcess = append(toProcess, ext)
	}

	for len(toProcess) > 0 {
		ext := toProcess[0]
		toProcess = toProcess[1:]

		// 已处理过则跳过
		if _, exists := extMap[ext.Name]; exists {
			continue
		}

		extMap[ext.Name] = ext
		inDegree[ext.Name] = 0

		// 处理此扩展的依赖
		for _, dep := range ext.DependsOn {
			depExt := reg.GetRealExt(dep.Name)
			if depExt == nil {
				if !dep.Optional {
					return nil, fmt.Errorf("extension %s requires missing dependency: %s", ext.Name, dep.Name)
				}
				continue // 跳过可选依赖
			}

			// 如果依赖尚未处理，加入队列
			if _, exists := extMap[dep.Name]; !exists {
				toProcess = append(toProcess, depExt)
			}

			// 添加边：dep.Name -> ext.Name（dep被ext依赖）
			graph[dep.Name] = append(graph[dep.Name], ext.Name)
			inDegree[ext.Name]++
		}
	}

	// 3. Kahn算法拓扑排序
	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// 按优先级排序入度为0的节点（高优先级在前）
	sortByPriority(queue, extMap)

	var sorted []string
	for len(queue) > 0 {
		// 取出第一个节点
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		// 更新邻接节点的入度
		for _, next := range graph[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
				// 保持优先级排序
				sortByPriority(queue, extMap)
			}
		}
	}

	// 4. 检测循环依赖
	if len(sorted) != len(extMap) {
		// 找出循环依赖的扩展
		var cycle []string
		for name := range extMap {
			found := false
			for _, s := range sorted {
				if s == name {
					found = true
					break
				}
			}
			if !found {
				cycle = append(cycle, name)
			}
		}
		return nil, fmt.Errorf("circular dependency detected among: %v", cycle)
	}

	// 5. 转换为ExtInfo列表
	result := make([]*types.ExtInfo, 0, len(sorted))
	for _, name := range sorted {
		if ext, ok := extMap[name]; ok {
			result = append(result, ext)
		}
	}

	return result, nil
}

// sortByPriority 按优先级对扩展名列表排序（高优先级在前）
func sortByPriority(names []string, extMap map[string]*types.ExtInfo) {
	sort.Slice(names, func(i, j int) bool {
		extI := extMap[names[i]]
		extJ := extMap[names[j]]
		return getPriority(extI) > getPriority(extJ)
	})
}

// getPriority 计算扩展的综合优先级
// 规则：Category权重 > Priority字段 > 名称排序
func getPriority(ext *types.ExtInfo) int {
	if ext == nil {
		return -1000000
	}

	// Category权重：Core(0) > System(1) > Utility(2)
	// 为了让Core优先级最高，我们用负数：越小的Category值，优先级越高
	categoryWeight := (2 - int(ext.Category)) * 1000000

	// Priority字段直接加上
	return categoryWeight + ext.Priority
}

// ValidateDependencies 验证扩展依赖是否满足
func (reg *ExtRegistry) ValidateDependencies(ext *types.ExtInfo) error {
	if ext == nil {
		return errors.New("extension is nil")
	}

	for _, dep := range ext.DependsOn {
		depExt := reg.GetRealExt(dep.Name)
		if depExt == nil {
			if !dep.Optional {
				return fmt.Errorf("missing required dependency: %s", dep.Name)
			}
			continue
		}

		// 检查版本要求（如果有）
		if dep.MinVer != "" {
			// TODO: 实现版本比较逻辑
			// 目前跳过版本检查
		}
	}

	return nil
}

// GetDependencyOrder 获取扩展的依赖顺序（用于卸载时反向执行）
func (reg *ExtRegistry) GetDependencyOrder() []string {
	dg := reg.dependencyGraph
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	if !dg.valid || len(dg.sortedNames) == 0 {
		return nil
	}

	// 返回副本
	result := make([]string, len(dg.sortedNames))
	copy(result, dg.sortedNames)
	return result
}

// CacheDependencyOrder 缓存依赖顺序
func (reg *ExtRegistry) CacheDependencyOrder(sorted []string) {
	dg := reg.dependencyGraph
	dg.mu.Lock()
	defer dg.mu.Unlock()

	dg.sortedNames = make([]string, len(sorted))
	copy(dg.sortedNames, sorted)
	dg.valid = true
}

// GetActiveDependencies 获取扩展的所有活跃依赖（递归）
func (reg *ExtRegistry) GetActiveDependencies(extName string) []*types.ExtInfo {
	ext := reg.GetRealExt(extName)
	if ext == nil {
		return nil
	}

	visited := make(map[string]bool)
	var deps []*types.ExtInfo

	var collectDeps func(e *types.ExtInfo)
	collectDeps = func(e *types.ExtInfo) {
		for _, dep := range e.DependsOn {
			if visited[dep.Name] {
				continue
			}
			visited[dep.Name] = true

			depExt := reg.GetRealExt(dep.Name)
			if depExt != nil && !depExt.IsDeleted {
				deps = append(deps, depExt)
				collectDeps(depExt)
			}
		}
	}

	collectDeps(ext)
	return deps
}
