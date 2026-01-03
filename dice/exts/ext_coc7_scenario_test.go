package exts

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

// ================= COC7 真实游戏场景测试 =================

// TestScenario_Coc7_CharacterCreation 测试完整的角色创建流程
// 场景：玩家加入新团，创建角色并录入属性
func TestScenario_Coc7_CharacterCreation(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 1. 玩家先用制卡指令获取属性点
	_, reply := executeCoc7Command(t, stub, ctx, msg, ".coc", "coc")
	require.Contains(t, reply, "力量")
	require.Contains(t, reply, "敏捷")
	t.Logf("制卡结果: %s", reply)

	// 2. 玩家录入基础属性（假设制卡得到以下数值）
	executeStCommand(t, stub, ctx, msg, ".st 力量50 敏捷65 意志60 体质55 外貌60 教育70 体型60 智力65", cmdSt)

	// 3. 录入衍生属性
	executeStCommand(t, stub, ctx, msg, ".st hp11 san60 mp12 幸运55", cmdSt)

	// 4. 录入职业技能（假设是私家侦探）
	executeStCommand(t, stub, ctx, msg, ".st 侦查60 图书馆50 心理学45 说服40 法律30 乔装35 锁匠40 手枪35", cmdSt)

	// 5. 验证角色数据完整性
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	require.Equal(t, int64(50), attrIntValue(t, ctx, attrsItem, "力量"))
	require.Equal(t, int64(65), attrIntValue(t, ctx, attrsItem, "敏捷"))
	require.Equal(t, int64(60), attrIntValue(t, ctx, attrsItem, "san"))
	require.Equal(t, int64(60), attrIntValue(t, ctx, attrsItem, "侦查"))

	// 6. 玩家查看自己的属性
	_, showReply := executeStCommand(t, stub, ctx, msg, ".st show", cmdSt)
	require.Contains(t, showReply, "侦查")
	t.Logf("角色属性: %s", showReply)
}

// TestScenario_Coc7_Investigation 测试调查场景
// 场景：调查员在废弃图书馆搜索线索
func TestScenario_Coc7_Investigation(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备：设置调查员属性
	executeStCommand(t, stub, ctx, msg, ".st 侦查60 图书馆50 聆听45 san60 hp11", cmdSt)

	// 1. 进入废弃图书馆，先用聆听判断是否有危险
	_, reply1 := executeCoc7Command(t, stub, ctx, msg, ".ra 聆听", "ra", "rc")
	require.Contains(t, reply1, "聆听")
	t.Logf("聆听检定: %s", reply1)

	// 2. 搜索书架，使用图书馆技能
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".ra 图书馆", "ra", "rc")
	require.Contains(t, reply2, "图书馆")
	t.Logf("图书馆检定: %s", reply2)

	// 3. 发现一本奇怪的书，用侦查检查细节
	_, reply3 := executeCoc7Command(t, stub, ctx, msg, ".ra 侦查", "ra", "rc")
	require.Contains(t, reply3, "侦查")
	t.Logf("侦查检定: %s", reply3)

	// 4. 书中有神秘符文，需要困难侦查
	_, reply4 := executeCoc7Command(t, stub, ctx, msg, ".ra 困难侦查", "ra", "rc")
	require.Contains(t, reply4, "检定")
	t.Logf("困难侦查检定: %s", reply4)
}

// TestScenario_Coc7_SanityCheck 测试理智检定场景
// 场景：调查员遭遇神话生物，进行理智检定
func TestScenario_Coc7_SanityCheck(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备：设置调查员属性
	executeStCommand(t, stub, ctx, msg, ".st san65 hp11", cmdSt)
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	initialSan := attrIntValue(t, ctx, attrsItem, "san")
	require.Equal(t, int64(65), initialSan)

	// 1. 看到一具奇怪的尸体 (轻微冲击)
	_, reply1 := executeCoc7Command(t, stub, ctx, msg, ".sc 0/1", "sc")
	require.Contains(t, reply1, "理智")
	t.Logf("轻微冲击SC: %s", reply1)

	san1 := attrIntValue(t, ctx, attrsItem, "san")
	require.LessOrEqual(t, san1, initialSan)

	// 2. 发现尸体复活了 (中等冲击)
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".sc 1/1d4", "sc")
	require.Contains(t, reply2, "理智")
	t.Logf("中等冲击SC: %s", reply2)

	san2 := attrIntValue(t, ctx, attrsItem, "san")
	require.LessOrEqual(t, san2, san1)

	// 3. 目睹深潜者 (严重冲击)
	_, reply3 := executeCoc7Command(t, stub, ctx, msg, ".sc 1d6/1d20", "sc")
	require.Contains(t, reply3, "理智")
	t.Logf("严重冲击SC: %s", reply3)

	san3 := attrIntValue(t, ctx, attrsItem, "san")
	require.LessOrEqual(t, san3, san2)
	t.Logf("理智变化: %d -> %d -> %d -> %d", initialSan, san1, san2, san3)
}

// TestScenario_Coc7_Combat 测试战斗场景
// 场景：调查员与邪教徒战斗
func TestScenario_Coc7_Combat(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备：设置调查员战斗相关属性
	executeStCommand(t, stub, ctx, msg, ".st 格斗45 闪避40 手枪35 hp11 san60", cmdSt)
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 回合1：尝试用拳头攻击
	_, reply1 := executeCoc7Command(t, stub, ctx, msg, ".ra 格斗", "ra", "rc")
	require.Contains(t, reply1, "格斗")
	t.Logf("格斗攻击: %s", reply1)

	// 回合2：敌人反击，需要闪避
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".ra 闪避", "ra", "rc")
	require.Contains(t, reply2, "闪避")
	t.Logf("闪避: %s", reply2)

	// 回合3：掏出手枪射击
	_, reply3 := executeCoc7Command(t, stub, ctx, msg, ".ra 手枪", "ra", "rc")
	require.Contains(t, reply3, "手枪")
	t.Logf("手枪射击: %s", reply3)

	// 受到伤害，扣除HP
	_, modReply := executeStCommand(t, stub, ctx, msg, ".st hp-3", cmdSt)
	require.Contains(t, modReply, "MOD")

	newHp := attrIntValue(t, ctx, attrsItem, "hp")
	require.Equal(t, int64(8), newHp)
	t.Logf("受伤后HP: %d", newHp)
}

// TestScenario_Coc7_BonusPenaltyDice 测试奖励骰/惩罚骰场景
// 场景：各种情况下使用奖励骰和惩罚骰
func TestScenario_Coc7_BonusPenaltyDice(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备
	executeStCommand(t, stub, ctx, msg, ".st 侦查60 潜行50", cmdSt)

	// 1. 有充足时间仔细搜索 - 使用奖励骰
	_, reply1 := executeCoc7Command(t, stub, ctx, msg, ".ra b 侦查", "ra", "rc")
	require.Contains(t, reply1, "检定")
	t.Logf("奖励骰侦查: %s", reply1)

	// 2. 双重奖励骰（非常有利的情况）
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".ra b2 侦查", "ra", "rc")
	require.Contains(t, reply2, "检定")
	t.Logf("双奖励骰侦查: %s", reply2)

	// 3. 在嘈杂环境中潜行 - 使用惩罚骰
	_, reply3 := executeCoc7Command(t, stub, ctx, msg, ".ra p 潜行", "ra", "rc")
	require.Contains(t, reply3, "检定")
	t.Logf("惩罚骰潜行: %s", reply3)

	// 4. 双重惩罚骰（非常不利的情况）
	_, reply4 := executeCoc7Command(t, stub, ctx, msg, ".ra p2 潜行", "ra", "rc")
	require.Contains(t, reply4, "检定")
	t.Logf("双惩罚骰潜行: %s", reply4)
}

// TestScenario_Coc7_SkillGrowth 测试幕间成长场景
// 场景：调查结束后，对使用过的技能进行成长检定
func TestScenario_Coc7_SkillGrowth(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备：设置调查员技能
	executeStCommand(t, stub, ctx, msg, ".st 侦查45 图书馆40 急救35", cmdSt)
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	initialInvestigate := attrIntValue(t, ctx, attrsItem, "侦查")
	t.Logf("初始侦查: %d", initialInvestigate)

	// 1. 对侦查进行成长检定
	_, reply1 := executeCoc7Command(t, stub, ctx, msg, ".en 侦查", "en")
	require.Contains(t, reply1, "侦查")
	t.Logf("侦查成长: %s", reply1)

	// 2. 批量成长检定（图书馆和急救都用过）
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".en 图书馆 急救", "en")
	require.Contains(t, reply2, "图书馆")
	require.Contains(t, reply2, "急救")
	t.Logf("批量成长: %s", reply2)

	// 3. 带自定义成长值的成长（某些规则变体）
	_, reply3 := executeCoc7Command(t, stub, ctx, msg, ".en 侦查50 +1d6", "en")
	require.Contains(t, reply3, "侦查")
	t.Logf("自定义成长: %s", reply3)
}

// TestScenario_Coc7_HouseRules 测试房规切换场景
// 场景：KP根据团规切换不同的大成功/大失败判定规则
func TestScenario_Coc7_HouseRules(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备
	executeStCommand(t, stub, ctx, msg, ".st 侦查50", cmdSt)

	// 1. 默认规则0（规则书）
	_, reply0 := executeCoc7Command(t, stub, ctx, msg, ".setcoc 0", "setcoc")
	require.Contains(t, reply0, "0")
	t.Logf("设置规则0: %s", reply0)

	_, check0 := executeCoc7Command(t, stub, ctx, msg, ".ra 侦查", "ra", "rc")
	t.Logf("规则0检定: %s", check0)

	// 2. 切换到规则2（国内常用）
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".setcoc 2", "setcoc")
	require.Contains(t, reply2, "2")
	require.Equal(t, 2, ctx.Group.CocRuleIndex)
	t.Logf("设置规则2: %s", reply2)

	_, check2 := executeCoc7Command(t, stub, ctx, msg, ".ra 侦查", "ra", "rc")
	t.Logf("规则2检定: %s", check2)

	// 3. 切换到DG规则
	_, replyDG := executeCoc7Command(t, stub, ctx, msg, ".setcoc dg", "setcoc")
	require.Contains(t, replyDG, "DeltaGreen")
	require.Equal(t, 11, ctx.Group.CocRuleIndex)
	t.Logf("设置DG规则: %s", replyDG)

	// 4. 使用 .rc 强制规则书判定
	ctx.Group.CocRuleIndex = 2 // 设置为规则2
	_, checkRc := executeCoc7Command(t, stub, ctx, msg, ".rc 侦查", "ra", "rc")
	t.Logf("强制规则书检定(.rc): %s", checkRc)
}

// TestScenario_Coc7_TemporaryMadness 测试临时疯狂场景
// 场景：调查员理智崩溃，抽取疯狂症状
func TestScenario_Coc7_TemporaryMadness(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备：设置一个即将疯狂的调查员
	executeStCommand(t, stub, ctx, msg, ".st san15 hp11", cmdSt)

	// 1. 遭受重大冲击
	_, scReply := executeCoc7Command(t, stub, ctx, msg, ".sc 1d6/1d10", "sc")
	t.Logf("理智检定: %s", scReply)

	// 2. 抽取临时疯狂症状
	_, tiReply := executeCoc7Command(t, stub, ctx, msg, ".ti", "ti")
	require.Contains(t, tiReply, "即时疯狂")
	t.Logf("临时疯狂: %s", tiReply)

	// 3. 抽取总结疯狂症状
	_, liReply := executeCoc7Command(t, stub, ctx, msg, ".li", "li")
	require.Contains(t, liReply, "总结疯狂")
	t.Logf("总结疯狂: %s", liReply)
}

// TestScenario_Coc7_CustomAttribute 测试自定义属性场景
// 场景：使用母语、信用评级等特殊属性
func TestScenario_Coc7_CustomAttribute(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备：设置包含特殊属性的角色
	executeStCommand(t, stub, ctx, msg, ".st 母语70 信用50 克苏鲁神话5", cmdSt)

	// 1. 母语检定（阅读古老文献）
	_, reply1 := executeCoc7Command(t, stub, ctx, msg, ".ra 母语", "ra", "rc")
	require.Contains(t, reply1, "母语")
	t.Logf("母语检定: %s", reply1)

	// 2. 信用检定（购买昂贵物品）
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".ra 信用", "ra", "rc")
	require.Contains(t, reply2, "信用")
	t.Logf("信用检定: %s", reply2)

	// 3. 克苏鲁神话检定（辨认神话符文）
	_, reply3 := executeCoc7Command(t, stub, ctx, msg, ".ra 克苏鲁神话", "ra", "rc")
	require.Contains(t, reply3, "克苏鲁神话")
	t.Logf("克苏鲁神话检定: %s", reply3)

	// 4. 增加克苏鲁神话（阅读禁忌典籍后）
	_, modReply := executeStCommand(t, stub, ctx, msg, ".st 克苏鲁神话+2", cmdSt)
	require.Contains(t, modReply, "MOD")
	t.Logf("增加神话: %s", modReply)
}

// TestScenario_Coc7_MultipleRolls 测试连续多次检定场景
// 场景：紧急情况下需要快速进行多次检定
func TestScenario_Coc7_MultipleRolls(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 准备
	executeStCommand(t, stub, ctx, msg, ".st 闪避40 格斗45", cmdSt)

	// 1. 被多个敌人攻击，需要多次闪避
	_, reply1 := executeCoc7Command(t, stub, ctx, msg, ".ra 3#闪避", "ra", "rc")
	require.NotEmpty(t, reply1)
	require.Contains(t, reply1, "D100") // 验证进行了骰点
	t.Logf("3次闪避: %s", reply1)

	// 2. 反击多个敌人
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".ra 2#格斗", "ra", "rc")
	require.NotEmpty(t, reply2)
	require.Contains(t, reply2, "D100") // 验证进行了骰点
	t.Logf("2次格斗: %s", reply2)
}
