# 审查报告：20261101-fix-createuser-password（第 2 轮）

审查者：审查员-3（未参与实现）｜基线：fcdd2ad465ad275f4801a0e2c80955213df72ece｜日期：2026-11-01

## 结论

**通过（无 blocker，3 项 minor open + 上轮 R1 已确认修复）**。本轮回退验证独立复跑：基线 worktree（fcdd2ad）下两个测试文件均失败、回到工作树均通过，抽样 2/2（≥ ceil(4/3)=2）。scope 声明问题（上轮 R1）已在 scope 修复轮补全。

## 本轮实测证据（全部本会话执行）

- **改动范围**：`git status --porcelain -uall` → 产品文件 user.go、router.go（修改）+ ocs_config.go、ocs-config.html、user_test.go、ocs_config_test.go、scripts/hashgen/main.go（新增），与 tier.json declared_files（7 个）一一对应。
- **FR-1 修复路径**：`git diff internal/controller/user.go` → 新增 `userCreateRequest`（内嵌 entity.User + 外层 `Password string json:"password"`），`user.Password = req.Password` 后入库；entity.User 的 `json:"-"` 未改。
- **FR-2 路由与页面**：diff 中 `router.GET("/ocs-config", controller.OcsConfig)`（internal/api/router.go:48）；ocs_config.go 经 `//go:embed ocs-config.html` 内嵌。
- **转义链路独立验证**（node）：页面 config 经 `JSON.stringify` → `JSON.parse` 后 options handler 内容为 `return (env)=>env.options?.split('\n')`（反斜杠+n，OCS eval 后即换行 split），images handler 存在——双层转义正确。
- **TDD 哈希**：`sha256sum` user_test.go=a2afec4a…、ocs_config_test.go=eb8fe54d…，与 tdd.jsonl 4 条记录一致；`newAliasTestRouter` 在基线已有 internal/api/router_alias_test.go:40。
- **测试复跑**：`go test ./internal/controller/ -run TestUser -count=1 -v` → 2 用例 PASS；`go test ./internal/api/ -run TestOcsConfig -count=1` → ok。
- **回退验证**：`git worktree add --detach /tmp/rev-tiku-fcdd2ad-r3 fcdd2ad` + 复制当前测试文件 → controller `undefined: userCreateRequest` [build failed]、api `GET /ocs-config 期望 200，实际 404`；工作树同命令全 PASS。worktree 已 `remove --force`，`git worktree list` 无残留。
- **AC-3 只读探活**：`curl http://127.0.0.1:8060/ocs-config` → 200；`/api/search` → 200。
- **上轮证据抽查**：logs/T1-red.log 实读与本轮回退实测输出逐字一致；review.md 引用的文件行号、命令、输出均核对无失实。

## Issues

| id | 严重度 | 状态 | 摘要 |
|----|--------|------|------|
| R1 | minor | fixed | 上轮：declared_files/tasks.md 与实际改动不一致 → 本轮实测 tier.json 7 个文件、tasks.md 已补全、tdd.jsonl 已补复验记录 |
| R2 | minor | open | 上一 run（windows-package）ship.json 遗留未提交改动仍在工作树，交付暂存勿混入 |
| R3 | minor | open | AC-1 端到端（创建用户→登录成功）仍无测试，T3-verify.log 无登录记录（grep 计数 0）；单元级根因已锁定，建议下轮补 httptest 端到端用例 |
| R4 | minor | open | gates.json run_id 已重跑为 0e3cb184fb6d，gates-plan.txt 仍 d83a5393bc04；head_sha 均为基线，内容无矛盾，陈旧产物提示 |

## 其他说明

- **featureId**：tasks.md 无 featureId 字段，tier.json 无账本条目——按约定不因账本未更新判 blocker，账本由 ⑥ 后 workflow 写回。
- **未验证项**：OCS 客户端「读取剪贴板→导入→搜题」链路需 OCS 环境无法在本机验证；未向已部署实例写数据（仅只读探活）。
- 临时 worktree 已清理，本轮回退验证无残留。
