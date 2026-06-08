# Spec Normalizer 设计

> 目标：在 OpenAPI spec 加载后、命令生成前，插入一层声明式 spec 修正，将硬编码补丁从 `command.go` 中移出。

## 问题

`command.go` 中存在 6 处硬编码逻辑（~60 行），专门处理 spec 质量问题：

- `isShortSynonym()` / `canonicalTag` / `normalizeTag()` — tag 同义词映射
- `resourceFromPath()` — path 段→资源名映射
- `isPathEncodedOpID()` / Priority 3.5 — GitCode operationId 格式
- `priorNonParam <= 4` — 版本化 API 路径容差

**根因**：spec 质量不够 → 代码中打补丁。应该在上游（spec 层）修正。

## 设计

### 架构

```
Spec 加载                  Normalizer               Builder
  │                           │                       │
  ▼                           ▼                       ▼
openapi3.T ──→ Normalizer.Normalize() ──→ openapi3.T ──→ Build()
  (raw)        │                              (clean)       │
               │                                           ▼
               └─ 每个服务注册的 rules              Cobra 命令树
```

### 核心接口

```go
// internal/spec/normalizer.go

// NormalizeRule describes one transformation to apply to a spec.
type NormalizeRule struct {
    // TagRemap renames a tag: {"Pull Requests" → "Pulls"}.
    TagRemap map[string]string

    // ResourceAlias maps a kebab-cased tag to a canonical resource name:
    // {"pull-requests" → "pulls", "repositories" → "repos"}.
    ResourceAlias map[string]string

    // TagReassign moves operations matching a path prefix to a target tag:
    // e.g. {path_prefix: "/repos/{o}/{r}/pulls", tag: "Pulls"}
    TagReassign []TagReassignRule

    // OpIDVerbExtract enables HTTP-method-prefix extraction for path-encoded
    // operationIds (GitCode style: get_api_v5_…).
    OpIDVerbExtract bool

    // StripVersionPrefix removes leading version-like segments (api/v5) from
    // path-derived names so they don't inflate priorNonParam counts.
    // Applied internally by the normalizer to clean path representations.
    StripVersionPrefix bool
}

type TagReassignRule struct {
    PathPrefix string // e.g. "/repos/{owner}/{repo}/pulls"
    Tag        string // target tag, e.g. "Pulls"
}

// Normalizer applies a set of rules to an OpenAPI spec.
type Normalizer struct {
    rules NormalizeRule
}

func NewNormalizer(rules NormalizeRule) *Normalizer
func (n *Normalizer) Normalize(spec *openapi3.T) *openapi3.T
```

### 规则注册

每个服务在 `internal/registry/builtin.go` 中注册自己的 normalizer：

```go
addBuiltin(r, cfg, builtinDef{
    name:      gitcodeName,
    specData:  assets.GitcodeSpec,
    cacheDir:  cacheDir,
    ttl:       ttl,
    normalize: NormalizeRule{
        ResourceAlias: map[string]string{
            "repositories":  "repos",
            "organizations": "orgs",
            "pull-requests": "pulls",
        },
        TagReassign: []TagReassignRule{
            {PathPrefix: "/api/v5/repos/{owner}/{repo}/pulls",  Tag: "Pulls"},
            {PathPrefix: "/api/v5/repos/{owner}/{repo}/issues", Tag: "Issues"},
            {PathPrefix: "/api/v5/repos/{owner}/{repo}/labels", Tag: "Labels"},
        },
        OpIDVerbExtract: true,
    },
})

addBuiltin(r, cfg, builtinDef{
    name:      githubName,
    specData:  assets.GithubSpec,
    cacheDir:  cacheDir,
    ttl:       ttl,
    normalize: NormalizeRule{}, // GitHub spec is clean, no rules needed
})
```

### 调用点

在 `Build()` 之前，spec 经过 normalizer：

```go
// registry/builtin.go 的 addBuiltin()
loader := spec.NewEmbeddedLoader(...)
normalizer := spec.NewNormalizer(b.normalize)

// ... later, when loading:
rawSpec, _ := loader.Load(ctx)
cleanSpec := normalizer.Normalize(rawSpec)
cmd := builder.Build(svcName, cleanSpec, ...)
```

## Normalizer 实现细节

### TagRemap

```go
// 遍历所有 operation，替换 tag 名称
for _, pathItem := range spec.Paths.Map() {
    for _, op := range pathItem.Operations() {
        for i, tag := range op.Tags {
            if newTag, ok := rules.TagRemap[tag]; ok {
                op.Tags[i] = newTag
            }
        }
    }
}
```

### ResourceAlias

等效于 `canonicalTag` fallback。在 `normalizeTag()` 被调用前生效——直接修改 tag 值。

### TagReassign

```go
for path, pathItem := range spec.Paths.Map() {
    for _, rule := range rules.TagReassign {
        if strings.HasPrefix(path, rule.PathPrefix) {
            for _, op := range pathItem.Operations() {
                op.Tags[0] = rule.Tag  // 覆盖第一个 tag
            }
        }
    }
}
```

### OpIDVerbExtract

不修改 spec。在 `verbName()` 中，当 `rules.OpIDVerbExtract==true` 且 `isPathEncodedOpID(opID)` 为 true 时，启用 Priority 3.5。

这个逻辑仍然在 `command.go` 中，但由 normalizer 标记控制，而非硬编码检测 GitCode。

## 影响分析

### 从 command.go 中移除的逻辑

| 当前硬编码 | 移除方式 |
|-----------|---------|
| `isShortSynonym()` map | → `ResourceAlias` |
| `canonicalTag` map | → `ResourceAlias` |
| `normalizeTag()` path 交叉验证 | → `ResourceAlias` + `TagReassign` |
| `resourceFromPath()` map | → `TagReassign` |
| `isPathEncodedOpID()` | → 仍保留在 `command.go` 中（通用的 opId 格式检测） |
| Priority 3.5 | → 由 `OpIDVerbExtract` 规则控制，而非自动检测 GitCode |
| `priorNonParam <= 4` | → 如果 spec 中 tag 正确，此阈值可以降低 |

### 代码量变化

- `command.go`：删除 ~40 行硬编码
- `normalizer.go`：新增 ~80 行
- `builtin.go`：每个服务 +5 行配置

## 实施步骤

1. 创建 `internal/spec/normalizer.go`，实现 `NormalizeRule` + `Normalizer.Normalize()`
2. 创建 `internal/spec/normalizer_test.go`
3. 修改 `internal/registry/builtin.go`，`builtinDef` 增加 `normalize` 字段
4. 修改 `internal/registry/builtin.go`，`addBuiltin()` 中接入 normalizer
5. 从 `command.go` 删除硬编码逻辑
6. 更新测试
7. 完整测试通过

## 备选方案（未采纳）

- **x-cora-* 扩展字段**：直接修改 spec JSON。优点是完全消除 Go 代码中的配置；缺点是需要手动维护 spec 文件，容易与上游不同步。
- **外部 YAML 配置**：类似 views.yaml。优点是 spec 和修正分离；缺点是增加配置复杂度。

当前选择的「builtin 注册」方案折中：配置和 spec 在同一个注册点，声明式但无需额外文件。
