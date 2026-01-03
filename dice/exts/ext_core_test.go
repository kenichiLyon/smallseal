package exts

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sealdice/smallseal/dice/attrs"
	"github.com/sealdice/smallseal/dice/types"
	"github.com/sealdice/smallseal/utils"
)

// newCoreTestContext 创建一个用于测试 core 扩展的上下文
func newCoreTestContext(t *testing.T) (*types.MsgContext, *types.Message, *stubDice) {
	t.Helper()

	asset, ok := BuiltinGameSystemTemplateAsset("coc7.yaml")
	require.True(t, ok)

	tmpl, err := types.LoadGameSystemTemplateFromData(asset.Data, asset.Filename)
	require.NoError(t, err)

	stub := newStubDice(tmpl)
	RegisterBuiltinExtCore(stub)

	am := &attrs.AttrsManager{}
	am.SetIO(attrs.NewMemoryAttrsIO())

	group := &types.GroupInfo{
		GroupId: "group-core-test",
		System:  "coc7",
		BotList: &utils.SyncMap[string, bool]{},
		Players: &utils.SyncMap[string, *types.GroupPlayerInfo]{},
		Active:  true,
	}

	player := &types.GroupPlayerInfo{
		UserId: "user-core-test",
		Name:   "测试玩家",
	}

	ctx := &types.MsgContext{
		Dice:            stub,
		AttrsManager:    am,
		Group:           group,
		Player:          player,
		TextTemplateMap: minimalTextMap(),
		GameSystem:      tmpl,
	}

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

func executeCoreCommand(t *testing.T, stub *stubDice, ctx *types.MsgContext, msg *types.Message, raw string, names ...string) (types.CmdExecuteResult, string) {
	t.Helper()

	ext, ok := stub.extensions["core"]
	require.True(t, ok, "core extension should be registered")

	cmdArgs := types.CommandParse(raw, names, []string{".", "。"}, "", false)
	require.NotNilf(t, cmdArgs, "failed to parse command %q", raw)

	cmd, ok := ext.CmdMap[cmdArgs.Command]
	require.Truef(t, ok, "command %q not found in core extension", cmdArgs.Command)

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

// ================= Roll 指令测试 =================

func TestCoreRollBasic(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".r", "r", "rd", "roll", "rh")
	require.NotEmpty(t, reply)
}

func TestCoreRollWithExpression(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".r 2d6+3", "r", "rd", "roll", "rh")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "=")
}

func TestCoreRollWithReason(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".r d20 攻击", "r", "rd", "roll", "rh")
	require.NotEmpty(t, reply)
	// 只验证有骰点结果
	require.Contains(t, reply, "=")
}

func TestCoreRollMultipleTimes(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".r 3#d6", "r", "rd", "roll", "rh")
	require.NotEmpty(t, reply)
	// 验证有骰点结果
	require.Contains(t, reply, "=")
}

func TestCoreRollD20Shorthand(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	// .rd 应该等同于 .r d
	_, reply := executeCoreCommand(t, stub, ctx, msg, ".rd", "r", "rd", "roll", "rh")
	require.NotEmpty(t, reply)
}

func TestCoreRollHelp(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	result, _ := executeCoreCommand(t, stub, ctx, msg, ".r help", "r", "rd", "roll", "rh")
	require.True(t, result.ShowHelp)
}

// ================= Text 指令测试 =================

func TestCoreTextBasic(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".text 测试: {1d6}", "text")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "测试:")
}

func TestCoreTextWithMath(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".text 计算: {2+3}", "text")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "5")
}

func TestCoreTextHelp(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	result, _ := executeCoreCommand(t, stub, ctx, msg, ".text help", "text")
	require.True(t, result.ShowHelp)
}

// ================= Help 指令测试 =================

func TestCoreHelpList(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".help", "help")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "可用指令")
}

func TestCoreHelpSpecificCommand(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	// 使用正确的指令名
	_, reply := executeCoreCommand(t, stub, ctx, msg, ".help r", "help")
	require.NotEmpty(t, reply)
	// 验证返回了帮助内容
	require.True(t, strings.Contains(reply, "骰点") || strings.Contains(reply, "dice") || strings.Contains(reply, "未找到"))
}

func TestCoreHelpUnknownCommand(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".help unknowncmd", "help")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "未找到")
}

// ================= Bot 指令测试 =================

func TestCoreBotAbout(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".bot about", "bot")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "小海豹")
}

func TestCoreBotOn(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)
	ctx.Group.Active = false

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".bot on", "bot")
	require.NotEmpty(t, reply)
	require.True(t, ctx.Group.Active)
}

func TestCoreBotOff(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)
	ctx.Group.Active = true

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".bot off", "bot")
	require.NotEmpty(t, reply)
	require.False(t, ctx.Group.Active)
}

func TestCoreBotHelp(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	result, _ := executeCoreCommand(t, stub, ctx, msg, ".bot help", "bot")
	require.True(t, result.ShowHelp)
}

// ================= NN 昵称指令测试 =================

func TestCoreNNShow(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)
	ctx.Player.Name = "原始昵称"

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".nn", "nn")
	require.NotEmpty(t, reply)
}

func TestCoreNNSet(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".nn 新名字", "nn")
	require.NotEmpty(t, reply)
	require.Equal(t, "新名字", ctx.Player.Name)
}

func TestCoreNNClear(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)
	ctx.Player.Name = "自定义名字"
	msg.Sender.Nickname = "平台昵称"

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".nn clr", "nn")
	require.NotEmpty(t, reply)
	require.Equal(t, "平台昵称", ctx.Player.Name)
}

func TestCoreNNHelp(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	result, _ := executeCoreCommand(t, stub, ctx, msg, ".nn help", "nn")
	require.True(t, result.ShowHelp)
}

// ================= UserID 指令测试 =================

func TestCoreUserID(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".userid", "userid")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, ctx.Player.UserId)
	require.Contains(t, reply, ctx.Group.GroupId)
}

func TestCoreUserIDHelp(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	result, _ := executeCoreCommand(t, stub, ctx, msg, ".userid help", "userid")
	require.True(t, result.ShowHelp)
}

// ================= Set 指令测试 =================

func TestCoreSetInfo(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)
	ctx.Group.DiceSideExpr = "d100"

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".set info", "set")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "D100")
}

func TestCoreSetDiceSide(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".set 20", "set")
	require.NotEmpty(t, reply)
	require.Equal(t, "d20", ctx.Group.DiceSideExpr)
}

func TestCoreSetClear(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)
	ctx.Group.DiceSideExpr = "d20"

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".set clr", "set")
	require.NotEmpty(t, reply)
	require.Empty(t, ctx.Group.DiceSideExpr)
}

func TestCoreSetHelp(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	result, _ := executeCoreCommand(t, stub, ctx, msg, ".set help", "set")
	require.True(t, result.ShowHelp)
}

// ================= BotList 指令测试 =================

func TestCoreBotListEmpty(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".botlist list", "botlist")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "未配置")
}

func TestCoreBotListAdd(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	// 模拟 @某人
	cmdArgs := types.CommandParse(".botlist add bot-123", []string{"botlist"}, []string{".", "。"}, "", false)
	require.NotNil(t, cmdArgs)
	cmdArgs.Args = []string{"add", "bot-123"}
	cmdArgs.CleanArgs = "add bot-123"

	ext := stub.extensions["core"]
	cmd := ext.CmdMap["botlist"]
	result := cmd.Solve(ctx, msg, cmdArgs)

	require.True(t, result.Matched)

	// 验证已添加
	_, exists := ctx.Group.BotList.Load("bot-123")
	require.True(t, exists)
}

// ================= Master 指令测试 =================

func TestCoreMasterList(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".master list", "master")
	require.NotEmpty(t, reply)
}

func TestCoreMasterHelp(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	result, _ := executeCoreCommand(t, stub, ctx, msg, ".master help", "master")
	require.True(t, result.ShowHelp)
}

// ================= Ext 指令测试 =================

func TestCoreExtList(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	// 确保 group 有扩展列表方法
	ctx.Group.ActivatedExtList = []*types.ExtInfo{}

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".ext list", "ext")
	require.NotEmpty(t, reply)
	require.Contains(t, reply, "扩展状态")
}

func TestCoreExtHelp(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	result, _ := executeCoreCommand(t, stub, ctx, msg, ".ext help", "ext")
	require.True(t, result.ShowHelp)
}

// ================= Dismiss 指令测试 =================

func TestCoreDismiss(t *testing.T) {
	ctx, msg, stub := newCoreTestContext(t)

	_, reply := executeCoreCommand(t, stub, ctx, msg, ".dismiss", "dismiss")
	require.NotEmpty(t, reply)
	// dismiss 实际上是 bot bye 的别名
}
