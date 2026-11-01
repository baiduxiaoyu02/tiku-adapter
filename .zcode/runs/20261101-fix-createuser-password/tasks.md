# 任务：CreateUser 修复 + OCS 配置生成页

说明：两个功能同一构建交付。TDD 任务只 T1（Go 控制器代码）；T2 为页面/路由任务（绑定测试覆盖路由注册），T3 构建部署。

- [x] T1 files: [internal/controller/user.go, internal/controller/user_test.go, scripts/hashgen/main.go] — 新增 userCreateRequest DTO（内嵌 entity.User + 外层 Password json:"password"），CreateUser 改绑 DTO 并把 req.Password 写回 user.Password 再 bcrypt 入库；TDD：internal/controller/user_test.go 写 TestUserCreateRequest（JSON 含 password → DTO.Password 收到值）与 TestUserEntityNoPasswordLeak（entity.User 序列化不含 password 字段），先 RED 后 GREEN；scripts/hashgen/main.go 为本轮辅助 bcrypt 哈希生成 CLI（scope 修复轮补记）
- [x] T2 files: [internal/controller/ocs_config.go, internal/controller/ocs-config.html, internal/api/router.go, internal/api/ocs_config_test.go] depends: [T1] — GET /ocs-config 返回内嵌 HTML 页（ocs_config.go 经 //go:embed ocs-config.html 内嵌；展示 AnswererWrapper JSON：url=location.origin+'/api/search'，含 options/type/images/suggestion handler，「复制配置」按钮 + 默认解析器导入步骤说明），路由注册 + 绑定测试（internal/api/ocs_config_test.go，GET /ocs-config 含 images handler 字符串）
- [x] T3 skip-tdd files: [dist/tiku-adapter.exe, dist/tiku-adapter-setup.exe] depends: [T1, T2] — 重建 Windows exe 与 Inno Setup 安装包，部署到 D:\user\ruanjian\TikuAdapter\ 并重启实例，验证=探活 200 + 搜题正确 + /ocs-config 页面可访问
