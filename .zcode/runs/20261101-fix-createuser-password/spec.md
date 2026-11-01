# 规格：题库服务功能修复 + OCS 配置生成

## 目标（Goal）

两件事一次交付：
1. 修复 CreateUser 密码绑定失效（面板创建的用户全部无法登录的真 bug）。
2. 新增「OCS 配置生成」页面：面板内置按本机地址生成的完整 OCS 题库配置，
   一键复制后粘进 OCS 即可导入，不再依赖外部生成流程。

## 用户故事

1. 作为用户，我在面板「用户列表」新建的用户能直接登录（当前创建的用户密码恒为空哈希）。
2. 作为用户，我打开 `http://localhost:8060/ocs-config` 就能看到按本机地址生成的 OCS
   题库配置 JSON，点「复制」后到 OCS「读取剪贴板」粘贴即可导入（含图片题 images 支持）。

## 需求（Functional Requirements）

- FR-1 CreateUser 密码绑定：控制器改绑包含 password 字段的请求 DTO
  （内嵌 entity.User + 外层 Password），创建时对 DTO.Password 做 bcrypt 后入库；
  entity.User 的 `json:"-"` 保持不变（列表响应仍不泄露哈希）。
- FR-2 OCS 配置页：新增 `GET /ocs-config` 路由，返回内嵌 HTML 页面：
  展示完整 AnswererWrapper JSON（url = `location.origin + '/api/search'`，含
  options/type/images/suggestion 处理器），带「复制配置」按钮与导入步骤说明。

- FR-3 重建与部署：重新构建 Windows exe 与 Inno Setup 安装包（包含上述两项修复），
  部署更新到用户已安装的 D:/user/ruanjian/TikuAdapter/ 目录并重启实例。

## 验收（Acceptance）

- AC-1（FR-1）自动化测试：userCreateRequest 绑定 JSON 后 Password 收到值；
  且 entity.User 序列化不含 password 字段（列表不泄露哈希）。端到端：API 创建
  用户 → 该用户登录成功。
- AC-2（FR-2）`GET /ocs-config` 返回 HTML，页面含正确 JSON（url 指向本机
  /api/search，含 images handler）；JSON 可被 JSON.parse 且通过 OCS 图片支持检测。

- AC-3（FR-3）dist/ 产出新 exe 与安装包；D:/user/ruanjian/TikuAdapter/ 的实例
  重启后探活 200，/ocs-config 页面可访问。

## 非目标（Out of Scope）

- 不做 OCS「一键获取」postMessage 提供方页（OCS Desktop 的一键获取按钮只连官方
  提供方，自定义提供方页无法被触发——剪贴板流程已够用）。
- 不改用户列表/删除接口。
- 不做 Docker 镜像重建（Windows 桌面版为本轮交付物；Docker 版下次构建自动带上）。

## 澄清（Clarification）

- 「生成配置文件」的落点 = 内嵌 HTML 页面（浏览器打开即用），不是下载文件——
  剪贴板复制 → OCS「读取剪贴板」是已验证的最短路径。
- 图片题支持已在 OCS 配置 JSON 的 images handler 中包含（上一轮已落地）。

## 假设

- 面板创建的用户 ParentID 指向创建者（沿用现有逻辑，不改）。
- 旧 admin 账号（changeme）保留，不做删除（用户可自行在面板删除）。
