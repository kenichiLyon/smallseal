package dice

import (
	"sync"

	"github.com/sealdice/smallseal/dice/types"
	"github.com/sealdice/smallseal/utils"
)

const maxChainDepth = 10

// ActiveWithGraph 描述扩展伴随关系：A -> [B,C] 表示激活 A 时需要附带 B、C
type ActiveWithGraph = utils.SyncMap[string, []string]

// activeWithGraphCache 用于缓存伴随激活图
type activeWithGraphCache struct {
	mu    sync.RWMutex
	graph *ActiveWithGraph
}

// rebuildActiveWithGraph 重建伴随激活图
func (d *Dice) rebuildActiveWithGraph() {
	d.activeWithGraphMu.Lock()
	defer d.activeWithGraphMu.Unlock()
	d.activeWithGraph = BuildActiveWithGraph(d.ExtList)
}

// getActiveWithGraph 获取伴随激活图（懒加载）
func (d *Dice) getActiveWithGraph() *ActiveWithGraph {
	d.activeWithGraphMu.RLock()
	graph := d.activeWithGraph
	d.activeWithGraphMu.RUnlock()
	if graph != nil {
		return graph
	}

	d.activeWithGraphMu.Lock()
	defer d.activeWithGraphMu.Unlock()
	if d.activeWithGraph == nil {
		d.activeWithGraph = BuildActiveWithGraph(d.ExtList)
	}
	return d.activeWithGraph
}

// invalidateActiveWithGraph 使伴随激活图失效
func (d *Dice) invalidateActiveWithGraph() {
	d.activeWithGraphMu.Lock()
	d.activeWithGraph = nil
	d.activeWithGraphMu.Unlock()
}

// BuildActiveWithGraph 根据扩展列表构建伴随激活图
// 返回反向图：A -> [B,C] 表示 B、C 跟随 A（当 A 开启时，B、C 也开启）
func BuildActiveWithGraph(exts []*types.ExtInfo) *ActiveWithGraph {
	graph := new(ActiveWithGraph)
	// 构建反向图：如果 ext 跟随 target，则在 graph[target] 中加入 ext
	for _, ext := range exts {
		if ext == nil || ext.IsDeleted {
			continue
		}
		for _, target := range ext.ActiveWith {
			followers, _ := graph.Load(target)
			followers = append(followers, ext.Name)
			graph.Store(target, followers)
		}
	}
	return graph
}

// CollectChainedNames 收集所有跟随 base 扩展的伴随扩展（递归查找，返回拓扑顺序）
func CollectChainedNames(graph *ActiveWithGraph, base string, depthLimit int) []string {
	visited := map[string]bool{}
	visiting := map[string]bool{}
	var order []string

	if graph == nil {
		return order
	}

	var dfs func(name string, depth int)
	dfs = func(name string, depth int) {
		if depth > depthLimit {
			// 超过深度限制，停止展开
			return
		}
		if visiting[name] {
			// 检测到循环，跳过
			return
		}
		if visited[name] {
			return
		}
		visiting[name] = true
		followers, _ := graph.Load(name)
		for _, follower := range followers {
			dfs(follower, depth+1)
		}
		visiting[name] = false
		visited[name] = true
		if name != base {
			order = append(order, name)
		}
	}

	dfs(base, 0)
	return order
}

// GetFollowerExtensions 获取跟随指定扩展的所有扩展
func (d *Dice) GetFollowerExtensions(extName string) []*types.ExtInfo {
	graph := d.getActiveWithGraph()
	names := CollectChainedNames(graph, extName, maxChainDepth)

	var exts []*types.ExtInfo
	for _, name := range names {
		if ext := d.ExtFind(name, false); ext != nil {
			exts = append(exts, ext)
		}
	}
	return exts
}
