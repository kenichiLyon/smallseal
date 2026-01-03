package exts

import (
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/sealdice/smallseal/dice/attrs"
	"github.com/sealdice/smallseal/dice/types"
	"github.com/sealdice/smallseal/utils"
)

// dnd5eTextMap 返回完整的 DND5E 文本模板
func dnd5eTextMap() types.TextTemplateWithWeightDict {
	toItem := func(text string) types.TextTemplateItem {
		return types.TextTemplateItem{text, 1}
	}

	return types.TextTemplateWithWeightDict{
		"COC": types.TextTemplateWithWeight{
			"属性设置": []types.TextTemplateItem{
				toItem("SET {$t玩家} {$t规则模板} {$t有效数量}"),
			},
			"属性设置_列出": []types.TextTemplateItem{
				toItem("LIST {$t属性信息}"),
			},
			"属性设置_列出_未发现记录": []types.TextTemplateItem{
				toItem("LIST_EMPTY"),
			},
			"属性设置_列出_隐藏提示": []types.TextTemplateItem{
				toItem("HIDDEN {$t数量} {$t判定值}"),
			},
			"属性设置_删除": []types.TextTemplateItem{
				toItem("DEL {$t属性列表} FAIL {$t失败数量}"),
			},
			"属性设置_清除": []types.TextTemplateItem{
				toItem("CLEAR {$t玩家} {$t数量}"),
			},
			"属性设置_增减": []types.TextTemplateItem{
				toItem("MOD {$t变更列表}"),
			},
			"属性设置_增减_单项": []types.TextTemplateItem{
				toItem("ITEM {$t属性}: {$t旧值}->{$t新值} ({$t增加或减少}{$t表达式}={$t变化值})"),
			},
		},
		"DND": types.TextTemplateWithWeight{
			"检定": []types.TextTemplateItem{
				toItem("{$t玩家}的\"{$t技能}\"检定（DND5E）结果为: {$t检定过程文本} = {$t检定结果}"),
			},
			"检定_单项结果文本": []types.TextTemplateItem{
				toItem("{$t检定过程文本} = {$t检定结果}"),
			},
			"检定_多轮": []types.TextTemplateItem{
				toItem("对{$t玩家}的\"{$t技能}\"进行了{$t次数}次检定（DND5E），结果为:\n{$t结果文本}"),
			},
			"制卡_分隔符": []types.TextTemplateItem{
				toItem("\n\n"),
			},
			"制卡_预设模式": []types.TextTemplateItem{
				toItem("{$t玩家}的DND5E人物属性:\n{$t制卡结果文本}"),
			},
			"制卡_自由分配模式": []types.TextTemplateItem{
				toItem("{$t玩家}的DND5E属性骰点结果:\n{$t制卡结果文本}"),
			},
			"先攻_设置_前缀": []types.TextTemplateItem{
				toItem("{$t玩家}设置先攻:\n"),
			},
			"先攻_设置_格式错误": []types.TextTemplateItem{
				toItem("先攻设置格式错误"),
			},
			"先攻_设置_指定单位": []types.TextTemplateItem{
				toItem("设置{$t目标}的先攻为{$t点数}"),
			},
			"先攻_查看_前缀": []types.TextTemplateItem{
				toItem("当前先攻列表:\n"),
			},
			"先攻_移除_前缀": []types.TextTemplateItem{
				toItem("移除了以下单位:\n"),
			},
			"先攻_清除列表": []types.TextTemplateItem{
				toItem("先攻列表已清空"),
			},
			"先攻_下一回合": []types.TextTemplateItem{
				toItem("{$t新轮开始提示}{$t当前回合角色名}的回合结束，轮到{$t下一回合角色名}"),
			},
			"先攻_新轮开始提示": []types.TextTemplateItem{
				toItem("===新的一轮===\n"),
			},
			"死亡豁免_结局_伤势稳定": []types.TextTemplateItem{
				toItem("伤势稳定，脱离危险"),
			},
			"死亡豁免_结局_角色死亡": []types.TextTemplateItem{
				toItem("角色死亡"),
			},
			"死亡豁免_D20_附加语": []types.TextTemplateItem{
				toItem("奇迹发生！恢复1点生命值并苏醒"),
			},
			"死亡豁免_D1_附加语": []types.TextTemplateItem{
				toItem("雪上加霜！失败次数+2"),
			},
			"死亡豁免_成功_附加语": []types.TextTemplateItem{
				toItem("坚持住！"),
			},
			"死亡豁免_失败_附加语": []types.TextTemplateItem{
				toItem("情况不妙..."),
			},
			"受到伤害_超过HP上限_附加语": []types.TextTemplateItem{
				toItem("受到{$t伤害点数}点伤害，超过生命上限，角色死亡"),
			},
			"受到伤害_昏迷中_附加语": []types.TextTemplateItem{
				toItem("昏迷中受到{$t伤害点数}点伤害，死亡豁免失败+1"),
			},
			"受到伤害_进入昏迷_附加语": []types.TextTemplateItem{
				toItem("受到{$t伤害点数}点致命伤害，陷入昏迷"),
			},
		},
		"核心": types.TextTemplateWithWeight{
			"骰点": []types.TextTemplateItem{
				toItem("{$t结果文本}"),
			},
			"骰点_单项结果文本": []types.TextTemplateItem{
				toItem("{$t表达式文本}{$t计算过程}={$t计算结果}"),
			},
			"骰点_原因": []types.TextTemplateItem{
				toItem("{$t原因句子}"),
			},
			"提示_私聊不可用": []types.TextTemplateItem{
				toItem("此功能在私聊中不可用"),
			},
			"暗骰_群内": []types.TextTemplateItem{
				toItem("{$t玩家}进行了暗骰"),
			},
			"暗骰_私聊_前缀": []types.TextTemplateItem{
				toItem("暗骰结果: "),
			},
		},
	}
}

// newDnd5eCmdTestContext 创建用于测试 DND5E 命令的上下文
func newDnd5eCmdTestContext(t *testing.T) (*types.MsgContext, *types.Message, *stubDice) {
	t.Helper()

	asset, ok := BuiltinGameSystemTemplateAsset("dnd5e.yaml")
	require.True(t, ok)

	tmpl, err := types.LoadGameSystemTemplateFromData(asset.Data, asset.Filename)
	require.NoError(t, err)

	stub := newStubDice(tmpl)
	RegisterBuiltinExtDnd5e(stub)

	am := &attrs.AttrsManager{}
	am.SetIO(attrs.NewMemoryAttrsIO())

	group := &types.GroupInfo{
		GroupId: "group-dnd5e-cmd-test",
		System:  "dnd5e",
		BotList: &utils.SyncMap[string, bool]{},
		Active:  true,
	}

	player := &types.GroupPlayerInfo{
		UserId: "user-dnd5e-cmd-test",
		Name:   "冒险者",
	}

	ctx := &types.MsgContext{
		Dice:            stub,
		AttrsManager:    am,
		Group:           group,
		Player:          player,
		TextTemplateMap: dnd5eTextMap(),
		GameSystem:      tmpl,
	}

	// 激活扩展
	ext := stub.extensions["dnd5e"]
	group.ExtActive(ext)

	_, err = am.Load(group.GroupId, player.UserId)
	require.NoError(t, err)

	msg := &types.Message{
		MessageType: "group",
		GroupID:     group.GroupId,
		Platform:    "test",
		Time:        time.Now().Unix(),
		Sender: types.SenderBase{
			UserID:   player.UserId,
			Nickname: player.Name,
		},
	}

	return ctx, msg, stub
}

func executeDnd5eCommand(t *testing.T, stub *stubDice, ctx *types.MsgContext, msg *types.Message, raw string, names ...string) (types.CmdExecuteResult, string) {
	t.Helper()

	ext, ok := stub.extensions["dnd5e"]
	require.True(t, ok, "dnd5e extension should be registered")

	cmdArgs := types.CommandParse(raw, names, []string{".", "。"}, "", false)
	require.NotNilf(t, cmdArgs, "failed to parse command %q", raw)

	cmd, ok := ext.CmdMap[cmdArgs.Command]
	require.Truef(t, ok, "command %q not found in dnd5e extension", cmdArgs.Command)

	ctx.CommandId++

	prev := len(stub.replies)
	result := cmd.Solve(ctx, msg, cmdArgs)

	require.Truef(t, result.Matched, "command %q should match", raw)

	if len(stub.replies) > prev {
		reply := stub.replies[len(stub.replies)-1]
		return result, reply.Segments.ToText()
	}

	return result, ""
}

// ================= DND 制卡测试 =================

func TestDnd5eDndCard(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".dnd", "dnd", "dndx")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "=")
}

func TestDnd5eDndCardMultiple(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".dnd 3", "dnd", "dndx")
	require.NotEmpty(t, reply)
	// 三组数据
	lines := strings.Split(reply, "\n")
	require.GreaterOrEqual(t, len(lines), 3)
}

func TestDnd5eDndxCard(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	ext := stub.extensions["dnd5e"]
	cmd := ext.CmdMap["dndx"]

	cmdArgs := types.CommandParse(".dndx", []string{"dnd", "dndx"}, []string{".", "。"}, "", false)
	require.NotNil(t, cmdArgs)

	ctx.CommandId++
	prev := len(stub.replies)
	result := cmd.Solve(ctx, msg, cmdArgs)

	require.True(t, result.Matched)

	if len(stub.replies) > prev {
		reply := stub.replies[len(stub.replies)-1].Segments.ToText()
		require.NotEmpty(t, reply)
		require.Contains(t, reply, "力量")
		require.Contains(t, reply, "敏捷")
	}
}

// ================= RC 检定测试 =================

func TestDnd5eRcHelp(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	result, _ := executeDnd5eCommand(t, stub, ctx, msg, ".rc help", "rc", "ra", "drc")
	require.True(t, result.ShowHelp)
}

func TestDnd5eRcBasic(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 5", "rc", "ra", "drc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestDnd5eRcWithAdvantage(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 优势 5", "rc", "ra", "drc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestDnd5eRcWithDisadvantage(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 劣势 5", "rc", "ra", "drc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestDnd5eRcMultiple(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 多轮检定会返回多个结果
	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 3# 5", "rc", "ra", "drc")
	require.NotEmpty(t, reply)
	// 验证有检定结果
	require.Contains(t, reply, "检定")
}

// ================= SS 法术位测试 =================

func TestDnd5eSsHelp(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	result, _ := executeDnd5eCommand(t, stub, ctx, msg, ".ss help", "ss", "spellslots", "dss", "法术位")
	require.True(t, result.ShowHelp)
}

func TestDnd5eSsInit(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots", "dss", "法术位")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "设置法术位")
	require.Contains(t, reply, "1环4个")
	require.Contains(t, reply, "2环3个")
	require.Contains(t, reply, "3环2个")
}

func TestDnd5eSsShow(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先初始化
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots", "dss", "法术位")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss", "spellslots", "dss", "法术位")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "法术位状况")
}

func TestDnd5eSsSet(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ss set 2环 5", "ss", "spellslots", "dss", "法术位")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "设置法术位")
	require.Contains(t, reply, "2环5个")
}

func TestDnd5eSsConsume(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先初始化
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots", "dss", "法术位")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ss 1环 -1", "ss", "spellslots", "dss", "法术位")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "消耗")
}

func TestDnd5eSsRecover(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先初始化并消耗
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots", "dss", "法术位")
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss 1环 -2", "ss", "spellslots", "dss", "法术位")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ss 1环 +1", "ss", "spellslots", "dss", "法术位")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "恢复")
}

func TestDnd5eSsRest(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先初始化并消耗
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots", "dss", "法术位")
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss 1环 -2", "ss", "spellslots", "dss", "法术位")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ss rest", "ss", "spellslots", "dss", "法术位")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "恢复")
}

func TestDnd5eSsClr(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先初始化
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots", "dss", "法术位")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ss clr", "ss", "spellslots", "dss", "法术位")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "清除")
}

// ================= Cast 法术使用测试 =================

func TestDnd5eCastBasic(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先初始化法术位
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots", "dss", "法术位")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast", "dcast")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "消耗")
}

func TestDnd5eCastMultiple(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先初始化法术位
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots", "dss", "法术位")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".cast 1 2", "cast", "dcast")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "消耗")
}

func TestDnd5eCastNotEnough(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 初始化少量法术位
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 1", "ss", "spellslots", "dss", "法术位")
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast", "dcast")

	// 再次消耗应该失败
	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast", "dcast")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "无法消耗")
}

// ================= LongRest 长休测试 =================

func TestDnd5eLongRestHelp(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	result, _ := executeDnd5eCommand(t, stub, ctx, msg, ".长休 help", "长休", "longrest", "dlongrest")
	require.True(t, result.ShowHelp)
}

func TestDnd5eLongRestNoHpmax(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".长休", "长休", "longrest", "dlongrest")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "没有设置hpmax")
}

func TestDnd5eLongRestWithHpmax(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 设置 hpmax 和 hp
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	VarSetValueInt64(ctx, "hpmax", 30)
	VarSetValueInt64(ctx, "hp", 15)

	// 验证设置
	require.Equal(t, int64(15), attrIntValue(t, ctx, attrsItem, "hp"))

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".长休", "长休", "longrest", "dlongrest")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "hp得到了恢复")

	// 验证 hp 恢复到 hpmax
	require.Equal(t, int64(30), attrIntValue(t, ctx, attrsItem, "hp"))
}

func TestDnd5eLongRestWithSpellSlots(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 初始化法术位并消耗
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3", "ss", "spellslots", "dss", "法术位")
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast", "dcast")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".长休", "长休", "longrest", "dlongrest")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "法术位得到了恢复")
}

// ================= DS 死亡豁免测试 =================

func TestDnd5eDsNoHp(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ds", "ds", "死亡豁免")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "未设置生命值")
}

func TestDnd5eDsHpNotZero(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	VarSetValueInt64(ctx, "hp", 10)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ds", "ds", "死亡豁免")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "生命值大于0")
}

func TestDnd5eDsBasic(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	VarSetValueInt64(ctx, "hp", 0)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ds", "ds", "死亡豁免")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "死亡豁免检定")
}

func TestDnd5eDsWithAdvantage(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	VarSetValueInt64(ctx, "hp", 0)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ds 优势", "ds", "死亡豁免")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "死亡豁免检定")
}

func TestDnd5eDsStat(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	VarSetValueInt64(ctx, "hp", 0)

	// 先进行一次豁免
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ds", "ds", "死亡豁免")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ds stat", "ds", "死亡豁免")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "死亡豁免情况")
}

func TestDnd5eDsModifySuccess(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ds s+1", "ds", "死亡豁免")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "成功")
}

func TestDnd5eDsModifyFailure(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ds f+1", "ds", "死亡豁免")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "失败")
}

// ================= RI 先攻设置测试 =================

func TestDnd5eRiHelp(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	result, _ := executeDnd5eCommand(t, stub, ctx, msg, ".ri help", "ri")
	require.True(t, result.ShowHelp)
}

func TestDnd5eRiSelf(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ri", "ri")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "设置先攻")
}

func TestDnd5eRiWithName(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ri 小明", "ri")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "小明")
}

func TestDnd5eRiWithValue(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ri 15 小明", "ri")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "小明")
	require.Contains(t, reply, "15")
}

func TestDnd5eRiWithModifier(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ri +3 小明", "ri")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "小明")
}

func TestDnd5eRiWithExpression(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ri =d20+5 小明", "ri")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "小明")
}

func TestDnd5eRiMultiple(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ri 小明, +2 李四, 15 王五", "ri")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "小明")
	require.Contains(t, reply, "李四")
	require.Contains(t, reply, "王五")
}

func TestDnd5eRiWithAdvantage(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".ri 优势 小明", "ri")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "小明")
}

// ================= Init 先攻管理测试 =================

func TestDnd5eInitList(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先添加一些先攻
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ri 15 小明, 12 李四", "ri")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".init", "init")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "先攻列表")
	require.Contains(t, reply, "小明")
	require.Contains(t, reply, "李四")
}

func TestDnd5eInitEnd(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先添加一些先攻
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ri 15 小明, 12 李四", "ri")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".init end", "init")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "回合结束")
}

func TestDnd5eInitDel(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先添加一些先攻
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ri 15 小明, 12 李四", "ri")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".init del 李四", "init")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "移除")
	require.Contains(t, reply, "李四")
}

func TestDnd5eInitSet(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".init set 小明 18", "init")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "小明")
	require.Contains(t, reply, "18")
}

func TestDnd5eInitClr(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先添加一些先攻
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ri 15 小明, 12 李四", "ri")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".init clr", "init")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "清空")
}

func TestDnd5eInitHelp(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	result, _ := executeDnd5eCommand(t, stub, ctx, msg, ".init help", "init")
	require.True(t, result.ShowHelp)
}

// ================= Buff 测试 =================

func TestDnd5eBuffHelp(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	result, _ := executeDnd5eCommand(t, stub, ctx, msg, ".buff help", "buff", "dbuff")
	require.True(t, result.ShowHelp)
}

func TestDnd5eBuffSet(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".buff 力量:4", "buff", "dbuff")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "SET")
}

func TestDnd5eBuffShow(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先设置buff
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".buff 力量:4", "buff", "dbuff")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".buff show", "buff", "dbuff")
	require.NotEmpty(t, reply)
}

func TestDnd5eBuffDel(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先设置buff
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".buff 力量:4", "buff", "dbuff")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".buff del 力量", "buff", "dbuff")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "DEL")
}

func TestDnd5eBuffClr(t *testing.T) {
	ctx, msg, stub := newDnd5eCmdTestContext(t)

	// 先设置buff
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".buff 力量:4", "buff", "dbuff")

	_, reply := executeDnd5eCommand(t, stub, ctx, msg, ".buff clr", "buff", "dbuff")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "CLEAR")
}
