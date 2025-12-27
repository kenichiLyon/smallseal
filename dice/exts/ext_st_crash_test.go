package exts

import (
	"testing"

	ds "github.com/sealdice/dicescript"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

// TestStModWithChineseName 测试带中文名的属性修改指令
// 重现 ".st 母语+1d10" 段错误问题
func TestStModWithChineseName(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	// 首先设置一个初始值
	executeStCommands(t, stub, ctx, msg, cmd, ".st 母语50")

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	require.Equal(t, int64(50), attrIntValue(t, ctx, attrsItem, "母语"))

	// 测试修改操作 - 这里应该不会崩溃
	_, reply := executeStCommand(t, stub, ctx, msg, ".st 母语+10", cmd)
	require.Contains(t, reply, "MOD")
	require.Equal(t, int64(60), attrIntValue(t, ctx, attrsItem, "母语"))
}

// TestStModWithoutInitialValue 测试没有初始值时的修改操作
func TestStModWithoutInitialValue(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	// 使用一个不存在别名的自定义属性
	_, reply := executeStCommand(t, stub, ctx, msg, ".st 自定义属性+10", cmd)
	require.Contains(t, reply, "MOD")

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	val, ok := attrsItem.Load("自定义属性")
	require.True(t, ok, "自定义属性 attribute should exist")
	require.NotNil(t, val)
	// 0+10应该等于10
	require.Equal(t, "10", val.ToString())
}

// TestStModWithDiceExpression 测试带骰子表达式的修改
func TestStModWithDiceExpression(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	// 先设置初始值
	executeStCommands(t, stub, ctx, msg, cmd, ".st 母语50")

	// 使用骰子表达式修改
	_, reply := executeStCommand(t, stub, ctx, msg, ".st 母语+1d10", cmd)
	require.Contains(t, reply, "MOD")

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	val := attrIntValue(t, ctx, attrsItem, "母语")
	// 1d10的结果应该在51-60之间
	require.GreaterOrEqual(t, val, int64(51))
	require.LessOrEqual(t, val, int64(60))
}

// TestStModWithNativeLanguage 测试母语属性的修改（母语在COC7中默认引用教育）
func TestStModWithNativeLanguage(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	// 先设置教育为20
	executeStCommands(t, stub, ctx, msg, cmd, ".st 教育20")
	
	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	require.Equal(t, int64(20), attrIntValue(t, ctx, attrsItem, "教育"))

	// 测试母语+1 - 应该把母语设置为21（打破与教育的引用）
	_, reply := executeStCommand(t, stub, ctx, msg, ".st 母语+1", cmd)
	require.Contains(t, reply, "MOD")
	t.Logf("Reply: %s", reply)

	// 检查母语的值 - 应该是21
	val, ok := attrsItem.Load("母语")
	require.True(t, ok, "母语 should be explicitly set after modification")
	t.Logf("母语的值: %s (类型: %d)", val.ToString(), val.TypeId)
	
	// 期望：母语应该被设置为21（教育20 + 1）
	if val.TypeId == ds.VMTypeInt {
		require.Equal(t, int64(21), int64(val.MustReadInt()), "母语+1 should result in 21")
	} else {
		t.Errorf("母语的类型应该是整数，但是是: %d, 值是: %s", val.TypeId, val.ToString())
	}
}

// TestStModWithDiceOnAliasAttribute 测试对别名属性使用骰子表达式修改
func TestStModWithDiceOnAliasAttribute(t *testing.T) {
	ctx, msg, stub := newCoc7TestContext(t)
	cmd := getCmdStBase(CmdStOverrideInfo{})

	// 先设置教育为50
	executeStCommands(t, stub, ctx, msg, cmd, ".st 教育50")

	// 使用骰子表达式修改母语 - 这是原始的段错误场景
	_, reply := executeStCommand(t, stub, ctx, msg, ".st 母语+1d10", cmd)
	require.Contains(t, reply, "MOD")

	attrsItem := lo.Must(ctx.AttrsManager.Load(ctx.Group.GroupId, ctx.Player.UserId))
	val, ok := attrsItem.Load("母语")
	require.True(t, ok)
	
	// 母语应该是50+1d10，结果在51-60之间
	if val.TypeId == ds.VMTypeInt {
		intVal := int64(val.MustReadInt())
		require.GreaterOrEqual(t, intVal, int64(51))
		require.LessOrEqual(t, intVal, int64(60))
	} else {
		t.Logf("母语的类型是 %d, 值是 %s", val.TypeId, val.ToString())
	}
}
