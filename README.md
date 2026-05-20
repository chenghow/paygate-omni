# PayGate-Omni 聚合支付网关

PayGate-Omni 是一个轻量、高性能、金融级安全的个人聚合支付网关。系统向下对接商户应用（统一 API），向上聚合支付宝与微信支付。

## 核心特性

- 多渠道聚合：支持支付宝、微信支付。
- 安全优先：敏感配置采用 AES-256-GCM 落盘加密。
- 防重放与幂等：基于 Redis Nonce 与分布式锁保障回调安全。
- 容器化部署：Docker Compose 一键启动后端、前端、PostgreSQL、Redis。

## 架构说明（已移除 Nginx）

当前仓库默认不包含 Nginx 服务，采用前后端直连：

- 前端管理后台：`http://localhost:3000`
- 后端 API：`http://localhost:8080`
- PostgreSQL：容器内网络访问
- Redis：容器内网络访问

说明：
- `docker-compose.yml` 中不再包含 `nginx` 服务。
- 反向代理与 HTTPS 证书由使用者在自己的部署环境中自行配置（如 Nginx/Caddy/Traefik/云负载均衡）。

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

访问地址：

- 后台：`http://localhost:3000`
- 健康检查：`http://localhost:8080/health`

## 管理后台使用

1. 打开 `http://localhost:3000`
2. 使用 `.env` 中的 `ADMIN_PASSWORD` 登录
3. 在商户管理中新增商户
4. 在渠道配置中绑定微信/支付宝参数
5. 在订单管理和总览页面查看数据

## API 接入示例

支付下单接口：`POST /api/v1/pay/order`

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

## 生产环境部署说明（反向代理与 HTTPS 由你自行配置）

本项目不内置 HTTPS 与网关服务，生产环境请自行配置：

1. 反向代理：将外部 `443` 请求转发到前端 `3000` 与后端 `8080`。
2. TLS 证书：使用 Let's Encrypt 或云厂商证书，强制启用 HTTPS。
3. 安全头：建议添加 HSTS、X-Forwarded-Proto、X-Real-IP 等头。
4. 访问控制：建议限制后台登录入口 IP 段，开启 WAF/限流。

参考路由策略：

- `/` -> `frontend:3000`
- `/api/` -> `backend:8080`

## 项目目录

```text
paygate-omni/
├── backend/
├── frontend/
├── docker/
│   └── postgres/
├── .env.example
├── docker-compose.yml
└── README.md
```

## 安全注意事项

- 不要将 `.env`、数据库数据、私钥证书提交到 Git。
- `MASTER_KEY` 丢失会导致历史加密数据无法解密。
- 生产环境务必启用 HTTPS，避免明文传输管理密码与支付数据。

