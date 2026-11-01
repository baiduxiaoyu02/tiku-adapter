# 任务：题库服务 v2 — GitHub 发布与模型供应商管理

说明：TDD 任务只 T1（Go 路由别名测试）与 T4（供应商 API 单元测试）；T2/T3/T5/T6 为页面与构建任务（标 skip-tdd 或行为已由 T1 覆盖的回归任务），验证方式见各自描述。

- [ ] T1 files: [internal/api/router.go, internal/api/router_alias_test.go] — /api/search 路由别名，TDD：先写 internal/api/router_alias_test.go（httptest：GET/HEAD 探活 200、POST 别名与 /adapter-service/search 同构）看 RED，加路由转 GREEN
- [ ] T2 files: [internal/controller/providers.go, internal/controller/providers_test.go, internal/registry/config.go, configs/config.go, internal/api/router.go] depends: [T1] — 供应商配置：Provider 结构（名称/base_url/api_key/模型/视觉模型/当前标记）进 configs.Config；API 增删改查+连通测试+当前切换（GET 列表密钥掩码）；config.yaml 持久化；TDD：providers_test.go 覆盖 Provider 结构校验与持久化往返（先红后绿）
- [ ] T3 skip-tdd files: [web/src/views/ProviderManage.vue, web/src/router 路由注册] depends: [T2] — 供应商管理页面（Vue）：增删改表单/连接测试按钮/当前切换/保存后提示重启生效（面板按钮触发重启接口）；验证=页面可访问且表单可用
- [ ] T4 skip-tdd files: [internal/controller/ocs_config.go, web/ocs-config-template.html] depends: 无 — GET /ocs-config 内嵌 HTML 页（按 location.origin 生成完整 OCS 配置 JSON + 复制按钮 + 四步导入说明 + images handler），验证=页面 200 且含 images handler 字符串
- [ ] T5 skip-tdd files: [.goreleaser.yaml, dist/tiku-adapter-setup.exe] depends: [T2, T4] — 补 .goreleaser.yaml archives 额外文件（config 模板/托盘/ocs-config 资源）；本机构建 exe+Inno Setup 安装包；验证=构建退出码 0+压缩包含关键文件
- [ ] T6 skip-tdd files: [web/src（版本号展示）] depends: [T2] — 面板显示当前运行版本号（configs.Version 或构建时注入），验证=页面可见版本号
- [ ] T7 skip-tdd files: [README.md] depends: [T5, T6] — README 更新（安装/填 key/OCS 对接/发布流程四节），验证=README 含四节标题
