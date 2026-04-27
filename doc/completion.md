# `otter completion` 与 `otter config-completion`

`otter completion` 用于生成 Shell 补全脚本。`otter config-completion` 用于在 Linux 上将补全脚本安装到常见补全目录，方便用户快速启用 Tab 补全。

补全能力由 `otter` 可执行文件自身提供，不需要额外生成代码。

## `completion`

生成指定 Shell 的补全脚本，并输出到 stdout。

```bash
otter completion [bash|zsh|fish|powershell]
```

支持的 Shell：

| Shell | 说明 |
| --- | --- |
| `bash` | 生成 bash completion 脚本 |
| `zsh` | 生成 zsh completion 脚本 |
| `fish` | 生成 fish completion 脚本 |
| `powershell` | 生成 PowerShell completion 脚本 |

示例：

```bash
source <(otter completion bash)
source <(otter completion zsh)
otter completion fish > ~/.config/fish/completions/otter.fish
otter completion powershell
```

## `config-completion`

在 Linux 上安装补全脚本。

```bash
otter config-completion [bash|zsh|fish]
```

不传 Shell 时，会根据 `$SHELL` 自动识别 `bash`、`zsh` 或 `fish`。无法识别时需要显式传入 Shell 名称。

选项：

| 选项 | 说明 |
| --- | --- |
| `--dir <dir>` | 指定补全脚本安装目录 |
| `--system` | 安装到常见系统级补全目录，通常需要 root 权限 |

默认用户级安装目录：

| Shell | 路径 | 文件名 |
| --- | --- | --- |
| `bash` | `~/.local/share/bash-completion/completions` | `otter` |
| `zsh` | `~/.local/share/zsh/site-functions` | `_otter` |
| `fish` | `~/.config/fish/completions` | `otter.fish` |

`--system` 安装目录：

| Shell | 路径 | 文件名 |
| --- | --- | --- |
| `bash` | `/etc/bash_completion.d` | `otter` |
| `zsh` | `/usr/local/share/zsh/site-functions` | `_otter` |
| `fish` | `/etc/fish/completions` | `otter.fish` |

示例：

```bash
otter config-completion bash
otter config-completion zsh
otter config-completion fish
sudo otter config-completion --system bash
otter config-completion --dir /tmp/completions zsh
```

## Linux 发行版差异

当前实现不按 Ubuntu、Debian、CentOS、Fedora 等发行版做分支。补全安装的主要差异来自 Shell 和补全目录，而不是发行版名称。

默认策略：

- 普通安装使用用户级目录，避免默认要求 sudo。
- `--system` 使用主流 Linux 上常见的系统级目录。
- 特殊发行版、企业镜像或打包场景可通过 `--dir` 显式指定目录。

## 生效方式

安装后通常重新打开 Shell 即可生效。

额外注意：

- `bash` 补全文件需要被当前交互式 shell 加载。立即生效可执行 `source ~/.local/share/bash-completion/completions/otter`；不要直接执行该文件，也不要用 `bash ~/.local/share/bash-completion/completions/otter`，那只会在子 shell 中注册补全并随子进程退出失效。
- `bash` 自动加载用户级目录依赖系统已启用 `bash-completion`。若重开 shell 后仍未生效，先确认系统已安装并加载 `bash-completion`。
- `zsh` 需要安装目录在 `fpath` 中，并在 `compinit` 前配置。
- `fish` 通常会自动读取用户级 completions 目录。

## 验证

修改补全命令相关行为后，至少执行：

```bash
go test ./cmd
```

如遇默认 Go build cache 权限问题，使用：

```bash
GOCACHE=$(pwd)/.build/go-cache go test ./cmd
```
