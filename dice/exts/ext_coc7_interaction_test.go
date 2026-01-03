package exts

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

// 这些测试专注于 COC7 指令之间的相互作用和副作用

// TestCoc7_StAffectsRa 测试 st 设置的属性如何影响 ra 检定
func TestCoc7_StAffectsRa(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 设置技能值
	executeStCommand(t, stub, ctx, msg, ".st 侦查60 聆听45", cmdSt)

	// 2. 使用 ra 检定该技能，验证使用了 st 设置的值
	_, reply := executeCoc7Command(t, stub, ctx, msg, ".ra 侦查", "ra", "rc")
	require.Contains(t, reply, "/60") // 验证使用了正确的技能值
	t.Logf("侦查检定: %s", reply)

	// 3. 修改技能值
	executeStCommand(t, stub, ctx, msg, ".st 侦查80", cmdSt)
	require.Equal(t, int64(80), attrIntValue(t, ctx, attrsItem, "侦查"))

	// 4. 再次检定，验证使用了新值
	_, reply2 := executeCoc7Command(t, stub, ctx, msg, ".ra 侦查", "ra", "rc")
	require.Contains(t, reply2, "/80")
	t.Logf("修改后侦查检定: %s", reply2)
}

// TestCoc7_ScModifiesSan 测试 sc 如何修改理智值并影响后续检定
func TestCoc7_ScModifiesSan(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 设置初始理智值
	executeStCommand(t, stub, ctx, msg, ".st 理智65", cmdSt)
	initialSan := attrIntValue(t, ctx, attrsItem, "理智")
	require.Equal(t, int64(65), initialSan)

	// 2. 进行理智检定
	_, scReply := executeCoc7Command(t, stub, ctx, msg, ".sc 1/1d6", "sc")
	require.NotEmpty(t, scReply)
	t.Logf("SC结果: %s", scReply)

	// 3. 验证理智值被修改了（无论成功失败都会扣除）
	newSan := attrIntValue(t, ctx, attrsItem, "理智")
	require.Less(t, newSan, initialSan, "理智值应该减少")
	t.Logf("理智变化: %d -> %d", initialSan, newSan)

	// 4. 使用 ra 检定理智（验证用新的值）
	_, raReply := executeCoc7Command(t, stub, ctx, msg, ".ra 理智", "ra", "rc")
	require.Contains(t, raReply, "理智")
	t.Logf("理智检定: %s", raReply)
}

// TestCoc7_EnModifiesSkill 测试 en 如何修改技能值并影响后续检定
func TestCoc7_EnModifiesSkill(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 1. 设置初始技能值
	executeStCommand(t, stub, ctx, msg, ".st 急救30", cmdSt)

	// 2. 查看初始值
	_, showReply1 := executeStCommand(t, stub, ctx, msg, ".st show 急救", cmdSt)
	require.Contains(t, showReply1, "30")
	t.Logf("初始急救: %s", showReply1)

	// 3. 进行技能成长（多次尝试）
	for i := 0; i < 5; i++ {
		_, enReply := executeCoc7Command(t, stub, ctx, msg, ".en 急救", "en")
		t.Logf("成长尝试%d: %s", i+1, enReply)
	}

	// 4. 验证成长后使用 ra 检定
	_, raReply := executeCoc7Command(t, stub, ctx, msg, ".ra 急救", "ra", "rc")
	require.Contains(t, raReply, "急救")
	t.Logf("成长后急救检定: %s", raReply)

	// 5. 查看最终值
	_, showReply2 := executeStCommand(t, stub, ctx, msg, ".st show 急救", cmdSt)
	t.Logf("最终技能值: %s", showReply2)
}

// TestCoc7_SetcocAffectsRaResult 测试 setcoc 规则如何影响 ra 的大成功/大失败判定
func TestCoc7_SetcocAffectsRaResult(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 设置技能
	executeStCommand(t, stub, ctx, msg, ".st 测试技能50", cmdSt)

	// 1. 默认规则0下进行检定
	_, setReply0 := executeCoc7Command(t, stub, ctx, msg, ".setcoc 0", "setcoc", "coc7")
	require.Contains(t, setReply0, "房规为0") // 输出: "已切换房规为0:..."
	t.Logf("设置规则0: %s", setReply0)

	// 多次检定观察结果格式
	for i := 0; i < 3; i++ {
		_, raReply := executeCoc7Command(t, stub, ctx, msg, ".ra 测试技能", "ra", "rc")
		t.Logf("规则0检定%d: %s", i+1, raReply)
	}

	// 2. 切换到规则2
	_, setReply2 := executeCoc7Command(t, stub, ctx, msg, ".setcoc 2", "setcoc", "coc7")
	require.Contains(t, setReply2, "房规为2")
	t.Logf("设置规则2: %s", setReply2)

	// 多次检定观察结果格式（规则2在1-5大成功）
	for i := 0; i < 3; i++ {
		_, raReply := executeCoc7Command(t, stub, ctx, msg, ".ra 测试技能", "ra", "rc")
		t.Logf("规则2检定%d: %s", i+1, raReply)
	}

	// 3. 切换到DG规则
	_, setReplyDG := executeCoc7Command(t, stub, ctx, msg, ".setcoc dg", "setcoc", "coc7")
	require.Contains(t, setReplyDG, "DeltaGreen")
	t.Logf("设置DG规则: %s", setReplyDG)

	// DG规则下检定
	for i := 0; i < 3; i++ {
		_, raReply := executeCoc7Command(t, stub, ctx, msg, ".ra 测试技能", "ra", "rc")
		t.Logf("DG规则检定%d: %s", i+1, raReply)
	}
}

// TestCoc7_CocCardGeneratesValidOutput 测试 coc 制卡生成有效的属性输出
func TestCoc7_CocCardGeneratesValidOutput(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)

	// 1. 使用 coc 制卡生成属性
	_, cocReply := executeCoc7Command(t, stub, ctx, msg, ".coc", "coc")
	require.NotEmpty(t, cocReply)
	require.Contains(t, cocReply, "力量")
	require.Contains(t, cocReply, "敏捷")
	require.Contains(t, cocReply, "体质")
	require.Contains(t, cocReply, "HP")
	t.Logf("制卡结果: %s", cocReply)

	// 2. 使用 coc 5 生成多组数据
	_, coc5Reply := executeCoc7Command(t, stub, ctx, msg, ".coc 3", "coc")
	require.NotEmpty(t, coc5Reply)
	require.Contains(t, coc5Reply, "力量") // 应该包含多组属性
	t.Logf("多组制卡结果: %s", coc5Reply)
}

// TestCoc7_MultipleScReducesSanProgressively 测试多次 sc 如何逐步降低理智
func TestCoc7_MultipleScReducesSanProgressively(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 设置较高初始理智
	executeStCommand(t, stub, ctx, msg, ".st 理智80", cmdSt)

	// 2. 记录每次 sc 后的理智变化
	sanValues := []int64{80}

	for i := 0; i < 5; i++ {
		_, scReply := executeCoc7Command(t, stub, ctx, msg, ".sc 1/1d3", "sc")
		currentSan := attrIntValue(t, ctx, attrsItem, "理智")
		sanValues = append(sanValues, currentSan)
		t.Logf("SC %d: 理智=%d, %s", i+1, currentSan, scReply)
	}

	// 3. 验证理智逐步减少
	for i := 1; i < len(sanValues); i++ {
		require.LessOrEqual(t, sanValues[i], sanValues[i-1],
			"每次SC后理智应该减少或保持不变")
	}

	t.Logf("理智变化轨迹: %v", sanValues)
}

// TestCoc7_StDeleteAndRaFallback 测试删除属性后 ra 的行为
func TestCoc7_StDeleteAndRaFallback(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))

	// 1. 设置自定义技能
	executeStCommand(t, stub, ctx, msg, ".st 自定义技能XYZ=55", cmdSt)
	require.Equal(t, int64(55), attrIntValue(t, ctx, attrsItem, "自定义技能XYZ"))

	// 2. 使用该技能检定
	_, raReply1 := executeCoc7Command(t, stub, ctx, msg, ".ra 自定义技能XYZ", "ra", "rc")
	require.Contains(t, raReply1, "/55")
	t.Logf("删除前检定: %s", raReply1)

	// 3. 删除该技能
	_, delReply := executeStCommand(t, stub, ctx, msg, ".st del 自定义技能XYZ", cmdSt)
	require.NotEmpty(t, delReply)
	t.Logf("删除结果: %s", delReply)

	// 4. 再次检定，应该提示未设置或使用默认值
	_, raReply2 := executeCoc7Command(t, stub, ctx, msg, ".ra 自定义技能XYZ", "ra", "rc")
	t.Logf("删除后检定: %s", raReply2)
}

// TestCoc7_HpDamageAndRecovery 测试生命值的损伤和恢复
func TestCoc7_HpDamageAndRecovery(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 1. 设置初始 HP
	executeStCommand(t, stub, ctx, msg, ".st hp15", cmdSt)

	// 2. 查看初始HP
	_, showReply1 := executeStCommand(t, stub, ctx, msg, ".st show hp", cmdSt)
	require.Contains(t, showReply1, "15")
	t.Logf("初始HP: %s", showReply1)

	// 3. 受到伤害
	_, dmgReply := executeStCommand(t, stub, ctx, msg, ".st hp-5", cmdSt)
	require.Contains(t, dmgReply, "MOD")
	t.Logf("受伤: %s", dmgReply)

	// 4. 查看受伤后HP
	_, showReply2 := executeStCommand(t, stub, ctx, msg, ".st show hp", cmdSt)
	require.Contains(t, showReply2, "10") // 15-5=10
	t.Logf("受伤后HP: %s", showReply2)

	// 5. 恢复生命
	_, healReply := executeStCommand(t, stub, ctx, msg, ".st hp+3", cmdSt)
	require.Contains(t, healReply, "MOD")
	t.Logf("恢复: %s", healReply)

	// 6. 查看恢复后HP
	_, showReply3 := executeStCommand(t, stub, ctx, msg, ".st show hp", cmdSt)
	require.Contains(t, showReply3, "13") // 10+3=13
	t.Logf("恢复后HP: %s", showReply3)
}

// TestCoc7_BonusPenaltyDiceWithSt 测试奖惩骰与属性设置的交互
func TestCoc7_BonusPenaltyDiceWithSt(t *testing.T) {
	ctx, msg, stub := newCoc7CmdTestContext(t)
	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	// 1. 设置技能
	executeStCommand(t, stub, ctx, msg, ".st 侦查50 潜行40", cmdSt)

	// 2. 普通检定
	_, normalReply := executeCoc7Command(t, stub, ctx, msg, ".ra 侦查", "ra", "rc")
	require.Contains(t, normalReply, "/50")
	t.Logf("普通侦查: %s", normalReply)

	// 3. 带奖励骰检定
	_, bonusReply := executeCoc7Command(t, stub, ctx, msg, ".rab 侦查", "ra", "rb", "rab", "rc")
	require.Contains(t, bonusReply, "奖励")
	t.Logf("奖励骰侦查: %s", bonusReply)

	// 4. 带惩罚骰检定
	_, penaltyReply := executeCoc7Command(t, stub, ctx, msg, ".rap 潜行", "ra", "rp", "rap", "rc")
	require.Contains(t, penaltyReply, "惩罚")
	t.Logf("惩罚骰潜行: %s", penaltyReply)

	// 5. 修改技能值后再次检定
	executeStCommand(t, stub, ctx, msg, ".st 侦查75", cmdSt)
	_, newBonusReply := executeCoc7Command(t, stub, ctx, msg, ".rab 侦查", "ra", "rb", "rab", "rc")
	require.Contains(t, newBonusReply, "/75")
	t.Logf("修改后奖励骰侦查: %s", newBonusReply)
}
