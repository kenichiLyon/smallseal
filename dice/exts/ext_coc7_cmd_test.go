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

// coc7TextMap 返回完整的 COC7 文本模板
func coc7TextMap() types.TextTemplateWithWeightDict {
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
				toItem("DEL {$t删除列表} FAIL {$t失败数量}"),
			},
			"属性设置_清除": []types.TextTemplateItem{
				toItem("CLEAR {$t玩家}"),
			},
			"属性设置_增减": []types.TextTemplateItem{
				toItem("MOD {$t变更列表}"),
			},
			"属性设置_增减_单项": []types.TextTemplateItem{
				toItem("ITEM {$t属性}: {$t旧值}->{$t新值} ({$t增加或减少}{$t表达式}={$t变化值})"),
			},
			"检定": []types.TextTemplateItem{
				toItem("{$t玩家}进行检定: {$t属性表达式文本} {$t结果文本}"),
			},
			"检定_格式错误": []types.TextTemplateItem{
				toItem("格式错误，请使用.ra <属性表达式>"),
			},
			"检定_单项结果文本": []types.TextTemplateItem{
				toItem("{$t检定表达式文本}{$t检定计算过程}={$tD100}/{$t判定值} {$t判定结果}"),
			},
			"检定_多轮": []types.TextTemplateItem{
				toItem("{$t玩家}进行{$t次数}次检定:\n{$t结果文本}"),
			},
			"检定_暗中_群内": []types.TextTemplateItem{
				toItem("{$t玩家}进行了暗中检定"),
			},
			"检定_暗中_私聊_前缀": []types.TextTemplateItem{
				toItem("暗骰结果: "),
			},
			"判定_大失败": []types.TextTemplateItem{
				toItem("大失败"),
			},
			"判定_失败": []types.TextTemplateItem{
				toItem("失败"),
			},
			"判定_成功_普通": []types.TextTemplateItem{
				toItem("成功"),
			},
			"判定_成功_困难": []types.TextTemplateItem{
				toItem("困难成功"),
			},
			"判定_成功_极难": []types.TextTemplateItem{
				toItem("极难成功"),
			},
			"判定_大成功": []types.TextTemplateItem{
				toItem("大成功"),
			},
			"判定_简短_大失败": []types.TextTemplateItem{
				toItem("大失败"),
			},
			"判定_简短_失败": []types.TextTemplateItem{
				toItem("失败"),
			},
			"判定_简短_成功_普通": []types.TextTemplateItem{
				toItem("成功"),
			},
			"判定_简短_成功_困难": []types.TextTemplateItem{
				toItem("困难成功"),
			},
			"判定_简短_成功_极难": []types.TextTemplateItem{
				toItem("极难成功"),
			},
			"判定_简短_大成功": []types.TextTemplateItem{
				toItem("大成功"),
			},
			"判定_必须_困难_成功": []types.TextTemplateItem{
				toItem("困难检定成功{$t附加判定结果}"),
			},
			"判定_必须_困难_失败": []types.TextTemplateItem{
				toItem("困难检定失败{$t附加判定结果}"),
			},
			"判定_必须_极难_成功": []types.TextTemplateItem{
				toItem("极难检定成功{$t附加判定结果}"),
			},
			"判定_必须_极难_失败": []types.TextTemplateItem{
				toItem("极难检定失败{$t附加判定结果}"),
			},
			"判定_必须_大成功_成功": []types.TextTemplateItem{
				toItem("大成功检定成功{$t附加判定结果}"),
			},
			"判定_必须_大成功_失败": []types.TextTemplateItem{
				toItem("大成功检定失败{$t附加判定结果}"),
			},
			"制卡": []types.TextTemplateItem{
				toItem("{$t玩家}的COC7人物作成:\n{$t制卡结果文本}"),
			},
			"制卡_分隔符": []types.TextTemplateItem{
				toItem("\n\n"),
			},
			"理智检定": []types.TextTemplateItem{
				toItem("{$t玩家}的理智检定: {$t结果文本}\n{$t提示_角色疯狂}理智: {$t旧值} -> {$t新值}"),
			},
			"理智检定_格式错误": []types.TextTemplateItem{
				toItem("理智检定格式错误"),
			},
			"理智检定_单项结果文本": []types.TextTemplateItem{
				toItem("{$t检定表达式文本}{$t检定计算过程}={$tD100}/{$t判定值} {$t判定结果}"),
			},
			"理智检定_附加语_大失败": []types.TextTemplateItem{
				toItem("【疯狂的深渊在向你招手】"),
			},
			"理智检定_附加语_失败": []types.TextTemplateItem{
				toItem(""),
			},
			"理智检定_附加语_成功": []types.TextTemplateItem{
				toItem(""),
			},
			"理智检定_附加语_大成功": []types.TextTemplateItem{
				toItem("【意志如铁】"),
			},
			"提示_永久疯狂": []types.TextTemplateItem{
				toItem("永久疯狂：理智归零！"),
			},
			"提示_临时疯狂": []types.TextTemplateItem{
				toItem("单次损失过大，可能陷入临时疯狂"),
			},
			"技能成长": []types.TextTemplateItem{
				toItem("{$t玩家}的{$t技能}成长检定:\n{$t结果文本}"),
			},
			"技能成长_结果_成功": []types.TextTemplateItem{
				toItem("{$t旧值}+({$t表达式文本})={$t新值}"),
			},
			"技能成长_结果_成功_无后缀": []types.TextTemplateItem{
				toItem("+({$t表达式文本})={$t新值}"),
			},
			"技能成长_结果_失败": []types.TextTemplateItem{
				toItem("失败，维持{$t旧值}"),
			},
			"技能成长_结果_失败变更": []types.TextTemplateItem{
				toItem("失败，{$t旧值}+({$t表达式文本})={$t新值}"),
			},
			"技能成长_结果_失败变更_无后缀": []types.TextTemplateItem{
				toItem("+({$t表达式文本})={$t新值}"),
			},
			"技能成长_属性未录入": []types.TextTemplateItem{
				toItem("未录入该属性"),
			},
			"技能成长_错误的属性类型": []types.TextTemplateItem{
				toItem("属性类型错误"),
			},
			"技能成长_错误的成功成长值": []types.TextTemplateItem{
				toItem("成功成长值错误"),
			},
			"技能成长_错误的失败成长值": []types.TextTemplateItem{
				toItem("失败成长值错误"),
			},
			"技能成长_属性未录入_无前缀": []types.TextTemplateItem{
				toItem("未录入该属性"),
			},
			"技能成长_错误的属性类型_无前缀": []types.TextTemplateItem{
				toItem("属性类型错误"),
			},
			"技能成长_错误的成功成长值_无前缀": []types.TextTemplateItem{
				toItem("成功成长值错误"),
			},
			"技能成长_错误的失败成长值_无前缀": []types.TextTemplateItem{
				toItem("失败成长值错误"),
			},
			"技能成长_批量": []types.TextTemplateItem{
				toItem("{$t玩家}的技能成长:\n{$t总结果文本}"),
			},
			"技能成长_批量_单条": []types.TextTemplateItem{
				toItem("{$t技能}: {$t骰子出目}/{$t判定值} {$t判定结果} {$t结果文本}"),
			},
			"技能成长_批量_单条错误前缀": []types.TextTemplateItem{
				toItem("{$t技能}: "),
			},
			"技能成长_批量_分隔符": []types.TextTemplateItem{
				toItem("\n"),
			},
			"技能成长_批量_技能过多警告": []types.TextTemplateItem{
				toItem("一次最多处理10个技能"),
			},
			"疯狂发作_即时症状": []types.TextTemplateItem{
				toItem("{$t玩家}的即时疯狂症状: {$t表达式文本}\n{$t疯狂描述}"),
			},
			"疯狂发作_总结症状": []types.TextTemplateItem{
				toItem("{$t玩家}的总结疯狂症状: {$t表达式文本}\n{$t疯狂描述}"),
			},
			"对抗检定": []types.TextTemplateItem{
				toItem("{$t玩家A}({$t玩家A判定式}) vs {$t玩家B}({$t玩家B判定式}):\n{$t玩家A}: {$t玩家A出目}/{$t玩家A属性} {$t玩家A结果}\n{$t玩家B}: {$t玩家B出目}/{$t玩家B属性} {$t玩家B结果}"),
			},
			"设置房规_当前": []types.TextTemplateItem{
				toItem("当前房规为{$t房规}: {$t房规文本}"),
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

// newCoc7CmdTestContext 创建用于测试 COC7 命令的上下文
func newCoc7CmdTestContext(t *testing.T) (*types.MsgContext, *types.Message, *stubDice) {
	t.Helper()

	asset, ok := BuiltinGameSystemTemplateAsset("coc7.yaml")
	require.True(t, ok)

	tmpl, err := types.LoadGameSystemTemplateFromData(asset.Data, asset.Filename)
	require.NoError(t, err)

	stub := newStubDice(tmpl)
	RegisterBuiltinExtCoc7(stub)

	am := &attrs.AttrsManager{}
	am.SetIO(attrs.NewMemoryAttrsIO())

	group := &types.GroupInfo{
		GroupId: "group-coc7-cmd-test",
		System:  "coc7",
		BotList: &utils.SyncMap[string, bool]{},
		Active:  true,
	}

	player := &types.GroupPlayerInfo{
		UserId: "user-coc7-cmd-test",
		Name:   "调查员A",
	}

	ctx := &types.MsgContext{
		Dice:            stub,
		AttrsManager:    am,
		Group:           group,
		Player:          player,
		TextTemplateMap: coc7TextMap(),
		GameSystem:      tmpl,
	}

	// 激活扩展
	ext := stub.extensions["coc7"]
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

func executeCoc7Command(t *testing.T, stub *stubDice, ctx *types.MsgContext, msg *types.Message, raw string, names ...string) (types.CmdExecuteResult, string) {
	t.Helper()

	ext, ok := stub.extensions["coc7"]
	require.True(t, ok, "coc7 extension should be registered")

	cmdArgs := types.CommandParse(raw, names, []string{".", "。"}, "", false)
	require.NotNilf(t, cmdArgs, "failed to parse command %q", raw)

	cmd, ok := ext.CmdMap[cmdArgs.Command]
	require.Truef(t, ok, "command %q not found in coc7 extension", cmdArgs.Command)

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

// ================= SetCOC 房规测试 =================

func TestCoc7SetCocHelp(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	result, _ := executeCoc7Command(t, stub, ctx, msg, ".setcoc help", "setcoc")
	require.True(t, result.ShowHelp)
}

func TestCoc7SetCocRule0(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".setcoc 0", "setcoc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "0")
	require.Equal(t, 0, ctx.Group.CocRuleIndex)
}

func TestCoc7SetCocRule1(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".setcoc 1", "setcoc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "1")
	require.Equal(t, 1, ctx.Group.CocRuleIndex)
}

func TestCoc7SetCocRule2(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".setcoc 2", "setcoc")
	require.NotEmpty(t, reply)
	require.Equal(t, 2, ctx.Group.CocRuleIndex)
}

func TestCoc7SetCocDG(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".setcoc dg", "setcoc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "DeltaGreen")
	require.Equal(t, 11, ctx.Group.CocRuleIndex)
}

func TestCoc7SetCocDetails(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".setcoc details", "setcoc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, ".setcoc 0")
	require.Contains(t, reply, ".setcoc dg")
}

func TestCoc7SetCocShowCurrent(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 先设置一个规则
	ctx.Group.CocRuleIndex = 2

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".setcoc show", "setcoc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "2")
}

// ================= COC 制卡测试 =================

func TestCoc7CocCard(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".coc", "coc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "力量")
	require.Contains(t, reply, "敏捷")
	require.Contains(t, reply, "意志")
}

func TestCoc7CocCardMultiple(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".coc 3", "coc")
	require.NotEmpty(t, reply)
	// 三组数据应该有多组力量
	count := strings.Count(reply, "力量")
	require.Equal(t, 3, count)
}

// ================= TI/LI 疯狂症状测试 =================

func TestCoc7Ti(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ti", "ti")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "即时疯狂")
}

func TestCoc7TiHelp(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	result, _ := executeCoc7Command(t, stub, ctx, msg, ".ti help", "ti")
	require.True(t, result.ShowHelp)
}

func TestCoc7Li(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".li", "li")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "总结疯狂")
}

func TestCoc7LiHelp(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	result, _ := executeCoc7Command(t, stub, ctx, msg, ".li help", "li")
	require.True(t, result.ShowHelp)
}

// ================= EN 技能成长测试 =================

func TestCoc7EnHelp(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	result, _ := executeCoc7Command(t, stub, ctx, msg, ".en help", "en")
	require.True(t, result.ShowHelp)
}

func TestCoc7EnSkillNotSet(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 使用一个没有默认值的技能
	_, reply := executeCoc7Command(t, stub, ctx, msg, ".en 自定义技能XYZ", "en")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "未录入")
}

func TestCoc7EnSkillWithValue(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 先设置技能
	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	executeStCommands(t, stub, ctx, msg, cmdSt, ".st 侦查50")

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".en 侦查", "en")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "侦查")
}

func TestCoc7EnWithExplicitValue(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".en 侦查60", "en")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "侦查")
}

func TestCoc7EnWithCustomGrowth(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 使用固定成长值
	_, reply := executeCoc7Command(t, stub, ctx, msg, ".en 侦查50 +5", "en")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "侦查")
}

func TestCoc7EnBatchSkills(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 先设置多个技能
	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	executeStCommands(t, stub, ctx, msg, cmdSt, ".st 侦查50 图书馆60")

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".en 侦查 图书馆", "en")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "侦查")
	require.Contains(t, reply, "图书馆")
}

// ================= RA 检定测试 =================

func TestCoc7RaHelp(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	result, _ := executeCoc7Command(t, stub, ctx, msg, ".ra help", "ra", "rc", "rah", "rch")
	require.True(t, result.ShowHelp)
}

func TestCoc7RaBasic(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ra 50", "ra", "rc", "rah", "rch")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestCoc7RaWithSkillName(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 先设置技能
	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	executeStCommands(t, stub, ctx, msg, cmdSt, ".st 侦查50")

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ra 侦查", "ra", "rc", "rah", "rch")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestCoc7RaHard(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ra 困难50", "ra", "rc", "rah", "rch")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestCoc7RaExtreme(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ra 极难50", "ra", "rc", "rah", "rch")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestCoc7RaWithBonusDice(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ra b 50", "ra", "rc", "rah", "rch")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestCoc7RaWithPenaltyDice(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ra p 50", "ra", "rc", "rah", "rch")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "检定")
}

func TestCoc7RaMultiple(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 多轮检定会返回多个结果
	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ra 3#50", "ra", "rc", "rah", "rch")
	require.NotEmpty(t, reply)
	// 应该包含多次检定结果（3个换行分隔的结果）
	require.Contains(t, reply, "检定")
}

func TestCoc7RcUsesRule0(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 设置房规为2
	ctx.Group.CocRuleIndex = 2

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".rc 50", "ra", "rc", "rah", "rch")
	require.NotEmpty(t, reply)
	// .rc 强制使用规则0
	require.Contains(t, reply, "检定")
}

// ================= SC 理智检定测试 =================

func TestCoc7ScHelp(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	result, _ := executeCoc7Command(t, stub, ctx, msg, ".sc help", "sc")
	require.True(t, result.ShowHelp)
}

func TestCoc7ScBasic(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 先设置理智
	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	executeStCommands(t, stub, ctx, msg, cmdSt, ".st san60")

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".sc 1/1d6", "sc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "理智")
}

func TestCoc7ScWithExpressions(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	executeStCommands(t, stub, ctx, msg, cmdSt, ".st san60")

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".sc 1d3/1d10", "sc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "理智")
}

func TestCoc7ScSimpleForm(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	executeStCommands(t, stub, ctx, msg, cmdSt, ".st san60")

	// 简写形式 .sc 1d6，失败时扣1d6，成功时扣0
	_, reply := executeCoc7Command(t, stub, ctx, msg, ".sc 1d6", "sc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "理智")
}

func TestCoc7ScWithBonusDice(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	executeStCommands(t, stub, ctx, msg, cmdSt, ".st san60")

	_, reply := executeCoc7Command(t, stub, ctx, msg, ".sc b 1/1d6", "sc")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "理智")
}

func TestCoc7ScPersistsSanValue(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	executeStCommands(t, stub, ctx, msg, cmdSt, ".st san60")

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	require.Equal(t, int64(60), attrIntValue(t, ctx, attrsItem, "san"))

	// 进行 SC，理智应该发生变化
	_, _ = executeCoc7Command(t, stub, ctx, msg, ".sc 1/1d6", "sc")

	// 理智应该小于等于60（可能扣血也可能不扣）
	newSan := attrIntValue(t, ctx, attrsItem, "san")
	require.LessOrEqual(t, newSan, int64(60))
}

// ================= ResultCheck 规则测试 =================

func TestResultCheckRule0(t *testing.T) {
	// 规则0：规则书规则
	// 出1大成功，不满50出96-100大失败，满50出100大失败

	// 大成功：骰1
	rank, _ := ResultCheckBase(0, 1, 50, 0)
	require.Equal(t, 4, rank, "should be critical success on 1")

	// 大失败：属性<50时骰96-100
	rank, _ = ResultCheckBase(0, 96, 40, 0)
	require.Equal(t, -2, rank, "should be fumble on 96 when attr < 50")

	// 大失败：属性>=50时只有100才是大失败
	rank, _ = ResultCheckBase(0, 99, 60, 0)
	require.Equal(t, -1, rank, "should be failure (not fumble) on 99 when attr >= 50")

	rank, _ = ResultCheckBase(0, 100, 60, 0)
	require.Equal(t, -2, rank, "should be fumble on 100")

	// 普通成功
	rank, _ = ResultCheckBase(0, 30, 50, 0)
	require.Equal(t, 1, rank, "should be regular success")

	// 困难成功
	rank, _ = ResultCheckBase(0, 20, 50, 0)
	require.Equal(t, 2, rank, "should be hard success")

	// 极难成功
	rank, _ = ResultCheckBase(0, 5, 50, 0)
	require.Equal(t, 3, rank, "should be extreme success")

	// 普通失败
	rank, _ = ResultCheckBase(0, 60, 50, 0)
	require.Equal(t, -1, rank, "should be failure")
}

func TestResultCheckRule1(t *testing.T) {
	// 规则1：不满50出1大成功，满50出1-5大成功
	// 不满50出96-100大失败，满50出100大失败

	// 属性<50时只有1是大成功
	rank, _ := ResultCheckBase(1, 3, 40, 0)
	require.Equal(t, 3, rank, "should be extreme success on 3 when attr < 50")

	// 属性>=50时1-5都是大成功
	rank, _ = ResultCheckBase(1, 3, 60, 0)
	require.Equal(t, 4, rank, "should be critical success on 3 when attr >= 50")
}

func TestResultCheckRule2(t *testing.T) {
	// 规则2：出1-5且<=成功率大成功，出96-100且>成功率大失败

	// 属性60，骰5是大成功
	rank, _ := ResultCheckBase(2, 5, 60, 0)
	require.Equal(t, 4, rank, "should be critical success")

	// 属性3，骰5不是大成功（因为5>3）
	rank, _ = ResultCheckBase(2, 5, 3, 0)
	require.Equal(t, -1, rank, "should be failure when roll > attr")

	// 属性99，骰97是成功（因为97<=99）
	rank, _ = ResultCheckBase(2, 97, 99, 0)
	require.Equal(t, 1, rank, "should be success when roll <= attr")

	// 属性90，骰97是大失败（因为97>90 且 97>=96）
	rank, _ = ResultCheckBase(2, 97, 90, 0)
	require.Equal(t, -2, rank, "should be fumble when roll > attr and >= 96")
}

func TestResultCheckRule3(t *testing.T) {
	// 规则3：出1-5大成功，出96-100大失败（无视判定结果）

	rank, _ := ResultCheckBase(3, 5, 10, 0)
	require.Equal(t, 4, rank, "should be critical success on 5")

	rank, _ = ResultCheckBase(3, 96, 99, 0)
	require.Equal(t, -2, rank, "should be fumble on 96 regardless of attr")
}

func TestResultCheckRule11DG(t *testing.T) {
	// 规则11：DG规则
	// 检定成功基础上个位十位相同为大成功
	// 检定失败基础上个位十位相同为大失败
	// 出1大成功，出100大失败

	// 出1必定大成功
	rank, _ := ResultCheckBase(11, 1, 50, 0)
	require.Equal(t, 4, rank, "should be critical success on 1")

	// 出100必定大失败
	rank, _ = ResultCheckBase(11, 100, 50, 0)
	require.Equal(t, -2, rank, "should be fumble on 100")

	// 成功且双位相同(22, 33等) = 大成功
	rank, _ = ResultCheckBase(11, 22, 50, 0)
	require.Equal(t, 4, rank, "should be critical success on 22 when success")

	// 成功但双位不同 = 普通成功
	rank, _ = ResultCheckBase(11, 23, 50, 0)
	require.Equal(t, 1, rank, "should be regular success on 23")

	// 失败且双位相同(66, 77等) = 大失败
	rank, _ = ResultCheckBase(11, 77, 50, 0)
	require.Equal(t, -2, rank, "should be fumble on 77 when failure")

	// 失败但双位不同 = 普通失败
	rank, _ = ResultCheckBase(11, 78, 50, 0)
	require.Equal(t, -1, rank, "should be regular failure on 78")
}

func TestResultCheckWithDifficultyRequired(t *testing.T) {
	// 测试带难度要求的检定

	// 困难检定：需要<=属性/2
	rank, _ := ResultCheckBase(0, 25, 60, 2)
	require.GreaterOrEqual(t, rank, 2, "should pass hard check when roll <= attr/2")

	rank, _ = ResultCheckBase(0, 35, 60, 2)
	require.Less(t, rank, 2, "should fail hard check when roll > attr/2")

	// 极难检定：需要<=属性/5
	rank, _ = ResultCheckBase(0, 10, 60, 3)
	require.GreaterOrEqual(t, rank, 3, "should pass extreme check when roll <= attr/5")

	rank, _ = ResultCheckBase(0, 15, 60, 3)
	require.Less(t, rank, 3, "should fail extreme check when roll > attr/5")
}
