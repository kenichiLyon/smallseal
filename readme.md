<div align="center">

# Small Seal Shell

[![Tests](https://github.com/sealdice/smallseal/actions/workflows/test.yml/badge.svg)](https://github.com/sealdice/smallseal/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sealdice/smallseal)](https://goreportcard.com/report/github.com/sealdice/smallseal)

一个强大的 DiceScript 虚拟机骰子游戏系统框架

支持 **CoC 7th** | **D&D 5e** | 更多规则扩展中...

</div>

---

## 快速开始

```bash
go run .
```

```
DiceScript Shell v0.0.1
>>> .r d100 测试
由于 测试，<seal>掷出了 d100=96=96
```

## 模块状态

| 模块 | 状态 | 说明 |
|------|:----:|------|
| core | ✅ | 内置核心功能 |
| coc7 | ✅ | 克苏鲁的呼唤 7版 |
| dnd5e | ✅ | 龙与地下城 5版 |
| reply | ⏳ | 自定义回复 |
| helpdoc | ⏳ | 帮助文档 |
| blacklist | ⏳ | 黑名单 |
| log | ⏳ | 日志记录 |
| fun | ⏳ | 娱乐功能 |

## TODO

- [x] adapter 重新设计
- [ ] 指令回复时携带更多信息，能进行二次渲染
- [x] 加入几个接收消息、发送后的钩子，以便于进行预处理、格式转换、敏感词处理等
- [ ] 指令解析
- [x] 实现 Text 和 Segment[] 的无缝互转
- [ ] 想一下 message_old.go 里面的 Message 能否和返回的那个 MsgToReply 统一化？
- [x] 重新实现coc难度检定
- [ ] detailOverwrite 机制对同义词无效，例如 .r pp 有效，但 .r pw 无效
- [ ] 可能与ds有关：需要一个方便的办法去设置detail，因为现在的 detail 机制只能处理最上面一层，深层是自动的，无法进行重写

## 内置扩展函数

<details>
<summary><code>toJSON(value)</code> - 将任意值转换为JSON字符串</summary>

**参数:**
- `value`: 要转换的值，可以是任意类型

**返回值:**
- 成功时返回JSON格式的字符串
- 失败时返回null并设置错误

**示例:**
```javascript
toJSON({name: "张三", age: 25})  // 返回: {"t":7,"v":{"dict":{"name":{"t":2,"v":"张三"},"age":{"t":0,"v":25}}}}
toJSON([1, 2, 3])               // 返回: {"t":6,"v":{"list":[{"t":0,"v":1},{"t":0,"v":2},{"t":0,"v":3}]}}
```
</details>

<details>
<summary><code>parseJSON(jsonStr)</code> - 将JSON字符串解析为对应的值</summary>

**参数:**
- `jsonStr`: 要解析的JSON字符串

**返回值:**
- 成功时返回解析后的值
- 失败时返回null并设置错误

**示例:**
```javascript
parseJSON('{"t":7,"v":{"dict":{"name":{"t":2,"v":"张三"},"age":{"t":0,"v":25}}}}')  // 返回对象 {'age': 25, 'name': '张三'}
parseJSON('{"t":6,"v":{"list":[{"t":0,"v":1},{"t":0,"v":2},{"t":0,"v":3}]}}')      // 返回数组 [1,2,3]
```
</details>

<details>
<summary><code>configShow()</code> - 显示当前虚拟机的配置信息</summary>

**参数:** 无

**返回值:** null

**功能:**
- 显示骰子语法开关状态（WOD、CoC、Fate、双重十字等）
- 显示语法限制设置（位运算、语句语法、Nd语法等）
- 显示性能限制参数（算力限制、默认骰子面数等）
</details>

<details>
<summary><code>configSet(key, value)</code> - 设置虚拟机配置项</summary>

**参数:**
- `key`: 配置项名称（字符串）
- `value`: 配置项的值

**返回值:**
- 成功时返回1
- 失败时返回null

**支持的配置项:**

| 配置项 | 说明 |
|--------|------|
| `EnableDiceWoD` | 启用WOD语法 (XaYmZkNqM格式) |
| `EnableDiceCoC` | 启用CoC语法 (bX/pX奖惩骰) |
| `EnableDiceFate` | 启用Fate语法 (fX格式) |
| `EnableDiceDoubleCross` | 启用双重十字语法 (XcY格式) |
| `DisableBitwiseOp` | 禁用位运算 |
| `DisableStmts` | 禁用语句语法 |
| `DisableNDice` | 禁用Nd语法 |
| `ParseExprLimit` | 解析算力限制 |
| `OpCountLimit` | 运算算力限制 |
| `DefaultDiceSideExpr` | 默认骰子面数 |
| `PrintBytecode` | 打印字节码 |
| `IgnoreDiv0` | 忽略除零错误 |

**示例:**
```javascript
configSet("EnableDiceCoC", true)   // 启用CoC语法
configSet("OpCountLimit", 10000)   // 设置运算限制
```
</details>

## 内置变量

| 变量 | 说明 |
|------|------|
| `player` | 人物卡 |
