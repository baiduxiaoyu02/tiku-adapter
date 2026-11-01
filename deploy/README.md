# 如何部署使用

## 您可以使用我们已经部署好的应用
等待建设中...
## 您可以独立将此程序部署到 linux,windows,macos,docker,k8s 等环境中

- windows 直接从release中下载对应的版本即可，解压后运行即可
- linux 直接从release中下载对应的版本即可，解压后运行即可
- macos 直接从release中下载对应的版本即可，解压后运行即可
- docker 使用 Docker 部署，参考下方 Docker 部署说明
- k8s 建设中...

## Docker 部署

### 构建镜像

```bash
docker build -t tiku-adapter .
```

### 运行容器

#### 基础运行（使用默认配置）

```bash
docker run -d \
  --name tiku-adapter \
  -p 8060:8060 \
  -v ./data:/app/data \
  tiku-adapter
```

#### 推荐方式：使用配置文件指定数据库路径

创建 `config.yaml` 文件并设置数据库路径为 `data/tiku.db`：
```yaml
database:
  path: "data/tiku.db"
```

然后运行容器：
```bash
docker run -d \
  --name tiku-adapter \
  -p 8060:8060 \
  -v ./data:/app/data \
  -v ./config.yaml:/app/config.yaml \
  tiku-adapter
```

### 目录结构说明

容器内的目录结构：
```
/app
├── tiku-adapter        # 程序可执行文件
├── config.yaml         # 配置文件（可选挂载）
└── data                # 数据目录（持久化）
    └── tiku.db         # SQLite 数据库文件
```

### 安全说明

- 容器以非 root 用户（UID 1001）运行，遵循最小权限原则
- 已通过 `cap_drop: ALL` 移除所有 Linux capabilities
- `security_opt: no-new-privileges` 防止提权
- 默认管理员密码通过环境变量 `TIKU_ADMIN_PASSWORD` 设置，使用 bcrypt 哈希存储
- JWT 密钥通过配置文件 `jwt_secret` 设置，**生产环境必须修改**
- 内置 HEALTHCHECK 每 30 秒检测服务健康状态

**生产部署务必修改以下默认值：**
1. 设置环境变量 `TIKU_ADMIN_PASSWORD` 为强密码
2. 在 `config.yaml` 中将 `jwt_secret` 修改为随机长字符串
3. 根据需要开启限流 `limit.enable: true`

### 配置说明

在 `config.yaml` 中可以自定义数据库路径：

```yaml
database:
  path: "data/tiku.db"  # SQLite数据库文件路径
```

**注意事项：**
- Docker 部署时，**强烈建议**在配置文件中设置 `database.path: "data/tiku.db"`，然后挂载 `/app/data` 目录
- 如果不指定 `database.path`，默认使用 `data/tiku.db`（Docker 持久化推荐）
- 将数据库路径设置在 `data` 目录下，可以只挂载该目录，不会影响容器内的程序文件
- 本地部署（非 Docker）保持默认即可，数据库会在程序所在目录的 `data/` 下创建