# PayGate-Omni 聚合支付网关

PayGate-Omni 是一个轻量、高性能、金融级安全的个人聚合支付网关。系统向下对接商户
应用（统一 API），向上聚合支付宝与微信支付。

## 核心特性

- 多渠道聚合：支持支付宝、微信支付。
- 安全优先：敏感配置采用 AES-256-GCM 落盘加密。
- 防重放与幂等：基于 Redis Nonce 与分布式锁保障回调安全。
- 前后端整合：前端和后端集成为单一服务，简化部署。
- 容器化部署：Docker Compose 一键启动，包含 PostgreSQL 和 Redis。

## 架构说明

当前架构为前后端整合设计，采用单一服务模式：

- **统一入口**：`http://localhost:8080`
  - 前端管理后台由后端直接提供（`/` 路径）
  - API 接口统一在 `/api/v1` 路径下
- **数据库**：PostgreSQL（容器内网络访问）
- **缓存**：Redis（容器内网络访问）

优势：
- 不需要前后端分别部署
- 避免跨域问题
- 单一反向代理配置
- 简化容器编排

## 快速启动

### 1. 环境准备

确保本机已安装：

- Docker
- Docker Compose

### 2. 初始化环境变量

```bash
cp .env.example .env
```

必须修改以下关键项：

- `MASTER_KEY`（必须 32 字节，且生产中严禁随意更换）
- `ADMIN_PASSWORD`
- `DB_PASSWORD`

### 3. 启动服务

```bash
docker compose up -d --build
```

### 4. 检查服务

```bash
docker compose ps
```

### 5. 访问系统

- **管理后台**：`http://localhost:8080`
- **健康检查**：`http://localhost:8080/health`
- **API**：`http://localhost:8080/api/v1/*`

## 管理后台使用

1. 打开浏览器，访问 `http://localhost:8080`
2. 使用 `.env` 中配置的 `ADMIN_PASSWORD` 登录
3. 在商户管理中新增商户
4. 在渠道配置中绑定微信/支付宝参数
5. 在订单管理和总览页面查看数据

## API 接入示例

支付下单接口：`POST /api/v1/pay/create`

请求头必须包含：

- `X-Pay-Appid`
- `X-Pay-Timestamp`
- `X-Pay-Nonce`
- `X-Pay-Signature`

示例请求体：

```json
{
  "out_trade_no": "ORDER20260520112233",
  "amount": 100,
  "channel_type": "alipay",
  "subject": "测试商品购买",
  "client_ip": "127.0.0.1",
  "notify_url": "https://your-shop.com/pay_callback"
}
```

## 易支付兼容接入

如果外部网站或插件按易支付风格对接，可以直接使用以下入口：

- 下单地址：`http://localhost:8080/submit.php`
- 统一接口：`http://localhost:8080/api.php`

常用参数：

- `pid`：商户后台创建的 `AppID`
- `key`：商户后台配置的 `SecretKey`
- `type`：`alipay` 或 `wechat` / `wxpay`
- `out_trade_no`：外部网站订单号
- `notify_url`：外部网站回调地址
- `return_url`：支付完成后的跳转地址，可选
- `name`：商品名
- `money`：金额，单位元

签名规则：

- `sign_type=MD5`
- 参与签名的参数按字段名排序，拼接为 `k=v`，最后追加 `key=商户密钥`
- 计算结果与 `sign` 比对

查询订单时可调用：

- `http://localhost:8080/api.php?act=query`

说明：

- 这个兼容层保留了项目内部的 `/api/v1` 接口，不影响现有管理后台和自研接入方式。
- 如果你的上游系统已经按易支付插件开发，优先使用 `submit.php` 和 `api.php`。

## 生产环境部署说明

本项目不内置反向代理和 HTTPS，生产环境需自行配置：

### 1. 反向代理配置

推荐使用 Nginx、Caddy 或云厂商负载均衡，转发到后端 8080 端口：

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    
    location / {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. HTTPS 配置

使用 Let's Encrypt 或云厂商证书：

```bash
# Certbot 示例
certbot certonly --webroot -w /var/www/yourdomain -d yourdomain.com
```

### 3. 安全建议

- 启用 HTTPS，设置 HSTS 头
- 限制管理后台访问 IP 段
- 启用 WAF、限流、DDoS 防护
- 定期备份 `.env` 和数据库数据

## 项目结构

```text
paygate-omni/
├── backend/              # Go 后端源码
│   ├── cmd/server/       # 程序入口
│   ├── config/           # 配置解析
│   ├── internal/         # 核心业务逻辑
│   │   ├── model/        # 数据模型
│   │   ├── service/      # 业务服务
│   │   ├── controller/   # HTTP 控制器
│   │   ├── repository/   # 数据持久化
│   │   └── middleware/   # 中间件
│   ├── pkg/              # 公共库（加密等）
│   ├── Dockerfile        # 三阶段构建：前端编译 + 后端编译 + 运行镜像
│   ├── go.mod
│   └── go.sum
├── frontend/             # Vue 3 前端源码
│   ├── src/
│   │   ├── api/          # API 调用
│   │   ├── views/        # 页面
│   │   ├── components/   # 组件
│   │   └── router/       # 路由
│   ├── package.json
│   ├── vite.config.ts
│   └── Dockerfile        # 构建前端（已集成到后端 Dockerfile）
├── docker/               # 辅助文件
│   └── postgres/
│       └── init.sql      # 数据库初始化
├── docker-compose.yml    # 容器编排：postgres, redis, backend
├── .env.example          # 环境变量模板
├── .gitignore
└── README.md
```

## 数据库初始化

PostgreSQL 自动执行 `docker/postgres/init.sql` 初始化脚本。如需手动执行：

```bash
docker exec paygate_postgres psql -U paygate -d paygate_omni -f /docker-entrypoint-initdb.d/init.sql
```

## 常见问题

**Q: 如何更改管理员密码？**
A: 修改 `.env` 中的 `ADMIN_PASSWORD`，重启后端服务 `docker compose restart backend`。

**Q: 前端无法加载？**
A: 确保后端服务正常运行：`docker logs paygate_backend`。检查 `frontend/dist` 目录是否存在。

**Q: 无法连接数据库？**
A: 确保 `MASTER_KEY` 配置正确，长度必须为 32 字节。检查 `.env` 中的数据库配置。

## 安全注意事项

- 不要将 `.env` 文件、数据库文件、私钥提交到 Git
- `MASTER_KEY` 丢失会导致所有历史加密数据无法解密，请妥善保管
- 生产环境务必启用 HTTPS，避免明文传输管理密码和支付数据
- 定期更新 Docker 镜像和依赖版本

## License

MIT
