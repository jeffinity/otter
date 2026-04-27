# `otter service`

`otter service` 是服务管理命令入口，用于承载原 `ambot-service` 的服务查询、启停、安装、编辑、审计和集群模式能力。

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

## 全局选项

| 选项 | 说明 |
| --- | --- |
| `-c, --cluster` | 集群模式，仅对支持集群模式的命令生效 |

集群模式允许的命令：

- `daemon-reload`
- `start`
- `stop`
- `restart`
- `reload`
- `enable`
- `disable`
- `status`
- `detail`
- `group-start`
- `group-stop`
- `group-restart`
- `group-list`

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

### `view`

展示服务文件。若服务文件包含 docker compose 元信息，会附带展示 compose 文件内容。

```bash
otter service view <service>
```

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

## 服务动作命令

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

### `daemon-reload`

重载系统 service 树。

```bash
otter service daemon-reload
```

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

### `group-stop`

停止一组服务。

```bash
otter service group-stop <groups...>
otter service stop-group <groups...>
```

### `group-restart`

重启一组服务。

```bash
otter service group-restart <groups...>
otter service restart-group <groups...>
```

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

## 编辑与刷新命令

### `edit`

编辑服务。

```bash
otter service edit <service>
```

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

## 隐藏维护命令

以下命令为维护入口，默认不在普通帮助中展示：

| 命令 | 别名 | 说明 |
| --- | --- | --- |
| `audit` | 无 | systemd hook 审计 |
| `self-check` | `check` | 自检 |
| `install` | 无 | 安装依赖服务 |
| `upsert-self` | `install-self`, `self-install`, `self-update`, `update-self`, `us` | 安装或更新当前节点 |
| `upsert-cluster` | `uc`, `i-c`, `ii-c`, `i-cluster`, `install-cluster`, `update-cluster` | 安装或更新集群 |

`audit` 选项：

| 选项 | 说明 |
| --- | --- |
| `-s, --service-name <name>` | service name |
| `-a, --action-name <name>` | action name |

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

兼容读取旧 `[X-Ambot]` metadata 可以作为过渡能力，但新写入内容必须使用 `[X-Otter]`。

## 当前迁移状态

当前仓库已经具备 `otter service` 命令骨架、完整子命令/flag/alias 契约、非 Linux 平台门禁、otter 路径配置基础包。Linux 下真实 systemd、gRPC、安装、编辑、审计等业务 handler 仍在迁移中。
