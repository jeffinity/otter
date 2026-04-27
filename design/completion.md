# `otter completion` 核心实现设计

本文档描述 `otter completion` 与 `otter config-completion` 的核心实现设计。用户可见用法维护在 `doc/completion.md`。

## 目标

`otter` 在可执行文件侧直接提供 Tab 补全能力：

- `completion` 输出 Shell 补全脚本。
- `config-completion` 在 Linux 上把补全脚本安装到常见补全目录。
- 子命令、flag 名称和 Cobra 已知的参数候选由 Cobra completion 机制提供。
- 动态参数补全通过各命令的 `ValidArgsFunction` 或 flag completion 函数扩展。

## 模块边界

| 位置 | 职责 |
| --- | --- |
| `cmd/completion.go` | 顶层补全命令、脚本生成、Linux 安装路径选择 |
| `cmd/main.go` | 注册 `completion` 与 `config-completion` |
| 各子命令文件 | 按需提供 `ValidArgsFunction` 或 `RegisterFlagCompletionFunc` |

补全命令不引入新的业务层。当前逻辑足够小，保留在 `cmd/`；如果后续增加复杂配置写入、profile 修改或发行版探测，再拆入 `internal/completionconfig`。

## `completion` 命令

命令：

```text
otter completion [bash|zsh|fish|powershell]
```

实现使用 Cobra 内建生成函数：

| Shell | 函数 |
| --- | --- |
| `bash` | `root.GenBashCompletion` |
| `zsh` | `root.GenZshCompletion` |
| `fish` | `root.GenFishCompletion` |
| `powershell` | `root.GenPowerShellCompletion` |

脚本写到 stdout，不写文件，不检查平台。该命令可在 Linux、macOS 或其他可运行 `otter` 的平台上使用。

## `config-completion` 命令

命令：

```text
otter config-completion [bash|zsh|fish]
```

设计约束：

- 仅支持 Linux，非 Linux 返回明确错误。
- 不传 shell 时从 `$SHELL` 识别 `bash`、`zsh`、`fish`。
- 默认写用户级目录，避免要求 sudo。
- `--system` 写常见系统级目录。
- `--dir` 允许用户、打包脚本或特殊发行版显式指定目录。
- 不自动修改 `.bashrc`、`.zshrc` 或 fish 配置，避免不可逆地改动用户 profile。

安装流程：

1. 校验平台为 Linux。
2. 解析或自动识别 Shell。
3. 解析 home 目录和安装目录。
4. 通过 Cobra 生成脚本到内存。
5. 创建安装目录。
6. 写入脚本文件。
7. 输出安装路径和生效提示。

## 路径策略

用户级默认路径：

| Shell | 目录 | 文件名 |
| --- | --- | --- |
| `bash` | `~/.local/share/bash-completion/completions` | `otter` |
| `zsh` | `~/.local/share/zsh/site-functions` | `_otter` |
| `fish` | `~/.config/fish/completions` | `otter.fish` |

系统级路径：

| Shell | 目录 | 文件名 |
| --- | --- | --- |
| `bash` | `/etc/bash_completion.d` | `otter` |
| `zsh` | `/usr/local/share/zsh/site-functions` | `_otter` |
| `fish` | `/etc/fish/completions` | `otter.fish` |

## Linux 发行版判断

当前不需要按发行版分支。

原因：

- 补全脚本内容由 Shell 类型决定，不由发行版决定。
- 安装差异主要是补全目录是否被 Shell 或系统配置加载。
- 用户级目录比发行版专有目录更稳定，且不需要 root。
- 系统级目录在不同发行版确实可能不同，所以提供 `--dir` 显式覆盖。

如果未来要支持包管理器安装流程，可以在打包脚本中使用 `--dir` 写入发行版指定目录，而不是把发行版判断固化进 `otter`。

## 动态补全扩展

当前命令可自动补全：

- 顶层子命令。
- 子命令。
- flag 名称。
- `completion` / `config-completion` 的 shell 参数。
- `config-completion --dir` 的目录路径。

后续应按命令补充：

- `otter service`：服务名、服务组名通过 `ValidArgsFunction` 动态补全。
- `otter new --repo`、`--output`：目录补全。
- 需要补全 flag value 时使用 `RegisterFlagCompletionFunc`。

动态补全函数必须足够快，并在外部系统不可用时优雅返回空候选，不能让 Tab 补全长时间阻塞。

## 错误语义

错误应保持可修复：

- 未传 `completion` shell：提示支持的 shell。
- 不支持的 shell：列出支持值。
- 非 Linux 执行 `config-completion`：返回 `otter config-completion is only supported on linux`。
- 自动识别 shell 失败：要求显式传入 `bash`、`zsh` 或 `fish`。
- 创建目录或写文件失败：保留底层错误，用户可据此判断权限或路径问题。
- `bash` 用户级安装成功提示会明确要求在当前 shell 中 `source` 补全文件，并说明不要直接执行补全文件或使用 `bash <file>`。

## 测试策略

测试覆盖：

- `completion bash` 输出包含 Cobra bash entrypoint。
- `completion` 输出包含新顶层命令。
- `config-completion bash` 写入用户级默认路径。
- `config-completion --dir` 配合自动 Shell 识别写入指定目录。
- 非 Linux 执行 `config-completion` 返回明确错误。

后续新增 shell、路径策略或 profile 修改能力时，必须补充测试，并同步更新 `doc/completion.md` 和本文档。
