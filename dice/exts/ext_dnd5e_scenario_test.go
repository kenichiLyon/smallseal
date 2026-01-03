package exts

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/sealdice/smallseal/dice/types"
)

// ================= DND5E 真实游戏场景测试 =================

// TestScenario_Dnd5e_CharacterCreation 测试完整的角色创建流程
// 场景：玩家创建一个人类战士角色
func TestScenario_Dnd5e_CharacterCreation(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 1. 使用制卡指令获取属性点
	ext := stub.extensions["dnd5e"]
	cmdDnd := ext.CmdMap["dnd"]

	cmdArgs := types.CommandParse(".dndx", []string{"dnd", "dndx"}, []string{".", "。"}, "", false)
	result := cmdDnd.Solve(ctx, msg, cmdArgs)
	require.True(t, result.Matched)
	t.Logf("制卡完成")

	// 2. 录入基础属性（人类战士，假设属性）
	executeStCommand(t, stub, ctx, msg, ".st 力量:16 敏捷:14 体质:15 智力:10 感知:12 魅力:8", cmdSt)

	// 3. 录入衍生属性
	executeStCommand(t, stub, ctx, msg, ".st hp:12 hpmax:12 熟练:2 ac:16", cmdSt)

	// 4. 录入战士技能（运动和威吓熟练）
	executeStCommand(t, stub, ctx, msg, ".st 运动*:0 威吓*:0", cmdSt)

	// 5. 验证角色数据
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	require.Equal(t, int64(16), attrIntValue(t, ctx, attrsItem, "力量"))
	require.Equal(t, int64(12), attrIntValue(t, ctx, attrsItem, "hp"))

	// 6. 查看属性
	_, showReply := executeStCommand(t, stub, ctx, msg, ".st show", cmdSt)
	t.Logf("角色属性: %s", showReply)
}

// TestScenario_Dnd5e_SkillCheck 测试技能检定场景
// 场景：冒险者探索地下城
func TestScenario_Dnd5e_SkillCheck(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 准备：设置角色属性
	executeStCommand(t, stub, ctx, msg, ".st 力量:14 敏捷:16 智力:12 感知:14 魅力:10 熟练:2", cmdSt)
	executeStCommand(t, stub, ctx, msg, ".st 察觉*:0 隐匿*:0 调查:0", cmdSt)

	// 1. 进入洞穴，用察觉检查周围环境
	_, reply1 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 察觉", "rc", "ra", "drc")
	require.Contains(t, reply1, "检定")
	t.Logf("察觉检定: %s", reply1)

	// 2. 想要潜行前进
	_, reply2 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 隐匿", "rc", "ra", "drc")
	require.Contains(t, reply2, "检定")
	t.Logf("隐匿检定: %s", reply2)

	// 3. 发现一扇门，用调查检查机关
	_, reply3 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 调查", "rc", "ra", "drc")
	require.Contains(t, reply3, "检定")
	t.Logf("调查检定: %s", reply3)

	// 4. 有优势的情况（队友帮助）
	_, reply4 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 优势 调查", "rc", "ra", "drc")
	require.Contains(t, reply4, "检定")
	t.Logf("优势调查检定: %s", reply4)

	// 5. 有劣势的情况（黑暗中）
	_, reply5 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 劣势 察觉", "rc", "ra", "drc")
	require.Contains(t, reply5, "检定")
	t.Logf("劣势察觉检定: %s", reply5)
}

// TestScenario_Dnd5e_Combat 测试战斗场景
// 场景：与哥布林战斗的完整回合
func TestScenario_Dnd5e_Combat(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 准备：设置战士角色
	executeStCommand(t, stub, ctx, msg, ".st 力量:16 敏捷:14 体质:15 熟练:2", cmdSt)
	executeStCommand(t, stub, ctx, msg, ".st hp:25 hpmax:25", cmdSt)

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 战斗开始，设置先攻
	_, riReply := executeDnd5eCommand(t, stub, ctx, msg, ".ri +2 战士, 12 哥布林A, 10 哥布林B", "ri")
	require.Contains(t, riReply, "战士")
	require.Contains(t, riReply, "哥布林")
	t.Logf("先攻设置: %s", riReply)

	// 2. 查看先攻列表
	_, initReply := executeDnd5eCommand(t, stub, ctx, msg, ".init", "init")
	require.Contains(t, initReply, "战士") // 验证列表中包含设置的角色
	t.Logf("先攻列表: %s", initReply)

	// 3. 战士的回合 - 攻击检定（力量+熟练）
	_, atkReply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 5", "rc", "ra", "drc") // 力量调整+3 + 熟练+2 = +5
	require.Contains(t, atkReply, "检定")
	t.Logf("攻击检定: %s", atkReply)

	// 4. 结束回合
	_, endReply := executeDnd5eCommand(t, stub, ctx, msg, ".init end", "init")
	require.NotEmpty(t, endReply) // 命令执行成功返回非空响应
	t.Logf("结束回合: %s", endReply)

	// 5. 受到伤害
	_, dmgReply := executeStCommand(t, stub, ctx, msg, ".st hp-7", cmdSt)
	require.Contains(t, dmgReply, "MOD")
	require.Equal(t, int64(18), attrIntValue(t, ctx, attrsItem, "hp"))
	t.Logf("受到伤害: %s", dmgReply)

	// 6. 战斗结束，清除先攻
	_, clrReply := executeDnd5eCommand(t, stub, ctx, msg, ".init clr", "init")
	require.NotEmpty(t, clrReply) // 命令执行成功返回非空响应
	t.Logf("清空先攻: %s", clrReply)
}

// TestScenario_Dnd5e_Spellcasting 测试施法场景
// 场景：法师在战斗中使用法术
func TestScenario_Dnd5e_Spellcasting(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 准备：设置法师角色
	executeStCommand(t, stub, ctx, msg, ".st 智力:18 感知:12 魅力:10 熟练:2", cmdSt)
	executeStCommand(t, stub, ctx, msg, ".st hp:18 hpmax:18", cmdSt)

	// 1. 初始化法术位（5级法师）
	_, ssInitReply := executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots")
	require.Contains(t, ssInitReply, "设置法术位")
	t.Logf("法术位初始化: %s", ssInitReply)

	// 2. 查看法术位
	_, ssShowReply := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss", "spellslots")
	require.Contains(t, ssShowReply, "法术位")
	t.Logf("法术位状况: %s", ssShowReply)

	// 3. 施放1环法术（魔法飞弹）
	_, cast1Reply := executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast", "dcast")
	require.Contains(t, cast1Reply, "消耗")
	t.Logf("施放1环: %s", cast1Reply)

	// 4. 施放2环法术（灼热射线）
	_, cast2Reply := executeDnd5eCommand(t, stub, ctx, msg, ".cast 2", "cast", "dcast")
	require.Contains(t, cast2Reply, "消耗")
	t.Logf("施放2环: %s", cast2Reply)

	// 5. 施放3环法术（火球术）
	_, cast3Reply := executeDnd5eCommand(t, stub, ctx, msg, ".cast 3", "cast", "dcast")
	require.Contains(t, cast3Reply, "消耗")
	t.Logf("施放3环: %s", cast3Reply)

	// 6. 再次查看法术位
	_, ssAfterReply := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss", "spellslots")
	t.Logf("施法后法术位: %s", ssAfterReply)
}

// TestScenario_Dnd5e_DeathSaving 测试死亡豁免场景
// 场景：角色HP归零后的死亡豁免
func TestScenario_Dnd5e_DeathSaving(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 准备：设置角色
	executeStCommand(t, stub, ctx, msg, ".st hp:1 hpmax:20", cmdSt)
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 受到致命伤害
	_, dmgReply := executeStCommand(t, stub, ctx, msg, ".st hp-10", cmdSt)
	t.Logf("受到致命伤害: %s", dmgReply)

	hp := attrIntValue(t, ctx, attrsItem, "hp")
	require.Equal(t, int64(0), hp)

	// 2. 进行死亡豁免
	_, ds1Reply := executeDnd5eCommand(t, stub, ctx, msg, ".ds", "ds", "死亡豁免")
	require.Contains(t, ds1Reply, "死亡豁免")
	t.Logf("第一次死亡豁免: %s", ds1Reply)

	// 3. 查看死亡豁免状态
	_, statReply := executeDnd5eCommand(t, stub, ctx, msg, ".ds stat", "ds", "死亡豁免")
	require.Contains(t, statReply, "情况")
	t.Logf("死亡豁免状态: %s", statReply)

	// 4. 队友施法治疗，恢复HP
	_, healReply := executeStCommand(t, stub, ctx, msg, ".st hp+5", cmdSt)
	t.Logf("治疗: %s", healReply)

	newHp := attrIntValue(t, ctx, attrsItem, "hp")
	require.Equal(t, int64(5), newHp)
}

// TestScenario_Dnd5e_LongRest 测试长休场景
// 场景：冒险结束后进行长休
func TestScenario_Dnd5e_LongRest(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 准备：设置受伤且消耗了法术位的角色
	executeStCommand(t, stub, ctx, msg, ".st hp:8 hpmax:25", cmdSt)

	// 初始化并消耗法术位
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss", "spellslots")
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast", "dcast")
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast", "dcast")
	_, _ = executeDnd5eCommand(t, stub, ctx, msg, ".cast 2", "cast", "dcast")

	// 查看休息前状态
	_, beforeSsReply := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss", "spellslots")
	t.Logf("休息前法术位: %s", beforeSsReply)

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	beforeHp := attrIntValue(t, ctx, attrsItem, "hp")
	t.Logf("休息前HP: %d", beforeHp)

	// 进行长休
	_, restReply := executeDnd5eCommand(t, stub, ctx, msg, ".长休", "长休", "longrest")
	require.Contains(t, restReply, "恢复")
	t.Logf("长休结果: %s", restReply)

	// 验证HP恢复
	afterHp := attrIntValue(t, ctx, attrsItem, "hp")
	require.Equal(t, int64(25), afterHp)
	t.Logf("休息后HP: %d", afterHp)

	// 查看休息后法术位
	_, afterSsReply := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss", "spellslots")
	t.Logf("休息后法术位: %s", afterSsReply)
}

// TestScenario_Dnd5e_TempHP 测试临时生命值场景
// 场景：法师施放虚假生命后受到攻击
func TestScenario_Dnd5e_TempHP(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 准备：设置角色
	executeStCommand(t, stub, ctx, msg, ".st hp:15 hpmax:15", cmdSt)
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 施放虚假生命，获得8点临时HP
	_, buffReply := executeDnd5eCommand(t, stub, ctx, msg, ".buff hp:8", "buff", "dbuff")
	t.Logf("获得临时HP: %s", buffReply)

	// 2. 查看当前状态
	_, showReply := executeStCommand(t, stub, ctx, msg, ".st show hp", cmdSt)
	t.Logf("当前HP状态: %s", showReply)

	// 3. 受到5点伤害（先扣临时HP）
	_, dmg1Reply := executeStCommand(t, stub, ctx, msg, ".st hp-5", cmdSt)
	t.Logf("受到5点伤害: %s", dmg1Reply)

	// 实际HP应该还是15
	hp1 := attrIntValue(t, ctx, attrsItem, "hp")
	require.Equal(t, int64(15), hp1)

	// 4. 再受到10点伤害（超过临时HP）
	_, dmg2Reply := executeStCommand(t, stub, ctx, msg, ".st hp-10", cmdSt)
	t.Logf("受到10点伤害: %s", dmg2Reply)

	// 临时HP用完，开始扣实际HP
	hp2 := attrIntValue(t, ctx, attrsItem, "hp")
	t.Logf("最终HP: %d", hp2)
}

// TestScenario_Dnd5e_SavingThrow 测试豁免检定场景
// 场景：面对法术效果进行豁免
func TestScenario_Dnd5e_SavingThrow(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 准备：设置角色，力量和体质豁免熟练
	executeStCommand(t, stub, ctx, msg, ".st 力量*:16 敏捷:14 体质*:15 智力:10 感知:12 魅力:8 熟练:2", cmdSt)

	// 1. 敏捷豁免（躲避火球术）- 无熟练
	_, dexSaveReply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 敏捷豁免", "rc", "ra", "drc")
	require.Contains(t, dexSaveReply, "检定")
	t.Logf("敏捷豁免: %s", dexSaveReply)

	// 2. 体质豁免（抵抗毒素）- 有熟练
	_, conSaveReply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 体质豁免", "rc", "ra", "drc")
	require.Contains(t, conSaveReply, "检定")
	t.Logf("体质豁免: %s", conSaveReply)

	// 3. 感知豁免（抵抗魅惑）- 无熟练
	_, wisSaveReply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 感知豁免", "rc", "ra", "drc")
	require.Contains(t, wisSaveReply, "检定")
	t.Logf("感知豁免: %s", wisSaveReply)
}

// TestScenario_Dnd5e_Buff 测试增益效果场景
// 场景：法师为战士施加各种增益
func TestScenario_Dnd5e_Buff(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 准备：设置战士角色
	executeStCommand(t, stub, ctx, msg, ".st 力量:16 敏捷:14 熟练:2", cmdSt)
	executeStCommand(t, stub, ctx, msg, ".st 运动*:0", cmdSt)

	// 1. 施加祝福术（攻击和豁免+1d4）- 这里简化为力量+2
	_, blessReply := executeDnd5eCommand(t, stub, ctx, msg, ".buff 力量:2", "buff", "dbuff")
	t.Logf("施加祝福术: %s", blessReply)

	// 2. 查看当前buff
	_, buffShowReply := executeDnd5eCommand(t, stub, ctx, msg, ".buff show", "buff", "dbuff")
	t.Logf("当前buff: %s", buffShowReply)

	// 3. 进行运动检定（应该受到力量buff影响）
	_, atkReply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 运动", "rc", "ra", "drc")
	t.Logf("带buff的运动检定: %s", atkReply)

	// 4. buff结束后清除
	_, clrReply := executeDnd5eCommand(t, stub, ctx, msg, ".buff clr", "buff", "dbuff")
	require.Contains(t, clrReply, "CLEAR")
	t.Logf("清除buff: %s", clrReply)
}

// TestScenario_Dnd5e_MultipleCharacters 测试多角色先攻场景
// 场景：多个玩家和怪物的战斗
func TestScenario_Dnd5e_MultipleCharacters(t *testing.T) {
	ctx, msg, stub, _ := newDnd5eTestContext(t)

	// 1. 多个角色设置先攻
	_, riReply := executeDnd5eCommand(t, stub, ctx, msg,
		".ri +3 战士, +2 盗贼, +1 法师, 15 地精头领, 12 地精A, 10 地精B, 8 地精C",
		"ri")
	require.Contains(t, riReply, "战士")
	require.Contains(t, riReply, "盗贼")
	require.Contains(t, riReply, "法师")
	require.Contains(t, riReply, "地精头领")
	t.Logf("先攻设置: %s", riReply)

	// 2. 查看完整先攻列表
	_, initReply := executeDnd5eCommand(t, stub, ctx, msg, ".init", "init")
	t.Logf("先攻列表: %s", initReply)

	// 3. 模拟几个回合
	for i := 0; i < 3; i++ {
		_, endReply := executeDnd5eCommand(t, stub, ctx, msg, ".init end", "init")
		t.Logf("回合%d结束: %s", i+1, endReply)
	}

	// 4. 地精A被击杀，从先攻中移除
	_, delReply := executeDnd5eCommand(t, stub, ctx, msg, ".init del 地精A", "init")
	require.Contains(t, delReply, "地精A")
	t.Logf("移除地精A: %s", delReply)

	// 5. 战斗结束，清空先攻
	_, clrReply := executeDnd5eCommand(t, stub, ctx, msg, ".init clr", "init")
	require.NotEmpty(t, clrReply) // 命令执行成功返回非空响应
	t.Logf("清空先攻: %s", clrReply)
}
