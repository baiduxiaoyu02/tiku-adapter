# tikuAdapter 云端独立部署包

## 目标

在云服务器（139.224.41.197）上独立运行 tikuAdapter 题库服务，**完全不依赖本地机器**（不需要本地 CPA、openresty、frpc 隧道）。

## 架构

```
云端服务器
│
└─ tiku-adapter 容器（唯一组件）
   ├─ 端口: 8060
   ├─ AI: 直连 https://token.sensenova.cn/v1（glm-5.2，公网）
   ├─ 题库: icodef + buguake（公网免费源）
   └─ 数据: 独立 SQLite（云端本地）
```

对比本地方案：
| 组件 | 本地部署 | 云端独立部署 |
|------|---------|-------------|
| tiku-adapter | ✓ 容器 | ✓ 容器 |
| AI 通路 | openresty → CPA（本地）| 直连 sensenova 公网 |
| 依赖本地 | 是 | **否** |
| 依赖 frpc | 是（外网访问时）| 否 |

## 部署步骤

### 1. 上传文件到云服务器

```bash
# 从本地执行（或用 1Panel 文件管理上传）
scp tiku-adapter-image.tar.gz config.yaml deploy.sh root@139.224.41.197:/opt/tiku-adapter/
```

### 2. 在云服务器上执行

```bash
cd /opt/tiku-adapter
# 改 JWT 密钥（可选但推荐）
sed -i "s/CHANGE_ME_TO_RANDOM_STRING_BEFORE_DEPLOY/$(openssl rand -hex 32)/" config.yaml
# 执行部署
bash deploy.sh
```

### 3. 验证

```bash
curl -s -X POST http://localhost:8060/adapter-service/search \
  -H "Content-Type: application/json" \
  -d '{"question":"以下哪项不是我国的基本国策","options":["计划生育","保护环境","对外开放","以经济建设为中心"],"type":0}'
```

应返回 `answerKey: ["A"]`。

## 端口与防火墙

- 容器端口：8060
- 云防火墙需放行 8060（如果要从外部访问）
- 若只给 frpc 内网用，无需放行

## 管理

```bash
docker logs -f tiku-adapter      # 日志
docker restart tiku-adapter      # 重启
docker stop tiku-adapter         # 停止
```

## 常见问题

**Q: AI 不工作？**
检查 config.yaml 里 `ai.enable: true` 且 base_url 是 `https://token.sensenova.cn/v1`。
测试连通：`docker exec tiku-adapter wget -q -O- https://token.sensenova.cn/v1/models`

**Q: glm-5.2 返回空？**
glm-5.2 是推理模型，max_tokens 太小会返回空正文。tiku-adapter 默认不限制 max_tokens，一般不会触发。

**Q: 免费题库搜不到？**
icodef/buguake 是第三方公网服务，可能限流或失效。AI fallback 会兜底。
