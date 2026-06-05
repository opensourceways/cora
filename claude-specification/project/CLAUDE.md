# CORA 项目 Claude 规范

> 本文件是 CORA 项目的 Claude 工作规范，优先级高于团队规范。
> 与团队规范冲突时以本文件为准。

---

## 团队规范引用

本项目遵循团队基础规范，同时参考：

```
claude-specification/team/docs/standards/
```

---

## 项目基础信息

- **项目名称**：Cora（Community Collaboration CLI）
- **主要语言**：Go 1.22+
- **架构风格**：单体命令行工具（CLI）
- **核心依赖**：cobra（命令框架）、kin-openapi/openapi3（OpenAPI 解析）、viper（配置加载）

---

## 目录结构

```
cora/
├── cmd/cora/main.go                  # 入口：两阶段命令加载、全局 flag 定义
├── internal/
│   ├── auth/resolver.go              # 认证注入（Discourse headers / Etherpad apikey / GitCode token）
│   ├── builder/command.go            # OpenAPI Spec → Cobra 命令树动态生成
│   ├── config/config.go              # 配置加载（viper + .env）
│   ├── executor/executor.go          # HTTP 请求执行
│   ├── log/                          # 统一日志包（叶子依赖，不引用其他 internal 包）
│   │   ├── log.go                    # Logger 核心：Init()、Error/Warn/Info/Debug
│   │   └── mask.go                   # 敏感信息脱敏：MaskURL()、MaskHeader()
│   ├── output/formatter.go           # Table / JSON 输出格式化
│   ├── registry/
│   │   ├── registry.go               # 服务注册表
│   │   └── builtin.go                # 内置服务注册（Etherpad、GitCode）
│   └── spec/
│       ├── loader.go                 # 三段式 Spec 加载（缓存 → 拉取 → 降级）
│       └── cache.go                  # 缓存读写（原子写入）
├── pkg/errs/errors.go                # 统一错误类型与退出码
├── assets/
│   ├── assets.go                     # go:embed 嵌入 Spec 文件
│   └── openapi/                      # 内置服务 OpenAPI Spec
│       ├── etherpad/openapi.json
│       ├── forum/openapi.json
│       └── gitcode/openapi.json
├── spec/                             # 架构与设计文档
│   ├── architecture-design.md
│   ├── logging-design.md             # 日志系统设计（必读）
│   └── *.md
├── config.example.yaml               # 配置文件示例
└── .env.example                      # 本地开发环境变量示例
```

---

## 关键架构约定

### 命令生成机制

- 命令树在运行时从 OpenAPI Spec 动态生成，**不手写具体 API 命令**
- 资源名由三层推导（详见 `spec/spec-optimization-design.md`）：
  1. **Path 优先**：path 中有明确子资源段（如 `/pulls/`、`/issues/`）的，优先采用 path 信号
  2. **Tag 规范化**：对 tag 做归一化（与 path 交叉验证、同义词映射、单复数统一）
  3. **Fallback**：path 最后非参数段
- 动词派生优先级（6 级）：
  1. `Using{METHOD}` 后缀（Etherpad/Spring-Boot 风格）
  2. 已知动词前缀（`list`、`get`、`create`…），GitCode path-encoded opId 跳过此级
  2.5 **新增**：path-encoded opId（GitCode 风格）提取 HTTP method 前缀恢复动词信号
  3. path param 后的动作段：`/{id}/comments` → `comments`
  4. HTTP method + trailing param 推断：GET 无尾参 → `list`，有尾参 → `get`
- Verb==resource 覆盖：verb 和 resource 单复数一致时，重新派生为 `list`/`get`/`create`…
- 同一 resource 下动词冲突时：pathContext → pathSuffix（suffix≠verb）→ HTTP method

### 认证机制

- Discourse：注入 `Api-Key` / `Api-Username` 请求头
- Etherpad：注入 `?apikey=` query 参数
- GitCode：注入 `?access_token=` query 参数
- 认证参数（`Api-Key`、`access_token` 等）不生成 CLI flag，由 executor 自动注入

### 内置服务

内置服务的 OpenAPI Spec 通过 `go:embed` 打包进二进制，无需 `spec_url`：

| 服务 | 命令名 | 默认 API 地址 |
|------|--------|--------------|
| GitCode | `gitcode` | `https://api.gitcode.com` |
| Etherpad | `etherpad` | `https://etherpad.openeuler.org/api/1.3.0` |

### OpenAPI Spec 编写规范

添加新服务或在现有 spec 中新增端点时，遵循以下规范以减少命令生成偏差：

**Tag 规范**
- 使用单数或复数**短名词**，避免多词组合：`Pulls` 而非 `Pull Requests`
- 与 GitHub API 保持一致命名：`pulls` / `issues` / `users` / `repos` / `orgs`
- 每个 operation 只标一个 tag（第一个 tag 用作资源名）
- Tag 含义应与 REST 资源概念一致，**不应跨资源**

**路径与 Tag 的对应关系**
- 路径以 `repos/{o}/{r}/...` 开头 → tag 为 `Repositories`
- 路径含 `repos/{o}/{r}/pulls/...` → tag 为 `Pulls`（不是 Repositories）
- 路径含 `repos/{o}/{r}/issues/...` → tag 为 `Issues`（不是 Repositories）
- 路径不含特定子资源段 → 使用通用资源 tag

**Tag 命名对照表**（GitCode ↔ GitHub 统一）

| 概念 | 推荐 Tag | 避免 |
|------|---------|------|
| PR | `Pulls` | `Pull Requests` |
| Issue | `Issues` | - |
| 仓库 | `Repositories` | `Repos`（GitHub用此名）|
| 组织 | `Organizations` | `Orgs` |
| 用户 | `Users` | - |
| 分支 | `Branch` | `Branches` |
| 标签 | `Labels` | - |
| 里程碑 | `Milestone` | `Milestones` |
| 发布 | `Release` | `Releases` |
| Webhook | `Webhooks` | - |

---

## 日志规范（项目特有）

> 详细设计见 `spec/logging-design.md`，本节为 Claude 工作快速参考。

### 使用 `internal/log` 包

所有日志输出必须通过 `internal/log` 包，**禁止**使用 `fmt.Fprintf(os.Stderr, ...)` 输出到 stderr：

```go
// ❌ 禁止
fmt.Fprintln(os.Stderr, "[warn]", err)
fmt.Fprintf(os.Stderr, "[warn] could not refresh spec...")

// ✅ 正确
log.Warn("could not refresh spec for %q: %v", svcName, err)
log.Debug("→ %s %s", req.Method, log.MaskURL(url))
```

### 日志级别选择

```
log.Error() → 程序无法继续，通常配合 return err（实际上 errs 包已处理大部分）
log.Warn()  → 降级行为，仍能继续运行（使用过期缓存、服务未找到）
log.Info()  → 正常流程的关键节点（spec 加载、缓存命中）—— verbose 时显示
log.Debug() → 细粒度诊断（HTTP 请求/响应、认证注入、.env 加载）—— verbose 时显示
```

### 必须脱敏的内容

```go
// URL 脱敏（log.MaskURL 自动处理 access_token、apikey 等）
log.Debug("→ %s %s", method, log.MaskURL(httpReq.URL.String()))

// Header 脱敏
log.Debug("headers: %v", log.MaskHeader(httpReq.Header))

// 永不记录配置中的 key 值
log.Info("config loaded from %s", path)   // ✅ 只记录路径
log.Debug("api_key: %s", cfg.APIKey)      // ❌ 禁止
```

### 响应 Body 记录规则

- verbose 模式下**必须记录**响应 body（是排查 API 异常的关键信息）
- 超过 2048 字节时截断，附加 `... [truncated, total: Xbytes]`

### 关键埋点清单

新增或修改以下模块时，Claude 必须确保对应埋点存在：

| 模块 | 必须有的日志 |
|------|------------|
| `config.Load()` | `[INFO] config loaded from <path>` |
| `config.loadDotEnv()` | `[DEBUG] .env loaded from <path> (<N> vars applied)` |
| `spec/loader.go Tier 1` | `[INFO] cache hit for "<name>"` |
| `spec/loader.go Tier 2` | `[INFO] fetching spec for "<name>"` |
| `spec/loader.go Tier 3` | `[WARN] fetch failed, using stale cache` |
| `auth/resolver.go` | `[DEBUG] auth: injecting <provider> for service "<name>"` |
| `executor.go` 发请求前 | `[DEBUG] → <METHOD> <masked_url>` |
| `executor.go` 收响应后 | `[DEBUG] ← <status> (<bytes>, <ms>ms)` 和响应 body |

---

## 常用开发命令

```bash
go build -o bin/cora ./cmd/cora   # 编译
go test ./...                      # 全量测试
go run ./cmd/cora -- <service> <resource> <verb> [flags]  # 直接运行

# 调试模式（开启详细日志）
./bin/cora --verbose gitcode issues get --owner openeuler --repo community --number 1367

# 预览请求不发送
./bin/cora --dry-run gitcode issues get --owner openeuler --repo community --number 1367
```

---

## 验证方式

完成修改后必须执行：

1. `go build ./...` — 确保编译通过
2. `go test ./...` — 确保所有测试通过
3. 如修改了 builder（命令生成），运行 `go test ./internal/builder/...`
4. 如修改了认证或 executor，用 `--dry-run` 验证 URL 构造正确
5. 如修改了日志相关，用 `--verbose` 运行确认输出符合设计文档格式

---

## 禁止事项

- **禁止**使用 `fmt.Fprintf(os.Stderr, ...)` 输出日志，统一使用 `internal/log`
- **禁止**在日志中记录 API key、token、密码等敏感值
- **禁止** `internal/log` 引用其他 `internal/*` 包（必须是叶子依赖）
- **禁止**为 `access_token`、`apikey`、`Api-Key` 等认证参数生成 CLI flag
- **禁止**在 `Build()` 函数中硬编码具体服务的处理逻辑（必须通用）
- **禁止**修改已缓存的 OpenAPI Spec 文件（只读，刷新走 `--refresh-spec`）

---

## 相关设计文档

| 文档 | 路径 |
|------|------|
| 整体架构设计 | `spec/architecture-design.md` |
| 日志系统设计 | `spec/logging-design.md` |
| API Token 调研 | `spec/api-token-investigation.md` |
| CLI 设计参考 | `spec/reference-cli-design-patterns.md` |
| Spec 优化设计 | `spec/spec-optimization-design.md` |

---

## Smoke 测试规范

### 新增/修改服务时

添加新服务或修改已有服务的 sub-command 后，**必须**同步更新 smoke 测试：

1. **Smoke Scenario 文件** (`scenarios/<service>/<name>.yaml`)
   - 优先覆盖 **只读接口**（GET 操作），做可达性验证
   - 每个 scenario 至少包含：`exit_code: 0` + `stdout_not_empty`
   - `name` 字段必须匹配正则 `^[a-z][a-z0-9\-./]*$`，格式：`<service>/<resource>-<verb>`
   - 需要外部实例的服务（Etherpad、Jenkins）设置 `skip: true` + `skip_reason`

2. **Smoke Config** (`config/smoke-config.example.yaml`)
   - 新服务的 auth 使用 `${SMOKE_<SVC>_<FIELD>}` 占位符（由 `os.ExpandEnv` 展开）
   - 同时在注释区添加对应的 env var 文档

3. **Smoke Workflow** (`.github/workflows/smoke.yml`)
   - 新服务需要在 `env:` 块添加对应的 `${{ secrets.SMOKE_* }}` 
   - CI 会从 `smoke-config.example.yaml` 拷贝生成运行时配置
   - artifacts 需要 `permissions.actions: read` 才能下载

### 本地运行

```bash
cp config/config.yaml config/smoke-config.yaml   # 用真实凭证替换占位符
make smoke-build
./bin/smoke-runner --cora-bin ./bin/cora --config ./config/smoke-config.yaml \
  --scenarios-dir ./scenarios --report-dir ./smoke-report
```
