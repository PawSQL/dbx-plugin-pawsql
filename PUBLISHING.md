# 发布到 DBX 插件市场（dbx-store）

本文是发布 PawSQL DBX 插件的操作手册与 PR 材料。按顺序执行即可。

## 关键结论（先读）

1. **提交方式**：单 PR 到 `t8y2/dbx-store`，只提交 `publishers/pawsql.json` + `candidates/com.pawsql.dbx.json`；`plugins/`、`catalog/` 由签名工作流生成，不可手改。
2. **native 插件不能本地交叉打包**：`dbx-plugin package . --target darwin-arm64` 会报 `does not match build host 'windows-x64'`。必须走 GitHub Release 工作流（`.github/workflows/plugin-release.yml`），它在 5 个平台的原生 runner 上各自打包并上传 `dist/*.dbxp` + `dist/*.artifact.json`。
3. **sha256/size 必须来自实际托管产物**：candidate 的 `targets[].sha256`/`size` 会被签名工作流下载核对，因此必须从 release 的 `*.artifact.json` 照抄，不能用本地估算值。
4. **本地已完成的准备**：源码已 git 提交（不含 `dist/`、`backend/*.exe`、`.claude/`、`.playwright-mcp/`）；`manifest.json` 与 `.dbx-store.json` 的 permissions 均为 `["host.events"]`；Go 后端已验证可在 darwin/arm64、darwin/amd64、linux/arm64、linux/amd64、windows/amd64 全平台编译（`CGO_ENABLED=0 go build ./...` 通过）。

---

## 步骤（在 GitHub 上操作）

### 1. 建仓库并推送源码

```bash
cd /e/Workspace/pawsql-dbx

# 若本地尚未提交，先执行：
#   git init
#   git add -A
#   git commit -m "Initial release"

gh repo create pawsql/dbx-plugin-pawsql --public --source . --push
git tag v1.0.0
git push origin v1.0.0
```

### 2. 创建 Release（触发 5 平台打包）

```bash
gh release create v1.0.0 \
  --repo pawsql/dbx-plugin-pawsql \
  --title "v1.0.0" \
  --notes "Initial release"
```

发布会触发 `.github/workflows/plugin-release.yml`（复用 `t8y2/dbx` 的 `plugin-release-reusable.yml@v0.6.15`）。等 Action 跑完后，release 上会出现 5 个 `com.pawsql.dbx-1.0.0-<target>.dbxp` 与 5 个 `dist/*.artifact.json`。

### 3. 回填 candidate 的 sha256/size

```bash
gh release view v1.0.0 --repo pawsql/dbx-plugin-pawsql --json assets \
  --jq '.assets[].name'
```

下载每个 `com.pawsql.dbx-1.0.0-<target>.artifact.json`，其内容即 `{target, url, sha256, size}`。把各 target 的 `sha256`/`size` 填进 `dbx-store/candidates/com.pawsql.dbx.json` 的对应项（`url` 已就绪，无需改）。

### 4. Fork dbx-store 并提交

```bash
gh repo fork t8y2/dbx-store --clone
cd dbx-store

# 把本机已准备好的两个文件放进去（E:\Workspace\dbx-store\）
#   publishers/pawsql.json
#   candidates/com.pawsql.dbx.json

git checkout -b add-pawsql-plugin
git add publishers/pawsql.json candidates/com.pawsql.dbx.json
git commit -m "Add PawSQL SQL Optimizer (com.pawsql.dbx)"
git push -u origin add-pawsql-plugin

# 本地校验通过后开 PR（PR 正文见下文）
node scripts/validate.mjs
gh pr create --repo t8y2/dbx-store \
  --title "Add PawSQL SQL Optimizer (com.pawsql.dbx)" \
  --body-file pr-body.md
```

### 5. 等待签名

CI 会保持红色，报 `open candidate(s) awaiting signing`，这是预期行为——它阻断未签名内容被合并。维护者 review 后跑 `/sign` 签名工作流，签名产物落 `plugins/` + `catalog/`，随后 merge。

---

## PR 正文（保存为 pr-body.md 后交给 `gh pr create --body-file`）

```text
## Plugin submission

- Plugin ID: com.pawsql.dbx
- Version: 1.0.0
- Publisher ID: pawsql

## Source repository and tag

- https://github.com/pawsql/dbx-plugin-pawsql
- Built from tag v1.0.0

## Capabilities and user workflow

A standalone workbench panel: paste a SQL statement, choose the database type
(or, when opened from a connection's context menu, the type is auto-detected),
and get a PawSQL optimization report — rule violations, recommended index DDLs,
query rewrites, and a What-If validation with before/after execution plans.
Supports anonymous mode (shared key) or account sign-in (workspace-aware
schema analysis).

## Permissions, data access, and network access

- Manifest permissions: `host.events`.
- Network: HTTPS JSON to the user-configured PawSQL server only (default
  https://www.pawsql.com), endpoints getUserKey / createAnalysisAsync /
  getAnalysisSummary / getStatementDetails / listWorkspaces.
- Data: the pasted SQL text and the login email/password (password is used only
  to exchange for an API key and is never stored). The API key, email, and
  option flags are persisted by the sidecar to config.json under
  DBX_PLUGIN_DATA_DIR (file mode 0600). No local files are read, and no other
  application data is accessed.

## Native sidecar behavior

Yes — a Go native sidecar (bin/dbx-plugin-pawsql) communicates with the UI over
DBX sidecar protocol v1 (stdio-jsonl). It:

- writes only its own config.json (email, API key, option flags — never the
  password) under DBX_PLUGIN_DATA_DIR, atomically (temp file + rename, 0600);
- opens the "more details" link in the default browser via rundll32/open/
  xdg-open, restricted to http(s) URLs, without a shell;
- makes outbound HTTPS requests to the configured PawSQL server;
- listens on no ports and spawns no other subprocesses.

## License

Apache-2.0

## Homepage or support URL

https://github.com/pawsql/dbx-plugin-pawsql

## Confirmation

- [x] The candidate packages are unsigned and match the disclosed source tag.
- [x] I have not included private keys, tokens, credentials, or proprietary data.
- [x] The candidates are listed in `candidates/com.pawsql.dbx.json`; `plugins/`
  and `catalog/` were not edited manually.
```

---

## 本地已完成的修改清单

- `dbx-store/candidates/com.pawsql.dbx.json`：`permissions` → `["host.events"]`；`source` → tag 寻址 `…/tree/v1.0.0`；`icon` → pin 到 `…/v1.0.0/assets/plugin.svg`。
- `pawsql-dbx/manifest.json`、`.dbx-store.json`：图标统一从占位 `assets/plugin.png` 改为 DBeaver 爪印 `assets/plugin.svg`（DBX manifest 的 `icon` 字段 schema 允许任意相对资源路径，SVG 合法）。
- `pawsql-dbx/README.md`：布局说明同步更新为 `assets/plugin.svg`。
- `pawsql-dbx/assets/plugin.png`：已删除（旧占位图，现无引用）。
- `pawsql-dbx/.gitignore`：追加 `.claude/`、`.playwright-mcp/`。
