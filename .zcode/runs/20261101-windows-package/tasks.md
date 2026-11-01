# 任务：题库服务 Windows 桌面打包

说明：TDD 任务只 T1（Go 路由代码）；T2-T5 为构建/打包/文档任务，标 skip-tdd，验证方式见各自描述。

- [x] T1 files: [internal/api/router.go, internal/api/router_alias_test.go] — /api/search 路由别名，TDD：先写 internal/api/router_alias_test.go（httptest：GET/HEAD 探活 200、POST 别名与 /adapter-service/search 同构）看 RED，再加路由转 GREEN
- [x] T2 skip-tdd files: [scripts/build-windows.ps1] depends: [T1] — Windows 原生构建脚本（GOTOOLCHAIN=go1.21.13 锁工具链），验证=编译退出码 0 + exe 存在 + 启动冒烟 8060 探活 200
- [x] T3 skip-tdd files: [packaging/tray/tiku-tray.ps1, packaging/config.yaml, web/ocs-config.json] depends: [T2] — 打包目录与托盘（托盘启动本地 exe 而非 Docker），ocs-config url 改 http://localhost:8060/api/search，验证=托盘脚本启动 exe 且探活 200
- [x] T4 skip-tdd files: [packaging/tiku-setup.iss] depends: [T3] — Inno Setup 安装包（安装目录/开始菜单/桌面快捷方式/可选自启/卸载保留数据），验证=ISCC 编译退出码 0 + 产出 setup exe
- [x] T5 skip-tdd files: [packaging/README.md] depends: [T4] — 端到端冒烟与文档（静默安装→启动→/api/search 探活→POST 搜题正确→卸载验证），README 记录安装/使用/卸载与 OCS 对接（冒烟发现 T3 托盘 UI 崩溃、T4 卸载器停进程失效两缺陷，见 logs/T5-verify.log；两缺陷（审查 R1/R2）已由修复轮修复并复验，setup exe 已重建去掉内嵌真 key）
