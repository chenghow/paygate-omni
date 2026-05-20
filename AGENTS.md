# PayGate-Omni 聚合支付网关 - AI Agent 指导手册 (AGENTS.md)

本文件专为 AI 编码助手编写。在进行任何代码编写、重构或架构设计之前，**必须完整阅读并严格遵守**本手册定义的架构规范、安全守则及目录结构。

---

## 1. 项目愿景与定位
这是一个轻量、高性能、金融级安全的**个人聚合支付网关**。系统向下对接商户应用（提供统一的 SDK/API 接口），向上聚合支付宝（Alipay）和微信支付（WeChat Pay）的多种支付方式（App、H5、扫码、小程序）。
* **核心目标**：统一API标准、自动化路由、绝对的安全性、严格的幂等性处理、开箱即用的容器化本地开发环境。

---

## 2. 技术栈规范
AI Agent 在生成代码时，必须严格使用以下指定技术栈，严禁引入未经说明的第三方大重型框架：

* **后端 (Backend)**: 
  * 语言：Go 1.22+
  * Web 框架：`github.com/gin-gonic/gin` (轻量、高性能)
  * 支付 SDK：`github.com/go-pay/gopay` (统一封装微信V3/支付宝，处理签名与证书)
  * ORM 框架：`gorm.io/gorm` + `gorm.io/driver/postgres`
  * 缓存/分布式锁：`github.com/redis/go-redis/v9`
* **前端 (Frontend)**:
  * 框架：Vue 3 + Vite + TypeScript
  * UI 库：Element Plus (按需引入)
  * 状态管理/路由：Pinia + Vue Router
* **基础设施 (Infrastructure)**:
  * 数据库：PostgreSQL 16+ (严禁明文存储密钥，需应用层加密)
  * 缓存：Redis 7+ (用于防重放、分布式锁、状态缓存)
  * 反向代理/WAF：Nginx (负责 TLS 终止、Rate Limiting 限流)
  * 运行环境：Docker Compose 容器化编排 (完美兼容 WSL2 运行环境)

---

## 3. 标准目录结构
AI Agent 创建文件时必须严格遵循以下 Go 标准工程布局（Standard Go Project Layout）：

paygate-omni/
├── .github/workflwos/     # CI/CD 自动化流水线
├── backend/               # 后端主程序
│   ├── cmd/
│   │   └── server/        # 程序入口 (main.go)
│   ├── config/            # 配置解析 (Viper)
│   ├── internal/          # 内部核心业务逻辑 (不可被外部包导入)
│   │   ├── model/         # 数据库结构体 (Order, Channel, Merchant)
│   │   ├── repository/    # 数据库持久层 (DB/Redis 操作)
│   │   ├── service/       # 核心业务层 (支付路由、回调处理、对账)
│   │   ├── controller/    # HTTP 控制层 (路由 Handler)
│   │   └── middleware/    # 中间件 (签名验证、防重放、日志、CORS)
│   ├── pkg/               # 外部可导入的公共库 (如 crypto加密工具)
│   ├── go.mod
│   └── go.sum
├── frontend/              # 商户配置后台 (Vue3 + Vite)
│   ├── src/
│   │   ├── api/           # 请求封装
│   │   ├── views/         # 页面组件 (通道配置、订单列表)
│   │   ├── components/    # 通用组件
│   │   └── main.ts
│   ├── package.json
│   └── vite.config.ts
├── docker/                # 基础设施配置文件
│   ├── nginx/
│   │   └── nginx.conf     # Nginx 反向代理与限流配置
│   └── postgres/
└── docker-compose.yml     # 一键编排 (Go, Postgres, Redis, Nginx)

---

## 4. 核心安全与业务守则 (Security & Business Guardrails)

AI Agent 在编写业务逻辑时，涉及以下四点必须无条件满足：

### 4.1 敏感数据落盘加密 (Data at Rest)

* **要求**：商户的渠道私钥、微信 APIv3 Key、支付宝公钥等核心资产，**绝不允许明文写入数据库**。
* **实现**：必须在 `internal/model` 的 GORM 钩子（如 `BeforeSave`/`AfterFind`）或 Service 层中，使用 **AES-256-GCM** 算法进行加解密。
* **密钥管理**：AES 的主密钥（Master Key）必须从系统环境变量获取，严禁硬编码。

### 4.2 接口签名与防重放 (API Security & Anti-Replay)

* **要求**：网关面向商户（下游）的接口，必须实现严格的验签与防重放。
* **字段规范**：所有请求 Header 必须包含 `X-Pay-Timestamp` (时间戳)、`X-Pay-Nonce` (16位随机字符串) 和 `X-Pay-Signature` (SHA256带SK签名)。
* **校验逻辑**：
1. 检查 `Timestamp` 与网关当前时间差是否超过 5 分钟，超过则拒绝。
2. 将 `Nonce` 存入 Redis 并设置 5 分钟过期时间，若 Redis 中已存在该 Nonce，判定为重放攻击，直接拒绝。
3. 提取请求体结合商户的 SecretKey 计算 HMAC-SHA256 签名，与 `X-Pay-Signature` 比对。



### 4.3 严格的支付幂等性 (Idempotency)

* **要求**：第三方支付渠道（微信/支付宝）的回调通知存在“至少投递一次”的特性。网关的回调处理函数必须具备强幂等性。
* **实现逻辑**：
1. 收到回调后，首先使用 Redis 分布式锁拦截该订单号：`SET pay_lock:{out_trade_no} ex 10 nx`。若获取锁失败，返回错误触发渠道重试。
2. 进入数据库事务，查询该订单状态。若状态已经是 `SUCCESS` 或 `FAILED`，则直接释放锁并对渠道返回 `SUCCESS`（不再重复处理）。
3. 若状态为 `PENDING`，则继续更新状态、记录流水、触发下游商户的异步通知。



### 4.4 错误处理与日志规约

* **要求**：严禁在生产环境向外暴露包含数据库堆栈、敏感路径的底层错误。
* **实现**：核心逻辑需使用 Go 的 `fmt.Errorf("... %w", err)` 包装错误。在 Controller 层统一捕获，记录底层错误到结构化日志（如 Zap），向外只返回统一定义的错误码（如 `INTERNAL_SERVER_ERROR`）。

---

## 5. Agent 行为准则 (Prompt Instructions)

当用户要求你开发新功能或修复 Bug 时，请按以下步骤行动：

1. **先做设计确认**：在修改代码前，先向用户简述你的实现思路，特别说明会影响哪些文件、是否涉及数据库迁移、是否需要增加 Redis 缓存。
2. **渐进式重构**：不要一次性重写整块文件。优先完善 `internal/model` 和 `repository`，通过编译后再编写 `service` 和 `controller`。
3. **自带规范代码**：生成的 Go 代码必须通过 `go fmt` 标准格式化，变量命名遵循驼峰法（非首字母大写表示私有，首字母大写表示公有），函数必须包含简短的意图注释。
4. **日志埋点**：在支付生命周期的关键节点（收到下单、发起渠道请求、收到回调、验签成功/失败、通知商户），必须打印结构化日志，包含 `trade_no` 或 `out_trade_no` 关键字以便排查。
"""


## 6. 第三方渠道接入规范 (WeChat Pay V3)

### 6.1 SDK 使用规范
* 微信支付接入必须使用 `github.com/go-pay/gopay/wechat/v3` 包。
* HTTP Header 解析和验签使用 `wechat.V3ParseNotify` 和 `wechat.V3VerifySignByPK`。

### 6.2 凭证解密与客户端初始化
* **路由到渠道**：在下单服务中，必须根据 `Merchant.AppID` 以及渠道类型（`wechat`）从数据库查询启用的 `PayChannel` 配置。
* **脱敏即用**：由于 `internal/model.PayChannel` 的 `AfterFind` 钩子已自动将 `APIv3KeyEnc` 和 `PrivateKeyEnc` 解密至明文字段 `APIv3Key` 和 `PrivateKey`，在 Service 层可**直接读取**，严禁在业务代码中重复调用 AES 解密。
* **初始化 ClientV3**：使用解密后的明文凭证初始化 `wechat.NewClientV3(mchid, serialNo, apiV3Key, privateKey)`。商户证书序列号需要从数据库或配置中读取/解析，或自行配置。

### 6.3 回调幂等性处理 (Idempotency)
根据文档 4.3 规范，必须实现双重幂等机制：
1. **Redis 悲观锁**：以 `pay_lock:wechat:{out_trade_no}` 为 key，执行 `SetNX`（10秒过期），防止渠道的高并发重复投递。
2. **DB 状态机校验**：开启 DB 事务，查询 Order 状态是否为 `PENDING`。如果已是 `SUCCESS` 或 `FAILED`，则直接释放锁并向渠道回写 `SUCCESS` 应答。
