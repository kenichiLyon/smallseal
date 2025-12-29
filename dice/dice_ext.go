package dice

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/sealdice/smallseal/dice/types"
)

// RegisterExtension 注册扩展
// 重构后的版本：集成ExtRegistry，支持冲突检测和Wrapper机制
func (d *Dice) RegisterExtension(extInfo *types.ExtInfo) {
	if extInfo == nil {
		panic("RegisterExtension: extInfo is nil")
	}

	// 1. 验证扩展信息
	if err := d.ExtRegistry.Validate(extInfo); err != nil {
		panic(fmt.Sprintf("RegisterExtension: %v", err))
	}

	// 2. 检查名称和别名冲突
	for _, name := range append(extInfo.Aliases, extInfo.Name) {
		if collide := d.ExtFind(name, false); collide != nil {
			panicMsg := fmt.Sprintf("扩展<%s>的名字%q与现存扩展<%s>冲突", extInfo.Name, name, collide.Name)
			panic(panicMsg)
		}
	}

	// 3. 注册到ExtRegistry
	if err := d.ExtRegistry.Register(extInfo); err != nil {
		panic(fmt.Sprintf("RegisterExtension: %v", err))
	}

	// 4. 保持ExtList向后兼容
	d.ExtList = append(d.ExtList, extInfo)

	// 5. 递增全局版本号
	d.ExtRegistryVersion.Add(1)

	// 6. 使伴随激活图失效（新扩展可能有 ActiveWith）
	d.invalidateActiveWithGraph()

	// 7. 设置加载时间戳
	extInfo.LoadedAt = time.Now().Unix()

	// 8. 触发OnLoad回调
	if extInfo.OnLoad != nil && !extInfo.IsWrapper {
		extInfo.OnLoad()
		extInfo.IsLoaded = true
	}
}

// ReloadExtension 重载扩展（用于JS扩展热重载）
func (d *Dice) ReloadExtension(name string, newExt *types.ExtInfo) error {
	if newExt == nil {
		return fmt.Errorf("newExt is nil")
	}

	// 1. 查找旧扩展
	oldExt := d.ExtRegistry.GetRealExt(name)
	if oldExt == nil {
		return fmt.Errorf("extension %s not found", name)
	}

	// 2. 调用旧扩展的OnUnload
	if oldExt.OnUnload != nil {
		oldExt.OnUnload()
	}

	// 3. 标记旧扩展为删除
	oldExt.IsDeleted = true
	oldExt.IsLoaded = false

	// 4. 替换扩展（使用封装的方法）
	if err := d.ExtRegistry.Replace(name, newExt); err != nil {
		return fmt.Errorf("failed to replace extension: %w", err)
	}

	// 5. 更新或创建Wrapper
	if err := d.ExtRegistry.UpdateWrapper(name, newExt); err != nil {
		// 如果Wrapper不存在，创建一个
		d.ExtRegistry.CreateWrapper(name)
	}

	// 6. 更新ExtList中的引用
	for i, ext := range d.ExtList {
		if ext.Name == name {
			d.ExtList[i] = newExt
			break
		}
	}

	// 7. 递增全局版本号
	d.ExtRegistryVersion.Add(1)

	// 8. 使伴随激活图失效
	d.invalidateActiveWithGraph()

	// 9. 标记所有群组需要重新同步（延迟同步机制）
	d.MarkAllGroupsDirty()

	// 10. 设置加载时间戳并触发OnLoad
	newExt.LoadedAt = time.Now().Unix()
	if newExt.OnLoad != nil {
		newExt.OnLoad()
		newExt.IsLoaded = true
	}

	return nil
}

// UnregisterExtension 卸载扩展
func (d *Dice) UnregisterExtension(name string) error {
	// 1. 查找扩展
	ext := d.ExtRegistry.GetRealExt(name)
	if ext == nil {
		return fmt.Errorf("extension %s not found", name)
	}

	// 2. 核心扩展不能卸载
	if ext.Category == types.ExtCategoryCore {
		return fmt.Errorf("core extension %s cannot be unregistered", name)
	}

	// 3. 调用OnUnload回调
	if ext.OnUnload != nil {
		ext.OnUnload()
	}

	// 4. 标记为删除（软删除）
	ext.IsDeleted = true
	ext.IsLoaded = false

	// 5. 从ExtList中移除
	newList := make([]*types.ExtInfo, 0, len(d.ExtList))
	for _, e := range d.ExtList {
		if e.Name != name {
			newList = append(newList, e)
		}
	}
	d.ExtList = newList

	// 6. 使依赖图失效
	d.ExtRegistry.InvalidateDependencyGraph()

	// 7. 递增全局版本号
	d.ExtRegistryVersion.Add(1)

	// 8. 使伴随激活图失效
	d.invalidateActiveWithGraph()

	// 9. 标记所有群组需要重新同步
	d.MarkAllGroupsDirty()

	return nil
}

// ExtFind 根据名称或别名查找扩展
func (d *Dice) ExtFind(s string, fromJS bool) *types.ExtInfo {
	find := func(_ string) *types.ExtInfo {
		for _, i := range d.ExtList {
			// 名字匹配，优先级最高
			if i.Name == s {
				return i
			}
		}
		for _, i := range d.ExtList {
			// 别名匹配，优先级次之
			if slices.Contains(i.Aliases, s) {
				return i
			}
		}
		for _, i := range d.ExtList {
			// 忽略大小写匹配，优先级最低
			if strings.EqualFold(i.Name, s) || slices.Contains(i.Aliases, strings.ToLower(s)) {
				return i
			}
		}
		return nil
	}
	ext := find(s)
	if ext != nil && ext.Official && fromJS {
		// return a copy of the official extension
		cmdMap := make(types.CmdMapCls, len(ext.CmdMap))
		for s2, info := range ext.CmdMap {
			cmdMap[s2] = &types.CmdItemInfo{
				Name:                    info.Name,
				ShortHelp:               info.ShortHelp,
				Help:                    info.Help,
				HelpFunc:                info.HelpFunc,
				AllowDelegate:           info.AllowDelegate,
				DisabledInPrivate:       info.DisabledInPrivate,
				EnableExecuteTimesParse: info.EnableExecuteTimesParse,
				IsJsSolveFunc:           info.IsJsSolveFunc,
				Solve:                   info.Solve,
				Raw:                     info.Raw,
				CheckCurrentBotOn:       info.CheckCurrentBotOn,
				CheckMentionOthers:      info.CheckMentionOthers,
			}
		}
		return &types.ExtInfo{
			Name:       ext.Name,
			Aliases:    ext.Aliases,
			Author:     ext.Author,
			Version:    ext.Version,
			AutoActive: ext.AutoActive,
			CmdMap:     cmdMap,
			Brief:      ext.Brief,
			Official:   ext.Official,
		}
	}
	return ext
}

func (d *Dice) GetExtList() []*types.ExtInfo {
	return d.ExtList
}
