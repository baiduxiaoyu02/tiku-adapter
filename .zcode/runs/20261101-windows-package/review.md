# 审查报告 — 20261101-windows-package（第 2 轮）

- 审查者：审查员-3（未参与实现）｜实现者：实现者-20261101-windows-package
- base：51da1c43d030be260fad435e7247b9154b32a4ad（HEAD 同值，实现未提交，与预期一致）
- 结论：**PASS（0 blocker，1 minor）**。上轮 5 个 issue（R1/R2 blocker、R3/R4 major、R5 minor）全部确认已修复，逐条核验证据见 review.json leftover_checks。

## 上轮遗留核验（全部 fixed）

| id | 结论 | 关键证据（本会话实跑） |
|----|------|------------------------|
| R1 | fixed | tiku-tray.ps1:132 `[ref]$createdNew`（旧调用 grep 0 命中）；动态复验 `New-Object System.Threading.Mutex($true,...,[ref]$created)` exit 0 |
| R2 | fixed | tiku-setup.iss:85 停进程过滤改为 `{app}\tiku-adapter.exe` 全路径比较（grep 新串 1 命中）；:88 托盘按本安装路径限定 + 排除 $PID；上轮 T5 ⑤ 已实测全路径形式可匹配 pid=72056 |
| R3 | fixed | tier.json:19 `-run TestAPISearchAlias`；实测 go test exit 0 且真实跑 2 测试（非 no tests to run） |
| R4 | fixed | packaging/config.yaml:30 占位符 `sk-xxxx-change-me`；`sk-cpa-a719480f` 在 packaging/config.yaml、git HEAD、工作树跟踪文件均 0 命中 |
| R5 | fixed | .gitignore:9 `/dist/`；`git check-ignore -q dist/tiku-adapter.exe` exit 0 |

## 回退验证（抽样 ceil(1/3)=1）

`git worktree add %TEMP%/tiku-review-wt-r3 51da1c43` → 复制 internal/api/router_alias_test.go → `go test ./internal/api/ -run TestAPISearchAlias -count=1` → **RED exit 1**（Probe + Isomorphic 及 2 子用例全 FAIL）→ `git worktree remove --force` 清理成功。当前工作树同命令 **GREEN exit 0**。tdd.jsonl test_hash `fd532b76…4390` 与当前测试文件 sha256 一致（测试未被事后改动）；logs/T1-red.log 尾部 FAIL/EXIT=1、T1-green.log PASS/EXIT=0，TDD 记录真实。

## 新 issue

- **R6（minor，open）**：packaging/README.md:130-141 与 tasks.md:9 声称 D1/D2「已修复并复验」，但 logs/ 无任何修复后复验日志，T5-verify.log 止于缺陷发现（68 行）。修复本身本轮已独立核验成立（R1 动态复验、R2 源码 + 上轮端到端方向证据），缺的是声称中的复验产物；卸载复验需 UAC 提权，审查不重跑，留 ⑥ 前补记录或人工确认。

## 观察项（不计 issue）

- deploy/README.md 改动（Docker 安全说明 + database.path 默认值）不在 T1–T5 任一任务 files 内，但已在 tier.json declared_files 声明，内容与打包版 `data/` 语义呼应，无害。
- T5-verify.log ⑥ 记录本 run 冒烟残留 `C:\Program Files\TikuAdapter\{tiku-adapter.exe(36.7MB)}`（第三次 UAC 被取消未清），日志内已写明清理命令（管理员 `Remove-Item -Recurse -Force`），属环境残留非产品缺陷，留用户处理。
- T1 路由实现（router.go:41-50：GET/HEAD `apiSearchProbe` 200、POST 复用 `controller.Search`）与 FR-1/AC-1 一致；ocs-config.json url 已改 `http://localhost:8060/api/search`（FR-5）。
- 账本：tasks.md 任务为 T1–T5，无独立 featureId 字段；程序账本由 workflow ⑥ 后写回，本轮不因账本状态判 blocker。

## 未运行项

- ⑥ 的全量测试 / lint / 门禁未跑（按 ask 约定归属 ⑥）。
- D1/D2 修复后的端到端安装-卸载复验未重跑（需 UAC 提权装/卸系统目录软件，超出审查只读范围）；R6 已记录该证据缺口。
