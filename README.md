# otter

[![Go Reference](https://pkg.go.dev/badge/github.com/jeffinity/otter.svg)](https://pkg.go.dev/github.com/jeffinity/otter)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jeffinity/otter)](https://github.com/jeffinity/otter/blob/main/go.mod)
[![License](https://img.shields.io/github/license/jeffinity/otter)](https://github.com/jeffinity/otter/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/jeffinity/otter)](https://goreportcard.com/report/github.com/jeffinity/otter)
[![Test Status](https://github.com/jeffinity/otter/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/jeffinity/otter/actions/workflows/ci.yml?query=branch%3Amain)
[![codecov](https://codecov.io/gh/jeffinity/otter/branch/main/graph/badge.svg)](https://codecov.io/gh/jeffinity/otter)

`otter` 是一个用于生成应用脚手架的 CLI，默认基于 [`app-layout`](https://github.com/jeffinity/app-layout) 模板创建项目，并支持单仓与大仓两种组织模式。

## 核心能力

### `new`

- 使用方式 1：单仓模式（默认）
  - 命令：`otter new <模块路径> <应用名>`
  - 示例：`otter new github.com/acme/order order-api`
- 使用方式 2：大仓模式建仓
  - 命令：`otter new -m <模块路径> <应用名>`
  - 示例：`otter new -m github.com/acme/mono order-api`
- 使用方式 3：大仓模式新增应用
  - 命令：`otter new -m <应用名>`
  - 示例：`otter new -m order-api`

参数说明：
- `-m, --mono`：启用大仓模式
- `-o, --output`：输出基目录
- `-r, --repo`：layout 源（支持 git 地址或本地目录）

## 环境要求

- Go `1.25.5+`
- Git
- [Task](https://taskfile.dev/)

## 快速开始

```bash
git clone <your-repo-url>
cd otter
task check
task build
```

构建产物默认在：

```text
target/otter/<os>/<arch>/otter
```

## 使用说明（按子命令）

### `new`

用于创建新项目脚手架，支持三条主要路线：

1. 单仓模式（默认）

```bash
otter new github.com/acme/order order-api
```

输出目录：`./order-api`

2. 大仓模式建仓（指定模块路径 + 应用名）

```bash
otter new -m github.com/acme/mono order-api
```

输出目录：`./order-api`  
应用目录：`./order-api/app/order-api`

3. 大仓模式建 app（仅应用名）

```bash
otter new -m order-api
```

要求输出目录已存在 `app/` 与有效 `go.mod`。  
输出目录：`./app/order-api`

常用参数：
- `-r, --repo`：layout 源（git 地址或本地目录）
- `-o, --output`：输出基目录
- `-m, --mono`：启用大仓模式

### `version`

显示构建信息与版本号。

```bash
otter version
```

后续新增子命令将继续按该结构补充文档。

## 开发命令

```bash
task check            # 环境与工具检查
task lint             # 代码检查
task conf -- .        # 生成配置 protobuf
task wire -- .        # 生成 wire
task build -- -f      # 本机构建
task build-linux-amd64
task deploy -- <app>
```
