# WIP
# miniagent

`miniagent` 是一个面向多轮智能体交互的 Go 项目。当前代码已经具备核心对话数据结构、模型 provider 抽象、单轮/多轮 agent loop，以及 Bubble Tea TUI 镜像渲染。后续架构目标是把系统稳定收敛为三个运行时边界：工具运行时、智能体运行时、用户界面。

本文档描述目标架构，同时标注当前实现状态。`core/` 是现有跨层契约目录，后续目录重排不移动它。

早期设计阶段参考了 [DeepSeek-Reasonix](https://github.com/deepseek-ai/DeepSeek-Reasonix) 的部分代码作为 `core/` 的部分类型和约束的参考

## 目标架构

系统运行时分为三部分：

1. 工具运行时（Tool Runtime）
   - 负责工具注册、schema 暴露、工具执行、取消、状态查询、进度事件和结果事件。
   - 包含除子智能体以外的所有可执行能力。
   - 不直接读取或修改任何 agent 的 `[]core.Turn`。

2. 智能体运行时（Agent Runtime）
   - 是对话状态与决策核心。
   - 每个 agent 持有自己的权威 `[]core.Turn`。
   - 一个 agent 由 prompt/提示规则、模型 provider、允许使用的工具/能力集合组成。
   - 负责处理用户操作、模型请求、工具事件、子智能体调度，并向 UI 输出同步事件。
   - 不承担工具执行细节，也不保存 UI 的临时渲染状态。

3. 用户界面（User Interface）
   - 负责下达指令、观察 agent 状态、展示 transcript 和辅助信息。
   - 只维护 `[]core.Turn` 的镜像副本以及临时 UI 状态。
   - 所有会改变对话语义的操作都必须发送给 Agent Runtime，由 Agent Runtime 决定并通过事件同步结果。

## 通信规则

系统边界必须保持单向事件流和只读查询的区别：

| 方向 | 通道类型 | 内容 |
| --- | --- | --- |
| Agent Runtime -> UI | 事件流 | `core.TurnSyncPrimitive`、`core.KeyNotify`，以及必要的外层 agent identity |
| UI -> Agent Runtime | 用户操作命令流 | 用户输入、打断、退出、revert 等 UI 自身无法完成的语义操作 |
| Agent Runtime -> Tool Runtime | 工具执行命令流 | 工具调用、取消、执行策略 |
| Tool Runtime -> Agent Runtime | 工具事件流 | 工具开始、进度、完成、失败、取消结果 |
| UI -> Agent Runtime / Tool Runtime | 只读查询 | 仅用于辅助渲染界面元素，例如按 id 查询长任务状态 |

除上述关系外，不允许跨层直接通信。尤其是 UI 不能直接调用 agent 的 setter 修改权威 `[]core.Turn`，Tool Runtime 也不能直接向 UI 推送会改变对话语义的事件。

## `[]Turn` 同步原则

`[]core.Turn` 是每个 agent 的核心对话状态。

- Agent Runtime 持有权威副本。
- UI 持有镜像副本，并通过 `core.TurnSyncPrimitive` 增量同步。
- `core.KeyNotify` 表示不改变 `[]core.Turn` 的关键事件，可用于状态栏、动画、阶段显示等 UI 行为。
- UI 可以自由维护仅用于渲染的临时状态，例如滚动位置、选区、折叠状态、动画状态和短期查询缓存。
- UI 不得在本地先改对话镜像再要求 Agent Runtime 追认该修改。

以 revert 为例，正确流程是：

1. UI 发送 `Revert` 用户操作给 Agent Runtime。
2. Agent Runtime 判断 revert 是否允许，并修改自己的权威 `[]core.Turn`。
3. Agent Runtime 通过同步事件让 UI 镜像收敛到新状态。

当前 `core.TurnSyncPrimitive` 覆盖新增 turn（`NewTurn`）、追加消息（`AppendMessages`）、开始响应（`StartResponding`）、delta chunk（`DeltaChunk`）、修改/追加 turn（`ModifyOrAppendTurns`）和删除 turn（`Remove`）。后续如果严格保持 `core/` 类型不变，应在 `core` 外增加外层 snapshot/reset 事件；如果允许扩展 `core` 类型，则应补齐对应的同步原语。

## 多智能体规则

- 每个 agent 拥有独立的权威 `[]core.Turn`。
- 子智能体属于 Agent Runtime 内部调度，不属于 Tool Runtime。
- UI 展示多个 agent 时，必须用外层 agent identity 包装同步事件。
- 不应把 agent id 塞进现有 `core.TurnSyncPrimitive`；这些原语只描述单个 `[]core.Turn` 持有者之间的同步。

## 当前实现状态

当前仓库的主要结构如下：

- `core/`
  - 已包含跨层契约：`Turn`、`Message`、`Provider`、`Tool`、`ToolSetAndRunner`、`TurnSyncPrimitive`（含 `ModifyOrAppendTurns`、`Remove`）、`KeyNotify`、`TurnMirror`。
  - 后续目录重排保持原位。

- `providers/`
  - 当前包含 OpenAI-compatible（`providers/openai`）和 Anthropic-compatible（`providers/anthropic`）provider 实现。
  - 均基于 SSE 流式协议，支持 reasoning 和 thinking 模式。

- `agent/intra_turn`
  - 负责单轮模型流、工具调用循环、工具结果追加，以及中断后工具调用补丁（`interrupt_patch.go`）。
  - 目标上属于 Agent Runtime 的内部单轮执行层。

- `agent/conversation`
  - 通过 `PlainConversationCtrl` 持有 `[]core.Turn`，接收命令队列，驱动多轮状态机，并输出 `core.NotifyOrSyncEvents`。
  - 目标上属于 Agent Runtime 的多轮调度层。
  - （注意：早期曾记为 `agent/inter_turn`，现已迁移至 `agent/conversation`。）

- `ui/tui/view_model/`
  - 包含两套并行的 TUI view model 实现：
    - `agent_interact/`：富交互，基于 Block 渲染，支持折叠/展开、选中复制、动画
    - `agent_interact_simple/`：轻量交互，基于 viewport 文字渲染
  - 各子模块：
    - `block/`：Turn 块渲染引擎（分区收集、折叠/展开状态、渲染样式）
    - `transcript/`：基于 `LazyScrollView` 的可滚动对话记录
    - `status_bar/`：Agent 阶段状态栏、spinner、ESC 打断
    - `prompt_input/`：多行文本输入组件
  - 总体基于 `core.TurnMirror` 维护 UI 镜像，符合目标方向。

- `ui/tui/common/`
  - 可复用 TUI 基础组件库：`LazyScrollView`、`TextareaWidget`、`SpinnerWidget`、`SelectionOverlay`、`Theme`、`StreamColumn`、`ColumnRelocatorRoot`、动画 tick 等。
  - 目标上属于 UI 层通用基础设施。

- `cmd/single_turn`
  - 非交互单轮 CLI demo，使用 `intra_turn.RunTurn()` 直接驱动，便于调试和基准测试。

- `cmd/multi_turn`
  - 可交互多轮 TUI 入口，使用 `agent_interact.ReadWriteModel`。
  - `cmd/multi_turn/readonly` 是同一多轮流程的只读特殊入口，用预置输入驱动 agent 并复用 TUI 的只读模型。

- `cmd/.../tools.go`
  - 当前包含重复的 demo tool registry（`echo`、`add`）。
  - 目标上应迁移到非入口目录，例如 `toolruntime/` 和 `internal/demo/tools/`。

工具运行时目前尚未独立实现。现在只有 `core.ToolSetAndRunner` 抽象和各 `cmd` 入口中的临时 registry，因此长任务状态查询、工具进度事件、工具取消协议仍属于后续工作。

## 当前已知问题

- 工具运行时还没有独立事件模型，长工具调用期间的 agent 决策逻辑还不能完整表达。
- `revert` 功能尚未实现；虽然同步协议已支持 `Remove` 和 `ModifyOrAppendTurns`，但 Agent Runtime 侧缺少对应的命令处理逻辑和 UI 交互入口。
