package exts

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

// 这些测试专注于 DND5E 指令之间的相互作用和副作用

// TestDnd5e_BuffAffectsRc 测试 buff 如何影响 rc 检定结果
func TestDnd5e_BuffAffectsRc(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 1. 设置基础属性
	executeStCommand(t, stub, ctx, msg, ".st 力量14 熟练2", cmdSt)

	// 2. 不带 buff 检定
	_, rcReply1 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 力量", "rc", "ra", "drc")
	require.Contains(t, rcReply1, "力量")
	t.Logf("无buff力量检定: %s", rcReply1)

	// 3. 添加力量 buff
	_, buffReply := executeDnd5eCommand(t, stub, ctx, msg, ".buff 力量+4", "buff")
	require.NotEmpty(t, buffReply)
	t.Logf("添加buff: %s", buffReply)

	// 4. 查看 buff 状态
	_, buffShowReply := executeDnd5eCommand(t, stub, ctx, msg, ".buff", "buff")
	t.Logf("当前buff: %s", buffShowReply)

	// 5. 带 buff 检定（应该使用更高的调整值）
	_, rcReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 力量", "rc", "ra", "drc")
	require.Contains(t, rcReply2, "力量")
	t.Logf("有buff力量检定: %s", rcReply2)

	// 6. 清除 buff 后检定
	_, clrReply := executeDnd5eCommand(t, stub, ctx, msg, ".buff clr", "buff")
	t.Logf("清除buff: %s", clrReply)

	_, rcReply3 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 力量", "rc", "ra", "drc")
	t.Logf("清除buff后力量检定: %s", rcReply3)
}

// TestDnd5e_CastConsumesSpellSlots 测试 cast 如何消耗法术位
func TestDnd5e_CastConsumesSpellSlots(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)
	_ = cmdSt // 用于设置属性

	// 1. 初始化法术位
	_, initReply := executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3 2", "ss")
	require.Contains(t, initReply, "1环4个")
	t.Logf("初始化法术位: %s", initReply)

	// 2. 查看初始法术位
	_, showReply1 := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss")
	require.Contains(t, showReply1, "1环:4/4")
	t.Logf("初始法术位: %s", showReply1)

	// 3. 施放1环法术
	_, castReply1 := executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast")
	require.Contains(t, castReply1, "消耗")
	t.Logf("施放1环: %s", castReply1)

	// 4. 验证法术位减少
	_, showReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss")
	require.Contains(t, showReply2, "1环:3/4")
	t.Logf("施放后法术位: %s", showReply2)

	// 5. 再施放2环法术
	_, castReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".cast 2", "cast")
	t.Logf("施放2环: %s", castReply2)

	// 6. 验证多个环位变化
	_, showReply3 := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss")
	require.Contains(t, showReply3, "1环:3/4")
	require.Contains(t, showReply3, "2环:2/3")
	t.Logf("再次施放后法术位: %s", showReply3)
}

// TestDnd5e_LongRestRecovery 测试长休息如何恢复 HP 和法术位
func TestDnd5e_LongRestRecovery(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 设置 HP 和最大 HP
	executeStCommand(t, stub, ctx, msg, ".st hp15 hpmax25", cmdSt)
	require.Equal(t, int64(15), attrIntValue(t, ctx, attrsItem, "hp"))

	// 2. 初始化并消耗法术位
	executeDnd5eCommand(t, stub, ctx, msg, ".ss init 4 3", "ss")
	executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast")
	executeDnd5eCommand(t, stub, ctx, msg, ".cast 2", "cast")

	// 3. 验证消耗后状态
	_, showReply1 := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss")
	require.Contains(t, showReply1, "1环:3/4")
	require.Contains(t, showReply1, "2环:2/3")
	t.Logf("休息前法术位: %s", showReply1)
	t.Logf("休息前HP: %d", attrIntValue(t, ctx, attrsItem, "hp"))

	// 4. 长休息
	_, restReply := executeDnd5eCommand(t, stub, ctx, msg, ".longrest", "longrest", "长休", "长休息")
	require.Contains(t, restReply, "恢复")
	t.Logf("长休息: %s", restReply)

	// 5. 验证 HP 恢复到满
	newHp := attrIntValue(t, ctx, attrsItem, "hp")
	require.Equal(t, int64(25), newHp, "HP应该恢复到最大值")
	t.Logf("休息后HP: %d", newHp)

	// 6. 验证法术位恢复
	_, showReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss")
	require.Contains(t, showReply2, "1环:4/4")
	require.Contains(t, showReply2, "2环:3/3")
	t.Logf("休息后法术位: %s", showReply2)
}

// TestDnd5e_DeathSavingStateChanges 测试死亡豁免状态的累积和重置
func TestDnd5e_DeathSavingStateChanges(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 设置 HP 为 0（濒死状态）
	executeStCommand(t, stub, ctx, msg, ".st hp0", cmdSt)
	require.Equal(t, int64(0), attrIntValue(t, ctx, attrsItem, "hp"))

	// 2. 进行多次死亡豁免
	for i := 0; i < 3; i++ {
		_, dsReply := executeDnd5eCommand(t, stub, ctx, msg, ".ds", "ds")
		t.Logf("死亡豁免%d: %s", i+1, dsReply)
	}

	// 3. 查看当前状态
	_, statReply := executeDnd5eCommand(t, stub, ctx, msg, ".ds stat", "ds")
	t.Logf("死亡豁免状态: %s", statReply)

	// 4. 被治疗（HP > 0 应该清除死亡豁免状态）
	executeStCommand(t, stub, ctx, msg, ".st hp5", cmdSt)
	require.Equal(t, int64(5), attrIntValue(t, ctx, attrsItem, "hp"))
	t.Logf("被治疗后HP: 5")

	// 5. 再次查看状态
	_, statReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".ds stat", "ds")
	t.Logf("治疗后死亡豁免状态: %s", statReply2)
}

// TestDnd5e_InitiativeFlow 测试先攻流程的状态变化
func TestDnd5e_InitiativeFlow(t *testing.T) {
	ctx, msg, stub, _ := newDnd5eTestContext(t)

	// 1. 添加多个角色到先攻
	_, riReply := executeDnd5eCommand(t, stub, ctx, msg,
		".ri +2 战士, +1 法师, 15 哥布林A, 12 哥布林B", "ri")
	require.Contains(t, riReply, "战士")
	require.Contains(t, riReply, "法师")
	t.Logf("添加先攻: %s", riReply)

	// 2. 查看先攻列表
	_, initReply1 := executeDnd5eCommand(t, stub, ctx, msg, ".init", "init")
	t.Logf("初始先攻列表: %s", initReply1)

	// 3. 推进回合
	for i := 0; i < 4; i++ { // 走完一轮
		_, endReply := executeDnd5eCommand(t, stub, ctx, msg, ".init end", "init")
		t.Logf("结束回合%d: %s", i+1, endReply)
	}

	// 4. 删除一个角色
	_, delReply := executeDnd5eCommand(t, stub, ctx, msg, ".init del 哥布林A", "init")
	require.Contains(t, delReply, "哥布林A")
	t.Logf("删除哥布林A: %s", delReply)

	// 5. 查看更新后的列表
	_, initReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".init", "init")
	require.NotContains(t, initReply2, "哥布林A")
	t.Logf("删除后先攻列表: %s", initReply2)

	// 6. 清空先攻
	_, clrReply := executeDnd5eCommand(t, stub, ctx, msg, ".init clr", "init")
	t.Logf("清空先攻: %s", clrReply)
}

// TestDnd5e_TempHpAbsorbsDamage 测试临时生命值如何吸收伤害
func TestDnd5e_TempHpAbsorbsDamage(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 设置基础 HP
	executeStCommand(t, stub, ctx, msg, ".st hp20 hpmax20", cmdSt)

	// 2. 添加临时 HP
	executeStCommand(t, stub, ctx, msg, ".st 临时HP=10", cmdSt)

	// 3. 查看当前状态（临时HP应该显示）
	_, showReply1 := executeStCommand(t, stub, ctx, msg, ".st show hp", cmdSt)
	t.Logf("添加临时HP后: %s", showReply1)

	// 4. 受到 5 点伤害（应该只扣临时 HP）
	_, dmgReply1 := executeStCommand(t, stub, ctx, msg, ".st hp-5", cmdSt)
	t.Logf("受到5点伤害: %s", dmgReply1)

	currentHp := attrIntValue(t, ctx, attrsItem, "hp")
	t.Logf("当前HP: %d", currentHp)

	// 5. 受到 10 点伤害（超过剩余临时 HP，应该扣到真实 HP）
	_, dmgReply2 := executeStCommand(t, stub, ctx, msg, ".st hp-10", cmdSt)
	t.Logf("受到10点伤害: %s", dmgReply2)

	finalHp := attrIntValue(t, ctx, attrsItem, "hp")
	t.Logf("最终HP: %d", finalHp)
}

// TestDnd5e_ManualAttributeAndRc 测试手动设置属性后使用 rc 检定
func TestDnd5e_ManualAttributeAndRc(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 1. 手动设置属性（模拟玩家录入制卡结果）
	executeStCommand(t, stub, ctx, msg, ".st 力量16 敏捷14 体质15 智力10 感知12 魅力8", cmdSt)
	executeStCommand(t, stub, ctx, msg, ".st 熟练2 运动*", cmdSt)

	// 2. 使用 st show 查看属性
	_, showReply := executeStCommand(t, stub, ctx, msg, ".st show", cmdSt)
	require.Contains(t, showReply, "力量")
	t.Logf("角色属性: %s", showReply)

	// 3. 使用 rc 检定力量相关技能
	_, rcReply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 运动", "rc", "ra", "drc")
	require.Contains(t, rcReply, "运动")
	t.Logf("运动检定: %s", rcReply)

	// 4. 使用 rc 检定豁免
	_, saveReply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 力量豁免", "rc", "ra", "drc")
	require.Contains(t, saveReply, "豁免")
	t.Logf("力量豁免: %s", saveReply)
}

// TestDnd5e_SpellSlotDepletion 测试法术位耗尽后的行为
func TestDnd5e_SpellSlotDepletion(t *testing.T) {
	ctx, msg, stub, _ := newDnd5eTestContext(t)

	// 1. 初始化少量法术位
	_, initReply := executeDnd5eCommand(t, stub, ctx, msg, ".ss init 2", "ss")
	t.Logf("初始化法术位: %s", initReply)

	// 2. 消耗所有法术位
	_, cast1 := executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast")
	t.Logf("施放1: %s", cast1)

	_, cast2 := executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast")
	t.Logf("施放2: %s", cast2)

	// 3. 查看耗尽状态
	_, showReply := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss")
	require.Contains(t, showReply, "1环:0/2")
	t.Logf("耗尽后法术位: %s", showReply)

	// 4. 尝试继续施放
	_, cast3 := executeDnd5eCommand(t, stub, ctx, msg, ".cast 1", "cast")
	t.Logf("尝试施放（应该失败）: %s", cast3)

	// 5. 长休息恢复
	_, restReply := executeDnd5eCommand(t, stub, ctx, msg, ".longrest", "longrest", "长休", "长休息")
	t.Logf("长休息: %s", restReply)

	// 6. 验证恢复
	_, showReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".ss", "ss")
	require.Contains(t, showReply2, "1环:2/2")
	t.Logf("恢复后法术位: %s", showReply2)
}

// TestDnd5e_BuffStackingBehavior 测试 buff 的叠加行为
func TestDnd5e_BuffStackingBehavior(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 1. 设置基础属性
	executeStCommand(t, stub, ctx, msg, ".st 力量14", cmdSt)

	// 2. 添加第一个 buff
	_, buff1 := executeDnd5eCommand(t, stub, ctx, msg, ".buff 力量+2", "buff")
	t.Logf("Buff1: %s", buff1)

	// 3. 添加第二个 buff（同属性）
	_, buff2 := executeDnd5eCommand(t, stub, ctx, msg, ".buff 力量+4", "buff")
	t.Logf("Buff2: %s", buff2)

	// 4. 查看 buff 状态
	_, buffShow := executeDnd5eCommand(t, stub, ctx, msg, ".buff", "buff")
	t.Logf("当前buff: %s", buffShow)

	// 5. 进行检定
	_, rcReply := executeDnd5eCommand(t, stub, ctx, msg, ".rc 力量", "rc", "ra", "drc")
	t.Logf("带buff检定: %s", rcReply)

	// 6. 删除特定 buff
	_, delReply := executeDnd5eCommand(t, stub, ctx, msg, ".buff del 力量", "buff")
	t.Logf("删除buff: %s", delReply)

	// 7. 再次检定
	_, rcReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 力量", "rc", "ra", "drc")
	t.Logf("删除buff后检定: %s", rcReply2)
}

// TestDnd5e_SkillProficiencyAffectsRc 测试技能熟练如何影响检定
func TestDnd5e_SkillProficiencyAffectsRc(t *testing.T) {
	ctx, msg, stub, cmdSt := newDnd5eTestContext(t)

	// 1. 设置基础属性和熟练
	executeStCommand(t, stub, ctx, msg, ".st 敏捷16 熟练3", cmdSt)

	// 2. 不熟练的技能检定
	_, rcReply1 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 隐匿", "rc", "ra", "drc")
	t.Logf("隐匿（不熟练）: %s", rcReply1)

	// 3. 添加熟练标记
	executeStCommand(t, stub, ctx, msg, ".st 隐匿*", cmdSt) // * 表示熟练

	// 4. 熟练后检定（应该加熟练加值）
	_, rcReply2 := executeDnd5eCommand(t, stub, ctx, msg, ".rc 隐匿", "rc", "ra", "drc")
	require.Contains(t, rcReply2, "隐匿")
	t.Logf("隐匿（熟练后）: %s", rcReply2)

	// 5. 查看属性确认熟练已设置
	_, showReply := executeStCommand(t, stub, ctx, msg, ".st show 隐匿", cmdSt)
	t.Logf("隐匿技能状态: %s", showReply)
}
