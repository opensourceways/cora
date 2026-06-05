# Cora OpenAPI Spec 优化与命令生成增强设计

> 目标：通过三层约束（path 优先 + tag 规范化 + verb 增强），减少 CLI 命令生成对 OpenAPI spec 质量的依赖，提高健壮性。

---

## 一、现状与问题全景

### 1.1 两平台 Spec 差异

| 维度 | GitHub | GitCode | 影响 |
|------|--------|---------|------|
| 路径格式 | `/repos/{o}/{r}/...` | `/api/v5/repos/{o}/{r}/...` | Priority 3 阈值需兼容 `api/v5/` 前缀 |
| operationId | 语义化：`listRepos`, `getIssue` | 路径编码：`get_api_v5_repos_{o}_{r}_issues` | Priority 2 被跳过，损失 verb 信号 |
| Tag 数量 | 44（细粒度） | 18（粗粒度） | GitCode 每个 tag 下 ops 更多，冲突更多 |
| Tag 命名 | `repos`, `pulls`, `orgs` | `Repositories`, `Pulls`, `Organizations` | 资源名不一致 |
| 多词 Tag | 无 | `Pull Requests`, `AI hub` | kebab 化后产生 `pull-requests`、`ai-hub` |
| Tag 范围 | 一 op 一 tag | 部分 op 错标（如 `GET repos` 标 `Users`） | 资源归类错误 |

### 1.2 已踩的坑（6 类）

```
TAG-NAME      "Pull Requests" → resource "pull-requests"（应为 pulls）
TAG-MISPLACE  GET repos/{o}/{r} tagged "Users" → 抢走 users get 动词
TAG-SYNONYM   GitHub "repos" vs GitCode "Repositories" → 两端资源名不同
VERB-PATH     POST/PATCH/DELETE 同路径 → assignees-assignees-assignees
VERB-RES      verb "issues" vs res "issue" → override 失效
OPID-STYLE    get_api_v5_user → 跳过 Priority 2 → 失去 verb 信号
```

---

## 二、设计原则（三层约束）

**核心原则**：控制信号越靠近 API 结构层，可靠性越高。

```
可靠性:   path > parameters > operationId > tags
影响范围: path 决定 URL（错了API不可用）> tags 决定分类（错了API仍可用）
```

### Layer 1：Path 优先（结构性约束）

Path 是 REST API 的结构性定义，不能随意修改。如果 path 已经暗示了资源归属，就应优先于 tag。

```
Rule 1.1: 如果 path 包含 "/repos/{owner}/{repo}/"，该 op 只属于 repos 资源
Rule 1.2: 如果 path 包含 "/{number}/comments"，verb 优先使用 "comments"
Rule 1.3: 如果 path 最后一段和 tag 名高度相关，用 path 段命名资源
```

### Layer 2：Tag 规范化（语义性增强）

Tag 是给人看的分类标签，不是严格的资源名。需要在消费前归一化。

```
Rule 2.1: 多词 tag 提取核心名词 → "Pull Requests" → "PullRequest"
Rule 2.2: 将 tag 和 path 中出现的词交叉验证，优先用 path 中的简写
Rule 2.3: 单复数归一化 → "Repositories" → "repository" ≈ "repos"
Rule 2.4: 特殊字符清理 → "Oauth2.0" → "oauth2"
```

### Layer 3：Verb 推导增强（当前逻辑修复）

在 Layer 1/2 提供正确 resource 的前提下，完善 verb 推导。

```
Rule 3.1: path-encoded opId 中仍可提取 HTTP method 前缀（"get_api_v5_..." → "get"）
Rule 3.2: 消歧优先顺序：pathContext → pathSuffix（检查 suffix≠verb）→ HTTP method
Rule 3.3: verb==res override 使用更宽松的匹配（已是 TrimSuffix）
```

---

## 三、具体修改清单

### 3.1 框架代码修改（`internal/builder/command.go`）

#### 3.1.1 `resourceName()` — Tag 规范化（Layer 2）

**当前代码：**
```go
func resourceName(op *openapi3.Operation, path string) string {
    if len(op.Tags) > 0 {
        return strings.ToLower(strings.ReplaceAll(op.Tags[0], " ", "-"))
    }
    return lastPathSegment(path)
}
```

**修改后：**
```go
func resourceName(op *openapi3.Operation, path string) string {
    if len(op.Tags) > 0 {
        return normalizeTag(op.Tags[0], path)
    }
    return lastPathSegment(path)
}

// normalizeTag converts a human-readable tag into a CLI resource name,
// using path segments as a cross-reference for preferred vocabulary.
func normalizeTag(tag string, path string) string {
    // Step 1: extract core noun from multi-word tags
    // "Pull Requests" → "pull-request", "AI hub" → "ai-hub"
    words := strings.Fields(strings.ToLower(tag))
    
    // Step 2: kebab-case the tag
    candidate := strings.Join(words, "-")
    
    // Step 3: check if path contains a preferred short form
    // e.g. path "/api/v5/repos/.../pulls/..." → prefer "pulls" over "pull-requests"
    pathSegs := strings.Split(strings.Trim(path, "/"), "/")
    for _, s := range pathSegs {
        s = strings.ToLower(s)
        if s == "" || strings.HasPrefix(s, "{") || s == "api" || len(s) < 2 {
            continue
        }
        // Skip version segments
        if len(s) >= 2 && s[0] == 'v' && isDigit(s[1]) {
            continue
        }
        // Check if path segment is a short synonym for the tag
        // "pulls" ↔ "pull-requests", "repos" ↔ "repositories", "orgs" ↔ "organizations"
        if isShortSynonym(s, candidate) {
            return s
        }
    }
    
    return candidate
}
```

**影响**：`"Pull Requests"` → 自动识别为 `"pulls"`，不需要手工改 spec tag。

#### 3.1.2 `verbName()` — path-encoded opId 提取 method（Layer 3）

**当前代码**跳过了 Priority 2 对 GitCode opId 的处理。

**新增**：在 path-encoded opId 不可用时，仍从 opId 前缀提取 HTTP method 对应的 verb：

```go
// 在 Priority 2 之后、Priority 3 之前插入：
// Priority 2.5: for path-encoded operationIds (GitCode style),
// extract the HTTP method prefix which is always the first segment.
// "get_api_v5_user" → "get", "post_api_v5_..." → "create".
if isPathEncodedOpID(opID) {
    parts := strings.SplitN(strings.ToLower(opID), "_", 2)
    if len(parts) > 0 {
        switch parts[0] {
        case "get":
            if hasTrailingParam(path) { return "get" }
            return "list"
        case "post":
            return "create"
        case "put", "patch":
            return "update"
        case "delete":
            return "delete"
        }
    }
}
```

**影响**：GitCode 的 `get_api_v5_user` 不再走到 Priority 4 `httpMethodVerb`，而是通过 Priority 2.5 获得 `"list"`。其他 path-encoded opId 同理。

#### 3.1.3 `Build()` — Path 优先校验（Layer 1）

在 `Build()` 中，对每个 operation 做 path 归属校验：

```go
// Path-priority check: if path strongly implies a different resource,
// override the tag-derived resource.
pathRes := resourceFromPath(e.path)
if pathRes != "" && pathRes != res {
    // Path contains a strong resource signal (e.g. "repos/{o}/{r}/issues")
    // that conflicts with the tag. Use path-derived resource.
    res = pathRes
}
```

`resourceFromPath()` 逻辑：

```go
// resourceFromPath extracts the resource name from the path structure.
// Recognizes patterns like:
//   /repos/{o}/{r}/issues/...   → "issues" (not "repos")
//   /repos/{o}/{r}/pulls/...    → "pulls"  (not "repos")
//   /enterprises/{e}/members/... → "organizations" (maps to orgs tag)
// Returns "" if no clear signal.
func resourceFromPath(path string) string {
    // Known sub-resource segments that indicate a different resource group
    subResources := map[string]string{
        "issues":        "issues",
        "pulls":         "pulls",
        "branches":      "branch",
        "labels":        "labels",
        "comments":      "comments",
        "commits":       "commit",
        "releases":      "release",
        "tags":          "tag",
        "members":       "member",
        "collaborators": "member",
        "webhooks":      "webhooks",
    }
    segs := strings.Split(strings.Trim(path, "/"), "/")
    for _, s := range segs {
        if res, ok := subResources[strings.ToLower(s)]; ok {
            return res
        }
    }
    return ""
}
```

**影响**：即使 spec 作者把 `GET repos/{o}/{r}` 标成 `Users` tag，这个 op 也会被强制归到 `repos` 资源（因为 path 里有 `repos/...`）。

#### 3.1.4 消歧顺序调整（已是当前代码，不需改）

当前消歧逻辑已经是：pathContext → pathSuffix（suffix≠verb）→ HTTP method。保持不变。

### 3.2 Spec 文件修改

#### 3.2.1 GitCode spec（`assets/openapi/gitcode/openapi.json`）

已经改过的不再重复。仍需修正：

| # | 端点 | 当前 tag | 修正为 | 原因 |
|---|------|---------|--------|------|
| 1 | `GET /api/v5/repos/{o}/{r}/notifications` | `Users` | `Repositories` | 不应属于 Users |
| 2 | `PUT /api/v5/repos/{o}/{r}/notifications` | `Users` | `Repositories` | 同上 |
| 3 | `GET /api/v5/user/notifications` | `Users` (already) | 保持不变 | 这个是 user 级别的通知 |

> 注：改完 3.1 后，第 1、2 项即使不改 spec，Layer 1 path 优先校验也会自动修正。但建议还是把 spec 改对。

#### 3.2.2 可选：GitHub spec（`assets/openapi/github/api.github.com.json`）

GitHub spec 质量较高，无需修改。仅作为 tag 规范化测试的参照。

### 3.3 CLAUDE.md 修改

`claude-specification/project/CLAUDE.md` 的关键架构约定部分需要更新：

**命令生成机制一节**，将：

```
- 资源名 = 操作的第一个 tag（小写 kebab-case）
```

替换为：

```
- 资源名由三层推导：
  1. Path 优先：path 中有明确子资源段（如 /pulls/、/issues/）的，优先使用 path 段
  2. Tag 规范化：对 tag 做归一化（多词→连词、与 path 交叉验证、单复数统一）
  3. Fallback：path 最后非参数段
- 动词按优先级派生：UsingGET 后缀 → 已知动词前缀（GitCode 额外提 method）→ path 结构 → HTTP method fallback
- GitCode 风格的 path-encoded operationId 在 Priority 2.5 提取 HTTP method 前缀
```

**Spec 文件规范一节**，新增：

```
## OpenAPI Spec 编写规范

添加新服务或在现有 spec 中新增端点时：

### Tag 规范
- 使用单数或复数短名词，避免多词组合：`"Pulls"` 而非 `"Pull Requests"`
- 与 GitHub API 保持一致命名：`repos`/`pulls`/`issues`/`users`/`orgs`
- 每个 operation 只标一个 tag（第一个 tag 用作资源名）
- Tag 含义应与 REST 资源概念一致，不应跨资源

### 路径中不宜出现的 Tag 映射
- 路径是 `repos/{o}/{r}/...` 开头 → tag 应为 `Repositories`
- 路径是 `repos/{o}/{r}/pulls/...` → tag 应为 `Pulls`（不是 Repositories）
- 路径是 `repos/{o}/{r}/issues/...` → tag 应为 `Issues`（不是 Repositories）
```

---

## 四、修改影响评估

| 修改 | 影响范围 | 风险 | 收益 |
|------|---------|------|------|
| Tag 规范化 | `resourceName()` | 低：仅在 path 有同义词时改变行为 | 不再需要手工改 spec tag |
| Path-encoded opId 提 method | `verbName()` | 低：仅影响 GitCode opId | GitCode 命令名从 `list-user` → `get` |
| Path 优先校验 | `Build()` | 中：可能改变已有资源归属 | 彻底解决 TAG-MISPLACE 问题 |
| Spec tag 修正 | spec 文件 | 低：不影响 API 调用 | spec 自文档一致性 |
| CLAUDE.md 更新 | 文档 | 无 | 后续开发有规范可循 |

---

## 五、实施顺序

```
Phase 1: 框架修改
  ├── 3.1.1  resourceName() Tag 规范化
  ├── 3.1.2  verbName() path-encoded opId 提 method  
  ├── 3.1.3  Build() Path 优先校验
  └── 补充测试用例

Phase 2: Spec 修正
  └── 3.2.1  GitCode spec 残留 tag 修正

Phase 3: 文档更新
  └── 3.3     CLAUDE.md 规范更新

Phase 4: 验证
  ├── go test ./... 全量通过
  ├── 两端命令对比输出一致
  └── 已修复 case 无回归
```
