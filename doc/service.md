# `otter service`

`otter service` 是服务管理命令入口，用于查询、启停、安装、编辑和审计 otter 关注的本机 systemd service。

该子命令仅支持 Linux。非 Linux 平台运行时应返回：

```text
otter service is only supported on linux
```

## 用法

```bash
otter service
otter service <services...>
otter service <subcommand> [flags] [args...]
```

默认行为：

```bash
otter service
otter service status

otter service api worker
otter service status api worker
```

当前不提供 `--cluster` / `-c` 集群模式参数。

## 服务查询命令

### `status`

查看服务状态。

```bash
otter service status [services...]
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |
| `--only-package` | 仅包含来源于 package 的服务 |
| `--only-classic` | 仅包含来源非 package 的服务 |
| `-t, --time-info` | 显示时间信息 |
| `--asc` | 按变化时间距离当前的时间差升序排列 |
| `--desc` | 按变化时间距离当前的时间差倒序排列 |
| `-s, --since <duration>` | 仅展示距当前多久之内有过启动或停止的服务 |
| `-M, --no-mono` | 不使用 monotonic 时间 |

行为：

- 不传服务名时，默认展示 enabled 或 running 服务。
- 服务候选集来自 otter 管理路径：classic service 位于 `/etc/otter/services/*.service`，package service 位于 `/etc/otter/services/.do-not-edit/*/*/*.service`；不会默认展示系统中所有 systemd service。
- 显式传入服务名或 glob 时，按匹配结果展示，不因 disabled 默认过滤被隐藏。
- `all` 和 `*` 表示所有 otter 关注的服务，且必须作为唯一服务 pattern。
- 未实例化的 systemd 模板 unit（例如 `name@.service`）不会作为可操作服务展示；已实例化的 `name@instance.service` 仍按普通服务处理。
- 默认按服务名稳定升序；`--asc` / `--desc` 按服务最近启动或停止时间距当前的时间差排序。
- `--time-info` 展示最近启动或停止时间和相对时长；默认优先使用 monotonic 时间，`--no-mono` 改用 wall clock 时间。
- 状态展示保持统一：running 状态为绿色，非 running 状态为红色，disabled 标记为黄色，time-info 异常提示为红色。
- `--only-package` 与 `--only-classic` 不能同时使用。

### `list`

列出服务名称。

```bash
otter service list [services...]
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-1, --one` | 在一行中展示 |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |
| `--only-package` | 仅包含来源于 package 的服务 |
| `--only-classic` | 仅包含来源非 package 的服务 |

行为：

- 服务选择规则与 `status` 一致。
- 默认每行输出一个服务名，且不带 `.service` 后缀。
- `--one` 在单行中以空格分隔输出所有服务名。

### `detail`

获取服务详细信息。

```bash
otter service detail [services...]
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |
| `-P, --no-pager` | 不使用翻页 |

行为：

- 服务选择规则与 `status` 一致。
- 筛选服务后等价执行 `systemctl status <services...>`。
- `--no-pager` 会追加到 systemctl 参数末尾。

### `show-property`

查看 systemd service properties。

```bash
otter service show-property <service>
otter service show <service>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-a, --all` | 展示所有参数 |

行为：

- 输入服务名可以带 `.service` 后缀。
- 默认按固定字段顺序展示常用 systemd properties。
- `--all` 会在默认字段后追加其他 properties。
- 时间字段会展示 RFC3339 时间和相对时间；无值时展示 `-`。
- 终端展示保持统一：property 名称为蓝色，缺失值 `<no value>` 为黄色，`NeedDaemonReload=true` 为红色。

### `view`

展示服务文件。若服务文件包含 docker compose 元信息，会附带展示 compose 文件内容。

```bash
otter service view <service>
```

行为：

- 输入服务名可以带 `.service` 后缀。
- 优先展示 classic service 文件；不存在时查找 package service 文件。
- package service 会裁剪开头的生成注释 header。
- 若 `[X-Otter]` 中配置了 `DockerComposeBaseDir`，会追加展示对应的 `docker-compose.yml`。
- service 文件正文和 Docker Compose 辅助注释块都按原始 plain text 输出，不额外注入 ANSI。

### `log`

查看服务日志。

```bash
otter service log <service>
otter service logs <service>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-f, --follow` | 实时跟踪日志 |
| `-n, --lines <n>` | 日志行数，默认 80，`-1` 表示不限制 |
| `-S, --since <time>` | 筛选开始时间 |
| `-U, --until <time>` | 筛选结束时间 |
| `-o, --output <mode>` | journalctl 输出模式，默认 `cat` |
| `-e, --pager-end` | 在 pager 中直接跳到末尾 |
| `-r, --reverse` | 新日志优先 |
| `-F, --force-journalctl` | 强制使用 journalctl，隐藏选项 |

行为：

- 输入服务名可以带 `.service` 后缀。
- 若 service 文件 `[X-Otter]` 中配置 `LogFile`，默认使用 `tail` / `less` 查看该文件。
- 未配置 `LogFile` 或指定 `--force-journalctl` 时，使用 `journalctl --unit=<service>`。
- 不指定时间过滤时默认展示 80 行；`--lines -1` 在 journalctl 下表示全部日志。
- 执行前会以 plain text 输出实际执行命令，日志内容本身不注入 ANSI。

### `show-pids`

查看服务对应进程的 PID。

```bash
otter service show-pids <services...>
otter service pid <services...>
```

别名：

- `show-pid`
- `pids`
- `pid`

行为：

- 输入服务名可以带 `.service` 后缀。
- 按传入顺序读取每个服务对应 systemd cgroup 的 `cgroup.procs`。
- 输出所有 PID，使用空格分隔；没有 PID 时输出空行。

### `show-ports`

查看服务对应进程的监听端口。

```bash
otter service show-ports <services...>
otter service port <services...>
```

别名：

- `show-port`
- `ports`
- `port`

行为：

- 输入服务名可以带 `.service` 后缀。
- 先读取服务对应 cgroup PID，再查询这些进程的监听连接。
- 仅输出处于 `LISTEN` 状态的连接，格式为 `<TCP|UDP|???> Listen <ip>:<port>`。
- 单个 PID 连接查询失败时跳过该 PID。

## 服务动作命令

`start`、`stop`、`restart`、`reload`、`enable` 会先按统一服务选择规则选择服务，再前台执行 `systemctl <action> <services...>`。
输入服务名可以带 `.service` 后缀；`all` 和 `*` 必须单独使用。`all` / `*` 默认只选择 enabled 服务，`--disabled` 选择全部服务，`--no-enabled` 选择 disabled 服务；显式服务名或 glob 会按匹配结果执行，不受 enabled 默认过滤隐藏。

### `start`

启动服务。

```bash
otter service start <services...>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |
| `--reload` | 启动前执行 daemon-reload |
| `--stop-after <duration>` | 指定时间后自动停止服务 |
| `-t, --trace` | 启动成功后展示日志，仅限单个服务 |

行为：

- `--trace` 匹配多个服务时，不会自动进入日志跟随；会输出黄色 warning，并逐个给出可手工执行的 `otter service log -f <service>` 命令。

### `stop`

停止服务。

```bash
otter service stop <services...>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |

### `restart`

重启服务。

```bash
otter service restart <services...>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |
| `--reload` | 重启前执行 daemon-reload |
| `-t, --trace` | 启动成功后展示日志，仅限单个服务 |

行为：

- `--trace` 匹配多个服务时，不会自动进入日志跟随；会输出黄色 warning，并逐个给出可手工执行的 `otter service log -f <service>` 命令。

### `reload`

重载服务。

```bash
otter service reload <services...>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |

### `enable`

启用服务。

```bash
otter service enable <services...>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |
| `--start` | enable 后启动服务 |

`--start` 会在 `systemctl enable <services...>` 成功后继续执行 `systemctl start <services...>`。

### `disable`

禁用服务。

```bash
otter service disable <services...>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-E, --no-enabled` | 不包含启用的服务列表 |
| `-d, --disabled` | 包含未启用的服务列表 |
| `--stop` | disable 后停止服务 |

`--stop` 会在 `systemctl disable <services...>` 成功后继续执行 `systemctl stop <services...>`。

### `daemon-reload`

重载系统 service 树。

```bash
otter service daemon-reload
```

行为：

- 不接受参数。
- 等价于前台执行 `systemctl daemon-reload`。
- 执行前会输出实际执行的 `systemctl` 命令。

## 服务组命令

### `group-list`

列出服务组。

```bash
otter service group-list
otter service list-group
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-1, --one` | 在一行中展示 |
| `-s, --services` | 同时展示包含的服务名称 |

行为：

- 默认按组名稳定升序逐行输出。
- `--one` 按组名升序输出为空格分隔的一行。
- `--services` 优先于 `--one`，输出 `<group>: <service1>, <service2>`，服务名同样稳定升序。
- 服务组来自 service 文件 `[X-Otter]` 段中的 `Group` 配置，多个组可用逗号分隔，也可写多行 `Group`。

### `group-start`

启动一组服务。

```bash
otter service group-start <groups...>
otter service start-group <groups...>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `--stop-after <duration>` | 指定时间后自动停止服务 |

行为：

- 必须至少传入一个组名。
- 组名来自 service 文件 `[X-Otter]` 段中的 `Group` 配置，与 `group-list` 使用同一套解析规则。
- 多个组按输入顺序展开，重复服务只保留第一次出现的结果。
- 展开后等价于执行 `otter service start <services...>`；`--stop-after` 会继续传递给 start 逻辑。
- 组不存在时返回 `cannot get services from group: group <name> is not exist`。

### `group-stop`

停止一组服务。

```bash
otter service group-stop <groups...>
otter service stop-group <groups...>
```

行为：

- 与 `group-start` 使用相同的组解析、按输入顺序展开和去重规则。
- 展开后等价于执行 `otter service stop <services...>`。
- 组不存在时返回 `cannot get services from group: group <name> is not exist`。

### `group-restart`

重启一组服务。

```bash
otter service group-restart <groups...>
otter service restart-group <groups...>
```

行为：

- 与 `group-start` 使用相同的组解析、按输入顺序展开和去重规则。
- 展开后等价于执行 `otter service restart <services...>`。
- 组不存在时返回 `cannot get services from group: group <name> is not exist`。

## 安装与接管命令

### `install-service`

安装一个 service 文件。

```bash
otter service install-service [选项] <file>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-n, --name <name>` | 服务名，默认使用文件名 |
| `-E, --no-enable` | 不 enable 服务 |
| `-S, --no-start` | 不 start 服务 |

行为：

- 未指定 `--name` 时使用 service 文件名，并裁剪 `.service` 后缀。
- 覆盖写入 classic service 管理路径，创建 systemd service symlink 和 drop-in 目录，然后执行 `systemctl daemon-reload`。
- 默认继续执行 `systemctl enable <service>` 和 `systemctl start <service>`；`--no-enable` / `--no-start` 可分别跳过。
- 安装成功后输出 `Service <service> install success`，后续 systemctl 动作会按执行顺序回显。

### `install-command`

将命令生成 service 并安装。

```bash
otter service install-command -n <service_name> -- <command...>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-n, --name <name>` | 服务名 |
| `-w, --wd <dir>` | 工作目录，`-` 表示当前目录 |
| `-I, --no-install` | 仅展示生成的服务，不安装 |
| `-E, --no-enable` | 不 enable 服务 |
| `-S, --no-start` | 不 start 服务 |

行为：

- `--name` 必填，命令参数必填。
- 只传一个包含空格的命令字符串时，会按 shell 参数规则拆分。
- 命令路径为相对路径时转为绝对路径；`--wd -` 使用当前工作目录。
- `--no-install` 只输出生成的 service 内容，不写文件、不执行 systemctl。

### `exp-install-docker-compose`

安装 docker-compose 文件为 service。

```bash
otter service exp-install-docker-compose [选项] [file]
otter service install-docker-compose [选项] [file]
otter service idc [选项] [file]
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-n, --name <name>` | 服务名 |
| `-d, --dir <dir>` | 服务基础路径，默认 `/opt/<服务名>` |
| `-f, --force` | 强制覆盖基础路径中已存在的 `docker-compose.yml` |
| `-E, --no-enable` | 不 enable 服务 |
| `-S, --no-start` | 不 start 服务 |

行为：

- 未传参数时使用当前目录下的 `docker-compose.yml`。
- 参数可以是目录或 compose 文件；未指定 `--name` 时使用 compose 文件所在目录名。
- `--dir` 默认为 `/opt/<服务名>`；compose 文件会复制为 `<dir>/docker-compose.yml`，已有文件默认报错，`--force` 才覆盖。
- 生成的 service 使用 `docker compose -p %N up --remove-orphans`，并写入 `[X-Otter] DockerComposeBaseDir` 元信息。

### `link-service`

将当前已有 systemd service 纳入 otter service 管理。

```bash
otter service link-service <service>
otter service link <service>
otter service fake <service>
```

别名：

- `link`
- `install-fake`
- `fake`
- `install-fake-service`

行为：

- 输入可带 `.service` 后缀。
- `otter-core` 必须由 otter 外部管理，不能接管。
- 如果 classic 管理路径中已存在同名 service，会返回已安装/已接管错误。
- 通过 systemd 查询到已有 unit 的 `FragmentPath` 后，在 classic 管理路径创建 symlink，不改写原始 unit。
- 接管成功后输出 `Create fake service (linked to <path>) success.`。

## 编辑与刷新命令

### `edit`

编辑服务。

```bash
otter service edit <service>
```

行为：

- classic service 会用 `$EDITOR`、`vim`、`vi` 或 `nano` 前台打开 service 文件，保存后执行 `systemctl daemon-reload`。
- package service 会复制源 `SERVICE` 和 `ENV` 到临时文件，按交互选项编辑、保存、重新生成 service。
- 有变更时会询问是否重启服务；确认后执行 `systemctl restart <service>`。
- package service 保存、跳过保存、重新生成和重启确认结果会输出统一提示。

### `re-generate`

刷新指定 service 文件配置。该命令为隐藏命令。

```bash
otter service re-generate [选项] <service>
otter service regen [选项] <service>
```

选项：

| 选项 | 说明 |
| --- | --- |
| `-r, --restart` | regenerate 后重启服务 |
| `-R, --not-restart` | regenerate 后不重启服务 |

行为：

- 输入可带 `.service` 后缀。
- 仅对 package service 生效，重新从 package 源 `SERVICE` / `ENV` 生成 installed service。
- `--restart` 会在 regenerate 后重启；`--not-restart` 明确不重启；两者都不传时会交互确认。若两者同时传入，以 `--restart` 为准。
- 重新生成、重启/不重启选择和完成结果会输出统一提示。

## 隐藏维护命令

以下命令为维护入口，默认不在普通帮助中展示：

| 命令 | 别名 | 说明 |
| --- | --- | --- |
| `audit` | 无 | systemd hook 审计 |
| `self-check` | `check` | 自检 |
| `install` | 无 | 安装依赖服务 |
| `upsert-self` | `install-self`, `self-install`, `self-update`, `update-self`, `us` | 安装或更新当前节点 |
| `upsert-cluster` | `uc`, `i-c`, `ii-c`, `i-cluster`, `install-cluster`, `update-cluster` | 安装或更新集群 |

`self-check` 会以 `otter-self-check` 作为目标子程序名重新执行当前二进制，并透传后续参数。
`install` 会以 `otter-install` 作为目标子程序名重新执行当前二进制，并透传后续参数。
`upsert-self` 和 `upsert-cluster` 会分别以 `otter-upsert-self`、`otter-upsert-cluster` 作为目标子程序名重进当前二进制，并设置 `AS=<子程序名>`。

`audit` 选项：

| 选项 | 说明 |
| --- | --- |
| `-s, --service-name <name>` | service name |
| `-a, --action-name <name>` | action name |

行为：

- 写入 JSONL 审计记录，默认路径为 `/etc/otter/otter-core-audit.log`。
- 缺少 `--service-name` 或 `--action-name` 时，输出红色提示 `Maybe you want to use \`otter-audit\`?`，不写审计记录。
- `OTTER_AUDIT_BYPASS=-1` 时跳过审计；`OTTER_AUDIT_BYPASS=1` 时审计写入失败不阻断调用方。

## 服务匹配规则

- 用户输入可以带 `.service` 后缀，内部统一裁剪。
- `all` 和 `*` 是特殊服务模式。
- 如果使用 `all` 或 `*`，必须是唯一服务 pattern。
- 每个普通 glob pattern 必须至少匹配一个已知服务。

动作类命令的默认过滤：

- 不传 glob 或使用 `all`、`*` 时，默认只选择 enabled 服务。
- `--disabled` 选择所有服务。
- `--no-enabled` 选择 disabled 服务。
- `--disabled --no-enabled` 仍选择 disabled 服务。

列表类命令的默认过滤：

- `status`、`list`、`detail` 默认选择 enabled 或 running 服务。
- `--disabled` 选择所有服务。
- `--no-enabled` 选择 disabled 服务。
- `--disabled --no-enabled` 仍选择 disabled 服务。

## 路径与命名约定

新实现统一使用 `otter` 命名：

| 用途 | 路径或名称 |
| --- | --- |
| 配置文件 | `/etc/otter/.config` |
| machine id | `/etc/otter/machine-id` |
| service db | `/etc/otter/otter-service.db` |
| classic services | `/etc/otter/services` |
| package services | `/etc/otter/services/.do-not-edit` |
| package data | `/data/.otter/otter-packages` |
| core socket | `/var/run/otter-core.socket` |
| core unit | `otter-core.service` |
| systemd 扩展段 | `[X-Otter]` |

新写入内容必须使用 `[X-Otter]`。

## 当前实现状态

当前仓库已经具备 `otter service` 命令骨架、完整子命令/flag/alias 契约、非 Linux 平台门禁、otter 路径配置基础包，以及 Linux 下真实 systemd、安装、编辑和审计等业务 handler。
