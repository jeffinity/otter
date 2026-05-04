# `otter service` 核心实现设计

本文档描述 `otter service` 的核心实现设计、实现边界和后续落地约束。用户可见的命令用法维护在 `doc/service.md`。

## 目标

`otter service` 是 Linux 服务管理入口，用于服务查询、启停、安装、编辑、审计和自检。当前代码已经建立命令树、flag/alias 契约、Linux 平台门禁、路径配置基础包，并已落地真实业务 handler。

设计目标：

- CLI 契约稳定，所有子命令、别名、flag 和隐藏状态由测试锁定。
- 命令层只做参数解析、互斥校验、handler 编排和用户输出。
- 服务发现、systemd 调用、日志读取、安装生成和审计等能力拆到 `internal/` 子包。
- Linux-only 能力在非 Linux 平台可编译，运行时报明确不支持错误。
- 测试通过接口注入或 fake runner 覆盖，不依赖真实 `systemctl`、`journalctl`、文件系统系统目录或网络。

## 模块边界

| 位置 | 当前职责 | 后续职责 |
| --- | --- | --- |
| `cmd/service.go` | 将顶层命令挂接到 root | 保持薄封装 |
| `internal/service/command` | Cobra 命令树、参数结构、平台门禁、补全占位、服务常量 | 注入 service manager、runner、renderer 并调用真实 handler |
| `internal/service/command/status` | `status` 服务查询、匹配、过滤、排序、systemctl store 和输出渲染 | 为查询类子命令提供共享服务选择逻辑 |
| `internal/service/command/list` | `list` 服务名称输出 | 复用 `status` 包的服务选择和 systemd 查询能力 |
| `internal/service/command/detail` | `detail` 服务选择后前台执行 `systemctl status` | 保持 systemctl 前台输出语义稳定 |
| `internal/service/command/showproperty` | `show-property` 读取并渲染 systemd properties | 保持字段顺序和 `--all` 语义稳定 |
| `internal/service/command/servicefile` | classic/package service 文件定位和 `[X-Otter]` 元信息解析 | 供 view、log、安装接管等命令复用 |
| `internal/service/command/view` | `view` 展示 service 文件和 docker compose 信息 | 保持 classic/package 查找与 header 裁剪语义稳定 |
| `internal/service/command/log` | `log` 选择 custom log 或 journalctl 并前台执行 | 保持 tail/less/journalctl 参数语义稳定 |
| `internal/service/command/daemonreload` | `daemon-reload` 前台执行 systemctl daemon-reload | 保持前台调用语义稳定 |
| `internal/service/command/start` | `start` 选择服务并前台执行 systemctl start | 保持 action 过滤、`--reload`、`--trace`、`--stop-after` 语义稳定 |
| `internal/service/command/stop` | `stop` 选择服务并前台执行 systemctl stop | 保持 action 过滤语义稳定 |
| `internal/service/command/restart` | `restart` 选择服务并前台执行 systemctl restart | 保持 action 过滤、`--reload`、`--trace` 语义稳定 |
| `internal/service/command/reload` | `reload` 选择服务并前台执行 systemctl reload | 保持 action 过滤和 reload 参数错误语义稳定 |
| `internal/service/command/enable` | `enable` 选择服务并前台执行 systemctl enable | 保持 action 过滤和 `--start` 语义稳定 |
| `internal/service/command/disable` | `disable` 选择服务并前台执行 systemctl disable | 保持 action 过滤和 `--stop` 语义稳定 |
| `internal/service/command/grouplist` | `group-list` 读取并渲染服务组 | 保持组名排序、`--one` 和 `--services` 输出语义稳定 |
| `internal/service/command/groupstart` | `group-start` 展开服务组后执行 start | 保持服务组去重、缺组错误和 `--stop-after` 透传语义稳定 |
| `internal/service/command/groupstop` | `group-stop` 展开服务组后执行 stop | 保持服务组去重和缺组错误语义稳定 |
| `internal/service/command/grouprestart` | `group-restart` 展开服务组后执行 restart | 保持服务组去重和缺组错误语义稳定 |
| `internal/service/command/installservice` | `install-service` 安装 classic service | 保持 name 推导和 enable/start 语义稳定 |
| `internal/service/command/installcommand` | `install-command` 生成并安装 command service | 保持模板、`--wd`、`--no-install` 和安装语义稳定 |
| `internal/service/command/installdockercompose` | `exp-install-docker-compose` 生成 docker compose service | 保持 compose 文件选择、复制和 service 生成语义稳定 |
| `internal/service/command/linkservice` | `link-service` 接管已有 systemd unit | 保持 symlink 接管语义稳定 |
| `internal/service/command/edit` | `edit` 编辑 classic/package service | 保持 editor、保存、regen 和 restart 确认语义稳定 |
| `internal/service/command/regenerate` | `re-generate` 刷新 package service | 保持 restart flag 语义稳定 |
| `internal/service/command/audit` | `audit` 写入 systemd hook 审计 | 保持 hidden、必填 flag 和 audit bypass 语义 |
| `internal/service/command/showpids` | `show-pids` 读取服务 cgroup PID | 保持 cgroup.procs 读取和空格分隔输出语义稳定 |
| `internal/service/command/showports` | `show-ports` 查询服务 PID 对应监听端口 | 保持 LISTEN 连接输出语义稳定 |
| `internal/service/command/selfcheck` | `self-check` 以维护子程序名重新执行当前二进制 | 保持 hidden pass-through 自检语义稳定 |
| `internal/service/command/install` | `install` 以维护子程序名重新执行当前二进制 | 保持 hidden pass-through 安装入口语义稳定 |
| `internal/service/command/upsertself` | `upsert-self` 以维护子程序名重进当前二进制 | 保持 hidden reexec 自更新入口语义稳定 |
| `internal/service/command/upsertcluster` | `upsert-cluster` 以维护子程序名重进当前二进制 | 保持 hidden reexec 集群更新入口语义稳定 |
| `internal/otterfs` | 可配置路径 provider | 为测试和安装流程提供路径注入 |
| `pkg/tuix` | 用户输出渲染 | 用于状态、列表、详情等命令输出 |
| `pkg/logx` | 日志 | 用于调试和错误上下文 |

后续新增内部包建议按能力拆分：

| 建议包 | 职责 |
| --- | --- |
| `internal/servicediscovery` | 读取 systemd、classic、package service 元数据，做服务匹配和过滤 |
| `internal/systemd` | 封装 `systemctl`、`journalctl`、unit 文件路径、drop-in 写入 |
| `internal/serviceinstall` | 生成和安装 service、command service、docker compose service |
| `internal/serviceaudit` | systemd hook 审计记录与 bypass 语义 |
| `internal/servicegroup` | 服务组读取、匹配、批量动作 |

包名可以按实际落地微调，但必须保持命令层不直接拼接系统命令、不直接读写系统路径。

## 命令树契约

根命令：

```text
otter service (status) <services...>
```

关键行为：

- `PersistentPreRunE` 统一做平台门禁：`runtime.GOOS != linux` 时返回 `ErrUnsupportedPlatform`。
- 当前不提供 `--cluster` / `-c` 集群模式参数。
- 根命令无子命令时默认分发到 `status` 语义。
- 子命令通过 `ValidArgsFunction` 保留补全入口。

命令类别：

| 类别 | 命令 |
| --- | --- |
| 查询 | `status`, `list`, `detail`, `show-property`, `view`, `log`, `show-pids`, `show-ports` |
| 动作 | `start`, `stop`, `restart`, `reload`, `enable`, `disable`, `daemon-reload` |
| 服务组 | `group-list`, `group-start`, `group-stop`, `group-restart` |
| 安装接管 | `install-service`, `install-command`, `exp-install-docker-compose`, `link-service` |
| 编辑刷新 | `edit`, `re-generate` |
| 隐藏维护 | `audit`, `self-check`, `install`, `upsert-self`, `upsert-cluster` |

任何命令名、别名、flag、默认值、hidden 状态或参数数量变化，都必须同步更新：

- `internal/service/command/command_test.go`
- `doc/service.md`
- 本设计文档

## 依赖注入设计

`command.New(deps Dependencies)` 当前注入 `RuntimeOS`、`FS`、`StatusStore`、`ListStore`、`DetailStore`、`StatusRunner`、`DetailRunner`、`ShowPropertyGetter`、`ShowPropertyRunner`、`DaemonReloadRunner`、`ActionStore`、`ActionRunner`、`ActionAutoStopper`、`ActionTraceRunner`、`ActionExecutable`、`ActionEnviron`、`GroupListStore`、`ShowPidsFinder`、`ShowPortsPidFinder`、`ShowPortsConnFinder`、`SelfCheckRunner`、`SelfCheckExecutable`、`SelfCheckEnviron`、`InstallRunner`、`InstallExecutable`、`InstallEnviron`、`UpsertSelfRunner`、`UpsertSelfPath`、`UpsertSelfEnviron`、`UpsertSelfMkdirAll`、`UpsertClusterRunner`、`UpsertClusterPath`、`UpsertClusterEnv`、`UpsertClusterMkdir`、`ViewFinder`、`LogFinder`、`LogRunner`、`LogLookPath`、`Out`、`ErrOut`、`In`、`NoColor`、`Now` 和 `MonoNow`。后续真实实现应继续扩展 `Dependencies`，但要保持零值可用。

建议依赖：

| 依赖 | 用途 |
| --- | --- |
| `RuntimeOS string` | 平台门禁和测试 |
| `FS otterfs.Provider` | 系统路径和测试路径注入 |
| `Runner CommandRunner` | 执行外部命令 |
| `Services ServiceStore` | 服务发现、查询、过滤 |
| `Systemd SystemdClient` | systemd 动作、属性、日志 |
| `Installer Installer` | service 安装、生成、链接 |
| `Groups GroupStore` | 服务组查询和批量动作 |
| `Out Renderer` | 用户输出 |

接口应按消费者定义在最靠近使用方的包内，不要提前抽象大而全的 manager。测试 fake 应覆盖命令需要的行为和参数断言。

## 服务模型

服务发现层应输出统一服务模型，屏蔽 systemd、classic、package 的来源差异。

建议核心字段：

| 字段 | 说明 |
| --- | --- |
| `Name` | 规范化服务名，不带 `.service` 后缀 |
| `UnitName` | systemd unit 名，带 `.service` 后缀 |
| `Source` | `classic`、`package`、`systemd` 等来源 |
| `Enabled` | 是否启用 |
| `ActiveState` | systemd active state |
| `SubState` | systemd sub state |
| `MainPID` | 主进程 PID |
| `FragmentPath` | unit 文件路径 |
| `DropInPaths` | drop-in 路径 |
| `Metadata` | `[X-Otter]` 扩展信息 |

unit metadata 必须使用 `[X-Otter]`，不读取或写入旧 metadata 段名。

默认服务发现只从 otter 管理路径收集服务名：classic service 来自 `ClassicServicePath/*.service`，package service 来自 `PackageServicePath/*/*/*.service`，再对这些 unit 执行 `systemctl show` 获取状态。`status`、`list`、`detail`、启停动作和服务组动作都应使用该受管理服务集合，避免默认展示或批量操作系统中所有 systemd service。

需要接管已有 systemd unit 的 `link-service` 可以使用全系统发现能力，从 `systemctl list-unit-files` 和 `systemctl list-units` 收集 systemd service；该路径同样只保留具体 service unit。未实例化模板 unit（`name@.service`）不能传给 `systemctl show`，应在发现阶段跳过；已实例化的 `name@instance.service` 继续按普通 service 处理。

## 路径设计

路径统一由 `internal/service/command` 常量和 `internal/otterfs.Provider` 提供。

固定常量：

| 用途 | 值 |
| --- | --- |
| core unit | `otter-core.service` |
| core socket | `/var/run/otter-core.socket` |
| TCP listen | `0.0.0.0:3456` |
| TCP dial | `127.0.0.1:3456` |
| env 文件 | `/etc/otter/systemd.env` |
| systemd drop-in | `/etc/otter/systemd.conf` |
| audit drop-in | `/etc/otter/audit.conf` |
| audit log | `/etc/otter/otter-core-audit.log` |

可注入路径默认值：

| 用途 | 默认值 |
| --- | --- |
| 配置 | `/etc/otter/.config` |
| machine id | `/etc/otter/machine-id` |
| service db | `/etc/otter/otter-service.db` |
| roles | `/etc/otter/roles` |
| package data | `/data/.otter/otter-packages` |
| systemd units | `/usr/lib/systemd/system` |
| classic services | `/etc/otter/services` |
| package services | `/etc/otter/services/.do-not-edit` |
| scripts | `/etc/otter/scripts` |
| cluster targets | `/etc/otter/targets` |
| current target | `/etc/otter/target` |

测试必须使用 `otterfs.New(Config{...})` 注入临时路径，不直接写入系统目录。

## 服务匹配与过滤

服务匹配规则应由服务发现层集中实现：

- 用户输入可以带 `.service` 后缀，内部规范化为不带后缀服务名。
- `all` 和 `*` 是特殊服务 pattern。
- `all` 或 `*` 必须是唯一 pattern。
- 普通 glob pattern 必须至少匹配一个已知服务。
- 匹配结果应稳定排序，避免输出和测试不稳定。

过滤选项：

| flag | 语义 |
| --- | --- |
| `--disabled` | 包含 disabled 服务 |
| `--no-enabled` | 排除 enabled 服务 |
| `--only-package` | 仅 package 来源 |
| `--only-classic` | 仅非 package 来源 |

动作类命令默认只在 otter 关注的服务集合内作用于 enabled 服务，除非显式选择 disabled。列表类命令默认在 otter 关注的服务集合内展示 enabled 或 running 服务。过滤冲突应在命令层或发现层返回明确错误，不静默选择空集合。

`status` 状态渲染应保持统一：运行中的状态字段使用绿色，非运行状态字段使用红色，`, disabled` 标记使用黄色；`--time-info` 的 monotonic 时间缺失或系统时间变化提示使用红色。列宽计算必须基于不带 ANSI 的状态文本，避免彩色输出破坏对齐。

其他终端展示样式也应保持统一：

- `show-property`：property 名称蓝色，`<no value>` 黄色，`NeedDaemonReload=true` 红色。
- `view` 和 `log`：service 正文、Docker Compose 注释块、执行命令预览均保持 plain text，不额外使用 tuix 样式。
- `link-service`：成功创建 symlink 后输出 `Create fake service (linked to <path>) success.`。
- `install-service`、`edit`、`re-generate`：保留安装成功、保存/未变更、重新生成、是否重启和完成等关键提示。
- `start --trace` / `restart --trace`：匹配多个服务时输出黄色 warning，并用 plain text 列出可手工执行的 `otter service log -f <service>` 命令。
- `audit`：缺少 `--service-name` 或 `--action-name` 时输出红色 `Maybe you want to use \`otter-audit\`?` 提示，不写审计记录。

## systemd 与日志设计

真实 systemd 能力必须通过接口封装，不在命令 handler 中直接调用 `exec.Command`。

建议 `SystemdClient` 能力：

| 能力 | 底层命令 |
| --- | --- |
| `Status/List` | `systemctl show` over otter managed units |
| `Link discovery` | `systemctl show`, `systemctl list-unit-files`, `systemctl list-units` |
| `Start/Stop/Restart/Reload` | `systemctl start/stop/restart/reload` |
| `Enable/Disable` | `systemctl enable/disable` |
| `DaemonReload` | `systemctl daemon-reload` |
| `Properties` | `systemctl show` |
| `Logs` | `journalctl -u` |

日志命令规则：

- `--follow` 不能和 `--until` 同时使用。
- 不传时间参数时，`--lines` 默认应按用户文档语义使用 80 行；`-1` 表示不限制。
- `--output` 透传 journalctl 支持的输出模式。
- `--force-journalctl` 是隐藏兼容选项。

## 安装与接管设计

安装类命令写入 systemd unit、drop-in、脚本和 package 目录，必须经过可注入路径和文件系统封装。

### `install-service`

输入现有 service 文件：

1. 解析 service 名，`--name` 优先，否则使用文件名。
2. 校验 unit 文件基本结构。
3. 复制到 otter 管理路径或 systemd 路径。
4. 写入 `[X-Otter]` metadata / drop-in。
5. 根据 `--no-enable`、`--no-start` 决定是否 enable/start。

### `install-command`

根据命令生成 service：

1. `--name` 必填。
2. `--wd -` 表示当前目录。
3. `--no-install` 时只输出生成内容，不写文件、不执行 systemd。
4. 正常安装时写 unit 并按 flag enable/start。

### `exp-install-docker-compose`

根据 docker compose 文件生成 service：

1. 未传 file 时默认当前目录 `docker-compose.yml`。
2. `--dir` 默认为 `/opt/<服务名>`。
3. 默认不覆盖已有 compose 文件，`--force` 才允许覆盖。
4. 生成的 service 记录 compose 文件路径和基础目录。

### `link-service`

接管已有 systemd service：

1. 校验 unit 存在。
2. 不改写原始 service 主体。
3. 通过 drop-in 或 otter metadata 将其纳入管理。

## 审计与隐藏命令

`audit` 是 systemd hook 入口，必须保持隐藏并要求：

- `--service-name`
- `--action-name`

缺少任一 flag 时应保持明确的交互语义：输出红色迁移提示后直接返回，不执行审计写入。审计流程应支持 `OTTER_AUDIT_BYPASS` 环境变量，避免 otter 自身维护动作递归触发不必要审计。审计写入路径由 `command.OtterCoreAuditFilePath` 或可注入配置提供。

隐藏维护命令 `audit`、`self-check`、`install`、`upsert-self`、`upsert-cluster` 已按 hidden 入口语义落地，并保持隐藏状态、别名和 `DisableFlagParsing`。`audit` 在 Otter 中写入本地 JSONL 审计文件，使用 `OTTER_AUDIT_BYPASS` 控制跳过审计或写入错误是否阻断调用方。

## 输出设计

命令输出应使用 `pkg/tuix` 或内部 renderer，避免 handler 直接散落 `fmt.Println`。

输出要求：

- `status`、`list`、`group-list` 输出稳定排序。
- `--one` 使用单行机器友好形式。
- `detail` 和 `view` 可使用 pager，但 `--no-pager` 必须完全绕过 pager。
- 错误输出保留可定位服务名、动作名和底层错误。

## 平台策略

代码必须在非 Linux 平台编译通过。Linux-only 行为集中在运行期门禁和封装层：

- `internal/service/command` 根命令先返回 `ErrUnsupportedPlatform`。
- Linux 专用实现如需使用 build tags，应提供非 Linux stub。
- 测试可以通过 `Dependencies{RuntimeOS: "linux"}` 覆盖平台分支。

## 错误语义

错误应保持用户可修复：

- 非 Linux：`otter service is only supported on linux`。
- 未实现 handler：仅保留内部防御路径，普通业务子命令不应返回 not implemented。
- 参数互斥：例如 `--asc` 与 `--desc` 同时出现必须报错。
- 服务 pattern 无匹配：说明具体 pattern。
- 系统命令失败：包含动作、服务名和命令输出摘要。
- 文件写入失败：包含目标路径。

## 测试策略

已有测试覆盖：

- 非 Linux 平台门禁。
- 根命令默认分发到 status。
- 子命令存在性、alias、hidden 状态。
- 关键 flags 存在性。
- `status --asc --desc` 互斥校验。
- `internal/service/command` 常量和 `otterfs.Provider` 路径契约。

`status` 已补充 fake store/runner 单元测试，覆盖 pattern、`.service` 后缀、`all` / `*`、enabled/disabled/package/classic 过滤、排序、时间信息和 systemctl 参数。`list` 已补充服务选择、source 过滤、多行/单行输出和命令层接入测试。`detail` 已补充服务选择、`--no-pager` 参数、runner 错误透传和命令层接入测试。`show-property` 已补充默认字段、`--all`、时间格式、runner 参数和命令层接入测试。`view` 已补充 classic/package service 查找、header 裁剪和 compose 信息输出测试。`log` 已补充 custom log、journalctl、tail/less 参数、时间过滤、lookPath/runner 错误和命令层接入测试。`daemon-reload` 已补充 runner 参数、前缀输出、错误透传和命令层接入测试。`start`、`stop`、`restart`、`reload` 已补充 action 服务过滤、`all` / `*`、`.service` 后缀、`--reload`、`--trace`、`--stop-after` 注入和命令层接入测试；默认 auto-stop 通过结构化 `systemd-run --on-active=... systemctl stop ...` 调度。`enable`、`disable` 已补充 action 服务过滤、后续 start/stop 和命令层接入测试。`group-list` 已补充组排序、多行/单行输出、服务输出、classic/package service 文件 `[X-Otter] Group` 解析和命令层接入测试。`group-start`、`group-stop`、`group-restart` 已补充组展开、重复服务去重、缺失组错误、`--stop-after` 透传和命令层 alias 接入测试。安装与接管命令已补充 service 生成、compose 复制、symlink 接管、systemctl 参数和命令层接入测试。`edit`、`re-generate` 已补充 editor、prompt、regen、restart 决策和命令层接入测试。`audit` 已补充 bypass、must-audit 写入错误和命令层接入测试。`show-pids` 已补充 `.service` 后缀、cgroup v1/v2 路径、缺失 cgroup、非法 PID 和命令层接入测试。`show-ports` 已补充 PID 查询、LISTEN 过滤、连接错误跳过、`/proc/net/tcp` 解析和命令层接入测试。隐藏维护入口 `self-check`、`install`、`upsert-self`、`upsert-cluster` 已补充当前二进制重新执行、参数透传、`AS=` 环境变量处理、日志目录预创建和命令层 alias 接入测试。

后续实现真实 handler 时，按风险补充：

- 命令层：参数校验、flag 默认值、互斥关系、hidden/alias 不变。
- 服务发现：pattern、`.service` 后缀、`all` / `*`、enabled/disabled/package/classic 过滤。
- systemd client：使用 fake runner 断言命令参数，不执行真实 `systemctl`。
- 安装流程：使用临时路径和 fake systemd，覆盖 no-install、force、enable/start 组合。
- 日志：覆盖 `--follow` / `--until` 冲突、默认 lines、输出模式传递。
- 非 Linux：所有 Linux-only 命令运行期返回明确错误且包可编译。

修改用户可见命令契约时同步 `doc/service.md`；修改核心流程、路径、副作用、错误语义或测试策略时同步本文档。

## 落地顺序

建议按以下顺序落地，降低一次性变更风险：

1. 抽象 `CommandRunner`、输出 renderer 和 service manager 接口，替换 `notImplemented` 的最小查询路径。
2. 落地服务发现与过滤，先支撑 `status`、`list`、`detail`。
3. 落地 systemd 动作命令：`start`、`stop`、`restart`、`reload`、`enable`、`disable`、`daemon-reload`。
4. 落地日志、属性、view、pid/port 查询。
5. 落地安装、链接、编辑、re-generate。
6. 落地审计、自检和自更新。

每一步都必须保持现有命令契约测试通过，并补充对应业务测试。

## 设计约束

- 不在单元测试中真实执行 `systemctl`、`journalctl`、`ssh`、`docker` 或网络请求。
- 不在测试中写入 `/etc`、`/usr/lib/systemd`、`/var/run`、`/data`。
- 不在命令层拼接复杂 shell；外部命令参数必须通过 runner 结构化传递。
- 不读写旧 metadata 段名。
- 不把服务发现、过滤、系统调用和输出渲染混在同一个函数里。
- 新增文件接近 800 行时按职责拆分。
