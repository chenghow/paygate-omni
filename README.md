# PayGate-Omni 聚合支付网关

![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)
![Vue Version](https://img.shields.io/badge/Vue-3.0-4FC08D.svg)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

PayGate-Omni 是一个轻量、高性能、金融级安全的**个人聚合支付网关**。系统向下对接商户应用（提供统一的 SDK/API 接口），向上聚合支付宝（Alipay）和微信支付（WeChat Pay）的多种支付方式。

本项目专为需要高度安全（敏感数据落盘加密）、严格并发处理（Redis 锁防重放与幂等回调）以及极简部署的开发者和运维人员设计。

## 🌟 核心特性与功能面板

- **多渠道聚合**：原生支持微信支付（V3）与支付宝支付，支持沙盒环境无缝切换。
- **全方位数据看板 (Dashboard)**：直观展示今日交易额、总订单量、接入商户数等核心运营指标统计图。
- **开箱即用**：前端（Vue3/Element Plus）+ 后端（Go 1.24/Gin）+ DB（PostgreSQL）+ 缓存（Redis）+ 代理限流（Nginx）全套 Docker Compose 容器编排。
- **金融级加密机制**：
  - 核心敏感配置（如商户密钥、API V3 Key、商户证书）在 PostgreSQL 中全部使用 **AES-256-GCM** 加密落盘，哪怕发生数据库拖库也无法还原可用密钥。
  - 下游 API 接口调用采用强签权机制（基于 HMAC-SHA256 的时间戳、16位随机数防重放攻击与请求体防篡改机制）。
- **极严谨的幂等与容错处理**：基于 Redis 悲观锁与 DB 状态机管理支付渠道的回调（Webhook），彻底杜绝因第三方网络波动重复通知导致的资金账目错误（无漏单、防重复记账）。

---

## 🚀 快速启动

本项目完全支持容器化，强烈推荐使用 Docker 一键部署所有微环境。

### 1. 环境准备
- 确保系统已部署最新版本 [Docker](https://docs.docker.com/get-docker/) 和 [Docker Compose](https://docs.docker.com/compose/install/)。
- 克隆本项目源码到本地机器。

### 2. 配置核心环境变量
由于网关具备高度安全的加密属性，必须要有一个外部应用主密钥。初始化时请复制我们提供的示范配置：
```bash
cp .env.example .env
```
*(注意：在将项目用于正式生产环境前，**必须**重置 `.env` 中的 `MASTER_KEY` [这是 AES 核心主加解密密钥] 以及数据库和缓存密码，并严格保密。)*

### 3. 一键编译与启动
在项目根目录下执行部署指令。Docker 会主动接管一切：自动下载环境依赖、自动构建执行 Go 后端服务、利用 Vite 编译打包前端 Web UI并装载至 Nginx：
```bash
docker compose up -d --build
```
*(因包含多阶段极简底包构建，初次执行可能需要数分钟，请耐心等待)*

### 4. 检查网关运行状态
```bash
docker compose ps
```
在此处确保后台（`backend`）、前端面版、底层数据库（`postgres`）和缓冲器（`redis`）服务状态均处于稳定的 `Up` 正常开启状态。

---

## 📖 管理后台运营指南

### 环节一：建立应用与通道准入配置
系统自带一套图形化控制端，用于管控下游使用方及对接的远程通道。

1. **登录控制面板**：
   本地直接访问机器地址 `http://localhost/`（Nginx 将隐切代理指向我们编译好前端视图）。
2. **新增商户 (Merchants 控制台)**：
   来到 **【商户管理】** 菜单栏。你可以为你的多个下游场景（如：你的博客商城、个人赞助计划）设立独立的商户账号。为其派发 `AppID` 与 `SecretKey`。界面中也为特定商户准备了实时的流水监控表！
3. **分配支付渠道 (Channels 控制台)**：
   跳往 **【渠道配置】**，新增微信或支付宝对接方式。开启**支付宝沙盒模式的开发者**，填入对应的官方测绘 AppID 等选项即可。这里注入的数据将被自动 AES 落盘加密。

### 环节二：订单审计与查询 (Orders)
在 **【订单管理】** 面板，你能可视化追查网关接管拦截到的所有交易。它将精准罗列外部订单追踪号 (`out_trade_no`) 并监控它如何在 `PENDING` -> `SUCCESS` 状态机之间实现安全流转。

### 环节三：API 模拟测试沙盒 (API Test)
前端工具箱附带了一个完全真实的终端模拟器：
1. 切入 **【API 测试】** 面版区。
2. 填入你在环节一设立的下游 商户 `AppID` 和 `SecretKey`。
3. 系统会在页面前端执行沙盒签名环境演练 —— `读取当前时间 Timestamp` -> `注入随机数 Nonce` -> `用密钥 SHA256 签名 Payload`，彻底省去了你手写加密 Demo 测试连通性的时间。立刻向你投递一个正式闭环的重定向支付链接(PayURL)。

---

## 💻 下游业务系统接入规范

外围商城、网站如何透过 PayGate-Omni 发起收揽款交易？请完全遵循我们的网关 RESTful 规范：

**通信网关请求端点**：`POST /api/v1/pay/order`

### 1. HTTP 鉴权请求头 (Headers)
必须把以下安全头部带入网关层，防止流量遭到横向抓包与伪造：
- `X-Pay-Appid`: 已经注册的商户应用 AppID
- `X-Pay-Timestamp`: 发起请求时的 Unix 10位时间戳（网关容忍误差±5分钟，超时直接丢弃）
- `X-Pay-Nonce`: 不可预测的 16位 随机字符串（一旦被 Redis 网关锁住记录，5分钟内全量排斥重放攻击）
- `X-Pay-Signature`: 进行签名的哈希散列。机制为：将 `Timestamp` + `Nonce` + HTTP 原生 `JSON Body 字符串` 打包，以该商户的 `SecretKey` 进行 HMAC-SHA256 结算后的 HEX 值。

### 2. 标准支付请求体 (JSON)
```json
{
  "out_trade_no": "ORDER-20260520-22119",   // 外部系统独立业务结算号 (必须保持系统内唯一)
  "amount": 100,                            // 以“分”为单位传输金额 (100 = 1.00元)
  "channel_type": "alipay",                 // 固定枚举参数：alipay | wechat
  "subject": "网站高级会员服务",              // 直接映射到消费者账单端的产品摘要
  "client_ip": "123.123.123.123",           // 下游消费者的真实地理边界IP (风控与原生通道要求)
  "notify_url": "https://xyz.com/webhook"   // 消费者打款成功后，PayGate异步投递成功消息的网络地址
}
```

---

## 🛠 开发目录结构

```text
paygate-omni/
├── backend/               # 纯粹使用 Go 进行支撑的后端
│   ├── cmd/server/        # 程序引导口与依赖参数解析
│   ├── internal/          # 受限边界划分的领域业务模型
│   │   ├── controller/    # 路由接管模块、数据结构映射验证脱水层
│   │   ├── middleware/    # Auth鉴权环锁、防重放屏护中间件
│   │   ├── model/         # GORM 映射对象库，附加全自动AES字段编解码钩子
│   │   └── repository/    # 抽象底层：PostgresSQL与高速Redis状态操作机
├── frontend/              # 采用 Vue3 + Element Plus 的面板套件
│   ├── src/views/         # 各可视视图组件封装
│   └── src/api/           # 封装的 Axios API 调用信使
├── docker/                # 预装组件库与基础组装线
│   └── nginx/             # 预置反向分发与请求限制脚本
├── AGENTS.md              # AI Agent 自动化设计纲领和指导准则
├── .env.example           # 应用核心配置字典样本
└── docker-compose.yml     # 无代码级别的集群启停枢纽文件
```

---

## 🛡️ 投产必看指导 (Production Deployment Guide)

1. **绝对隔离保护 MASTER_KEY**：本系统全部基于加密信任根运转。在服务端初始建设时设防的 `MASTER_KEY` **极其关键**！投产后如果遗忘或环境突变丢失该变量，将导致**历史所有通道密钥彻底无法解译！因此务必建立安全备份或经由云端 KMS 服务下发指令**。
2. **强制推行 HTTPS 证书**：目前的开源形态下暴露 80 / HTTP 端口，但在真实的金融环境内明朗的 HTTP 环境极易面临链路劫持。生产中务必修改 Nginx 设定并装载合规强健的 SSL 证书。
3. **数据灾备规划**：在 `.gitignore` 保护下这些数据并未上云，需系统管理人员主动部署脚本定周期通过 `pg_dump` 对 Postgres 以及 AOF 文件对 Redis 进行隔离归档备份。
