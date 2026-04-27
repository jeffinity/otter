# `otter new`

`otter new` 用于基于 app-layout 模板创建 Go 应用项目，支持单仓模式和大仓模式。

## 用法

单仓模式：

```bash
otter new [选项] <模块路径> <应用名>
```

大仓模式，创建独立仓：

```bash
otter new -m [选项] <模块路径> <应用名>
```

大仓模式，向现有 `app/` 目录新增应用：

```bash
otter new -m [选项] <应用名>
```

## 参数

| 参数 | 说明 |
| --- | --- |
| `<模块路径>` | Go module path，例如 `github.com/acme/order` |
| `<应用名>` | 应用目录名，不能包含空白字符或路径分隔符 |

## 选项

| 选项 | 默认值 | 说明 |
| --- | --- | --- |
| `-r, --repo` | `github.com/jeffinity/app-layout` | layout 源，支持 git 地址或本地目录 |
| `-o, --output` | `.` | 输出基目录 |
| `-m, --mono` | `false` | 启用大仓模式 |

## 使用方式

### 单仓模式

```bash
otter new github.com/acme/order order-api
```

输出目录：

```text
./order-api
```

### 大仓模式创建独立仓

```bash
otter new -m -o /tmp/work github.com/acme/mono order-api
```

输出目录：

```text
/tmp/work/order-api
```

应用目录：

```text
/tmp/work/order-api/app/order-api
```

### 大仓模式向现有仓库新增应用

```bash
otter new -m order-api
```

要求：

- 当前输出目录存在 `app/`。
- 当前输出目录或上级目录存在有效 `go.mod`。

输出目录：

```text
./app/order-api
```

### 指定 layout 源

使用远程 git 源：

```bash
otter new -r https://github.com/acme/app-layout github.com/acme/order order-api
```

使用本地 layout 目录：

```bash
otter new -r /Users/jeff/tao/workspace/app-layout github.com/acme/order order-api
```

## 行为约束

- 单仓模式必须传入 `<模块路径>` 和 `<应用名>`。
- 大仓模式支持传入 `<模块路径> <应用名>` 或仅传入 `<应用名>`。
- `<应用名>` 不能为空，不能包含空白字符、`/` 或 `\`。
- 当 `--repo` 指向本地目录时，直接使用该目录作为模板源。
- 当 `--repo` 指向 git 地址时，会先克隆到临时目录再生成项目。

## 验证

修改 `new` 子命令相关行为后，至少执行：

```bash
go test ./internal/newapp ./cmd
```

如遇默认 Go build cache 权限问题，使用：

```bash
GOCACHE=$(pwd)/.build/go-cache go test ./internal/newapp ./cmd
```
