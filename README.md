# otter

[![Go Reference](https://pkg.go.dev/badge/github.com/jeffinity/otter.svg)](https://pkg.go.dev/github.com/jeffinity/otter)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jeffinity/otter)](https://github.com/jeffinity/otter/blob/main/go.mod)
[![License](https://img.shields.io/github/license/jeffinity/otter)](https://github.com/jeffinity/otter/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/jeffinity/otter)](https://goreportcard.com/report/github.com/jeffinity/otter)
[![Test Status](https://github.com/jeffinity/otter/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/jeffinity/otter/actions/workflows/ci.yml?query=branch%3Amain)
[![codecov](https://codecov.io/gh/jeffinity/otter/branch/main/graph/badge.svg)](https://codecov.io/gh/jeffinity/otter)

`otter` 是一个面向开发与运维场景的 Go CLI 工具集合。当前仓库提供应用脚手架生成、Shell 补全脚本生成与安装、版本信息展示，并正在迁移服务管理能力。

主要支持平台：

- `new`、`completion`、`version`：Linux 与 macOS。
- `config-completion`：仅 Linux。
- `service`：仅 Linux，非 Linux 运行会直接返回不支持错误。

## 安装与构建

环境要求：

- Go `1.25.5+`
- Git
- [Task](https://taskfile.dev/)

常用命令：

```bash
task check
task lint
task build
task build-linux-amd64
task build-linux-arm64
task deploy -- <host>
```

构建产物默认位于：

```text
target/otter/<os>/<arch>/otter
```

## 子命令

### `new`

基于 app-layout 模板创建新项目，支持单仓模式和大仓模式。

常用示例：

```bash
otter new github.com/acme/order order-api
otter new -m github.com/acme/mono order-api
otter new -m order-api
```

完整说明见 [doc/new.md](doc/new.md)。

### `service`

服务管理命令入口，用于承载原 `ambot-service` 的服务查询、启停、安装、编辑、审计和集群模式能力。该子命令仅支持 Linux。

常用形式：

```bash
otter service status
otter service start <service>
otter service log -f <service>
```

完整说明见 [doc/service.md](doc/service.md)。

### `completion`

生成 Shell 补全脚本，支持 `bash`、`zsh`、`fish` 和 `powershell`。

```bash
otter completion bash
otter completion zsh
```

完整说明见 [doc/completion.md](doc/completion.md)。

### `config-completion`

在 Linux 上安装补全脚本到用户级或系统级补全目录，支持 `bash`、`zsh` 和 `fish`。

```bash
otter config-completion bash
otter config-completion --system zsh
```

完整说明见 [doc/completion.md](doc/completion.md)。

### `version`

显示当前二进制的构建信息、版本、构建时间和提交号。

```bash
otter version
```

## 开发说明

本仓库使用 `AGENTS.md` 约定 AI 代码变更流程、测试要求和文档同步规则。涉及顶层子命令、命令参数、flag、示例或行为变更时，必须同步更新 README 和 `doc/` 下对应子命令文档；涉及核心实现设计时同步更新 `design/`。
