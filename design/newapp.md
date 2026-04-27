# `otter new` 核心实现设计

本文档描述 `otter new` / `internal/newapp` 的核心实现边界、生成流程、副作用和测试策略。用户可见的命令用法维护在 `doc/new.md`。

## 目标

`otter new` 基于 `app-layout` 模板生成 Go 应用工程，支持三种创建路线：

| 路线 | 命令形态 | 目标 |
| --- | --- | --- |
| 单仓模式 | `otter new <module> <app>` | 创建一个以应用为根目录的独立 Go module |
| 大仓建仓模式 | `otter new -m <module> <app>` | 复制完整 layout，并将 `app/app_layout` 重命名为 `app/<app>` |
| 大仓新增应用模式 | `otter new -m <app>` | 在现有大仓 `app/` 下新增 `app/<app>` |

实现必须保持 CLI 层薄、生成逻辑集中在 `internal/newapp`。命令层负责参数个数、空值、应用名格式等输入边界校验；领域层负责路线选择、模板获取、复制、目录变换、文本替换和后置任务。

## 模块边界

| 位置 | 职责 |
| --- | --- |
| `cmd/new.go` | Cobra 命令定义、参数解析、flag 绑定、帮助输出、调用 `newapp.Run` |
| `internal/newapp/newapp.go` | 模板获取、路线解析、文件复制、目录调整、token 替换、后置任务执行 |
| `pkg/logx` | 进度和错误日志 |
| `pkg/tuix` | 自定义 usage/help 渲染 |

`internal/newapp` 不对外承诺稳定 API。新增可复用能力优先保持包内私有，只有命令层必须调用的入口保留为公开函数或类型。

## 命令层设计

`CmdNew` 绑定以下选项：

| flag | 默认值 | 说明 |
| --- | --- | --- |
| `-r, --repo` | `newapp.DefaultLayoutRepo` | 模板源，支持 git 地址或本地目录 |
| `-o, --output` | `.` | 输出基目录 |
| `-m, --mono` | `false` | 启用大仓模式 |

参数解析分两步：

1. `parseNewArgs(monoRepo, args)` 只处理不同模式下的参数数量。
2. `parseAndValidateArgs(monoRepo, args)` 处理边界校验：
   - 单仓模式必须是 `<模块路径> <应用名>`。
   - 大仓模式允许 `<模块路径> <应用名>` 或 `<应用名>`。
   - `<应用名>` 不能为空，不能包含空白字符、`/` 或 `\`。
   - 只有传入 `<模块路径>` 时才校验其非空白。

命令层不检查模板目录结构、输出目录状态或 Go module 探测结果，这些属于领域流程。

## 领域入口

`newapp.Run(modulePath, appName string, opts Options) error` 是当前核心入口。

`Options`：

| 字段 | 说明 |
| --- | --- |
| `LayoutSource` | 模板源；空值回退到 `DefaultLayoutRepo` |
| `OutputDir` | 输出基目录；空值回退到 `.` |
| `MonoRepo` | 是否启用大仓模式 |
| `SkipPostTasks` | 测试或特殊调用时跳过单仓 `task conf` / `task wire -- .` |

入口流程：

1. 归一化 `OutputDir` 和 `LayoutSource`。
2. 将输出目录转为绝对路径。
3. 调用 `resolveCreationRoute` 选择创建路线。
4. 调用 `fetchLayout` 获取模板到临时目录。
5. 校验模板应用目录 `app/app_layout` 存在。
6. 调用 `executeRoute` 执行复制、目录变换、文本替换与后置任务。
7. 清理临时模板目录。

## 创建路线

路线由 `creationRoute` 描述，避免在执行阶段散落模式判断。

| 字段 | 说明 |
| --- | --- |
| `kind` | 路线类型，便于测试和后续分支扩展 |
| `label` | 日志和错误上下文 |
| `targetDir` | 最终写入目录 |
| `modulePath` | 替换后的 Go module path |
| `copyStrategy` | 复制 layout 根目录或仅复制模板应用目录 |
| `flattenAppLayout` | 是否将 `app/app_layout` 内容搬到项目根 |
| `renameTemplateDir` | 是否将 `app/app_layout` 改名为 `app/<app>` |
| `rewriteSingleImport` | 单仓模式是否清理 Go import 中的 `/app/<app>/` |
| `runPostTasks` | 是否执行单仓后置生成 |
| `mustNotExist` | 目标目录是否必须不存在 |

### 单仓模式

`resolveCreationRoute` 生成：

- `targetDir = <output>/<app>`
- `copyStrategy = copyFromLayoutRoot`
- `flattenAppLayout = true`
- `rewriteSingleImport = true`
- `runPostTasks = !SkipPostTasks`

执行结果：

1. 复制完整 layout 到目标应用目录。
2. 将 `app/app_layout/*` 移到项目根。
3. 删除剩余 `app/` 目录。
4. 替换 `app_layout`、模板 module path 和单仓 import 前缀。
5. 执行 `task conf` 与 `task wire -- .`。

单仓模式允许输出基目录非空，但目标应用目录必须不存在或为空。

### 大仓建仓模式

触发条件：`MonoRepo=true` 且传入 `modulePath`。

`resolveCreationRoute` 生成：

- `targetDir = <output>/<app>`
- `copyStrategy = copyFromLayoutRoot`
- `renameTemplateDir = true`
- `mustNotExist = true`

执行结果：

1. 复制完整 layout 到 `<output>/<app>`。
2. 将 `app/app_layout` 重命名为 `app/<app>`。
3. 替换模板 token。

该路线不执行 `task conf` / `task wire`，避免在未进入目标仓上下文时产生不明确副作用。

### 大仓新增应用模式

触发条件：`MonoRepo=true` 且只传入应用名。

`resolveCreationRoute` 要求：

- `<output>/app` 必须存在。
- 在 `<output>` 下执行 `go list -m` 自动识别 module path。

执行结果：

1. 仅复制模板应用目录 `app/app_layout` 到 `<output>/app/<app>`。
2. 替换模板 token。

该路线要求目标应用目录不存在，避免覆盖现有应用。

## 模板获取

`fetchLayout(layoutSource)` 总是将模板复制或克隆到临时目录，并返回 `cleanup`：

- 当 `layoutSource` 是本地目录时，递归复制目录。
- 否则要求本机存在 `git`，执行浅克隆：
  - `git clone --depth=1 --recurse-submodules --shallow-submodules`
  - 使用 `-c advice.detachedHead=false` 降低 clone 噪声。

这样可以保证后续复制逻辑只面对普通本地目录，并且不会直接修改源模板。

## 文件复制与目录变换

`copyDir` 使用 `filepath.WalkDir`：

- 跳过 `.git` 目录。
- 跳过 submodule metadata 形式的 `.git` 文件。
- 普通文件保留源文件 mode。
- 符号链接按链接目标重新创建。

目录变换：

- `flattenAppLayoutToRoot`：移动 `app/app_layout` 内容到目标根目录，并删除 `app/`。
- `renameTemplateAppDir`：将 `app/app_layout` 改名为 `app/<app>`。
- `moveDirContents`：如果目标路径已存在，直接返回冲突错误。

## Token 替换

`replaceTemplateTokens(rootDir, modulePath, appName)` 的替换顺序固定：

1. 在 `.proto` 文件中将 `app_layout` 替换为安全名，即把应用名中的 `-` 替换为 `_`。
2. 在所有文本文件中将 `app_layout` 替换为原始应用名。
3. 在 `.go` 文件中将模板 module path `github.com/jeffinity/app-layout` 替换为目标 module path。
4. 如果存在 `go.mod` 或 `.golangci.yml`，再对这两个文件做 module path 替换。

替换函数必须跳过二进制文件和非普通文件。后续新增替换规则时，应优先通过 matcher 限定文件范围，避免误改生成产物、图片或二进制资源。

单仓模式额外执行：

```text
<module>/app/<app>/ -> <module>/
```

该步骤仅作用于 `.go` 文件，用于消除模板在大仓结构中的 import 前缀。

## 外部命令与副作用

当前涉及外部命令：

| 命令 | 场景 | 失败语义 |
| --- | --- | --- |
| `git clone` | 远程模板源 | 返回 clone 错误和 stderr/stdout 摘要 |
| `go list -m` | 大仓新增应用自动识别 module path | 返回识别失败，包含命令输出 |
| `task conf` | 单仓后置生成 | 返回 task 失败和输出 |
| `task wire -- .` | 单仓后置生成 | 返回 task 失败和输出 |

测试不得依赖真实远程网络。需要覆盖外部命令时使用本地 git 仓库、fake executable 或临时 PATH。

## 错误与日志

错误信息应指向用户可修复的输入或环境问题，例如：

- 模板目录不存在。
- 输出路径不是目录。
- 目标目录非空或已存在。
- 大仓新增应用时缺少 `app/`。
- 未安装 `git` 或 `task`。
- `go list -m` 返回为空或失败。

日志通过 `pkg/logx` 输出关键步骤，包含路线、目标目录、模板源和耗时。不要在 `internal/newapp` 中直接向 stdout 打印面向用户的表格或帮助内容。

## 测试策略

已有测试重点覆盖：

- 三条创建路线的正常路径。
- 单仓目标目录非空、路径冲突、模板目录缺失。
- 本地模板、`file://` git 模板和 git 不可用。
- `.git` 跳过、符号链接复制、二进制文件跳过。
- token 替换 matcher、无匹配替换、错误路径。
- 单仓后置任务通过 fake `task` 验证调用顺序。

后续修改规则：

- 修改参数契约时，补充 `cmd` 层测试。
- 修改路线选择或目录变换时，补充 `internal/newapp` 表驱动测试。
- 修改外部命令调用时，用 fake executable 或本地临时仓库覆盖，不依赖真实机器状态。
- 修改用户可见行为时，同步 `doc/new.md`。
- 修改核心流程、错误语义或副作用时，同步本文档。

## 设计约束

- 不覆盖非空目标目录。
- 不修改模板源目录。
- 不手工编辑生成文件；需要生成时通过 Taskfile 流程。
- 不把参数校验下沉到文件复制或系统调用层重复实现。
- 不在测试中写入系统目录。
- 新增复杂逻辑优先拆分为短函数，保持 `internal/newapp/newapp.go` 在 800 行以内；接近阈值时按职责拆分文件。
