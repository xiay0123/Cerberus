# Cerberus

<div align="center">

**轻量级软件许可证管理系统**

面向客户端软件授权、设备绑定、在线验证与审计追踪的 Go License 管理服务。

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Non--Commercial-orange.svg)](#license)

</div>

---

## 简介

Cerberus 是一个基于 Go 开发的轻量级在线 License 管理系统，提供服务端、Web 管理后台、CLI 客户端和 Go SDK。

它适用于需要对客户端软件进行授权控制的场景，例如：商业软件激活、设备数量限制、License 在线验证、许可证续期/撤销、设备解绑和操作审计。

## 核心能力

- **License 管理**：创建、查询、续期、撤销、删除许可证
- **在线验证**：客户端实时请求服务端验证 License 状态
- **设备绑定**：基于机器指纹限制可激活设备数量
- **心跳监控**：客户端定期上报在线状态
- **自助换绑**：支持用户解绑旧设备，并限制换绑次数
- **IP 绑定**：可选的 IP 绑定校验能力
- **地理位置检测**：基于 IP 记录位置变化并支持异地策略
- **审计日志**：记录创建、验证、激活、续期、撤销等操作
- **管理后台**：内置 Web 控制台，支持许可证、设备、验证与统计查看
- **CLI / SDK**：提供命令行客户端和 Go SDK，便于接入业务系统
- **认证与限流**：支持 Admin Token / JWT 管理认证和请求限流

## 适用场景

- 商业软件 License 授权管理
- 客户端软件激活与在线验证
- 内部工具授权控制
- 设备数量限制与换绑管理
- 需要审计记录的授权系统

## 项目结构

```txt
Cerberus/
├── go.work                 # Go Workspace
├── server/                 # 服务端与 Web 管理后台
│   ├── main.go
│   ├── config.yaml
│   ├── web/
│   │   ├── index.html      # Web 管理后台
│   │   └── assets/         # Vue / Axios 静态资源
│   └── internal/
│       ├── config/         # 配置加载
│       ├── database/       # 数据库初始化
│       ├── engine/         # 业务引擎
│       ├── handler/        # HTTP 路由与接口
│       ├── middleware/     # 鉴权、限流、CORS 等中间件
│       ├── model/          # 数据模型
│       └── response/       # 响应格式
├── cli/                    # 命令行客户端
├── sdk/                    # Go SDK
├── pkg/                    # 公共包
│   ├── crypto/             # 加密相关
│   ├── fingerprint/        # 机器指纹采集
│   ├── geoip/              # 地理位置检测
│   ├── token/              # Token 处理
│   └── types/              # 公共类型
└── docs/                   # 使用文档
```

## 快速开始

### 环境要求

- Go 1.21+
- SQLite，无需单独安装服务端数据库

### 运行服务端

```bash
cd server
go run main.go
```

默认服务启动后可访问：

- Web 管理后台：`http://localhost:8080`
- API 服务：`http://localhost:8080`
- 健康检查：`http://localhost:8080/health`

### 配置说明

服务端配置文件位于 `server/config.yaml`。

示例配置：

```yaml
server:
  port: 8080
  mode: release

database:
  path: ./data/cerberus.db

license:
  max_machines: 3
  heartbeat_ttl: 5m

auth:
  admin_token: your-secure-token-change-me
  jwt_secret: your-jwt-secret-change-me
  jwt_ttl: 24h

geoip:
  enabled: true
  policy: alert
  allowed_distance: 500

rate_limit:
  enabled: true
  rps: 20
  burst: 40

cors_origins:
  - "*"
```

> 生产环境请务必修改 `admin_token` 和 `jwt_secret`。

## Web 管理后台

Cerberus 内置 Web 管理后台，无需额外构建前端项目。

后台功能包括：

- 仪表盘统计
- License 创建、续期、撤销、详情查看
- License Key 验证工具
- 设备绑定状态查看
- 即将过期许可证提醒
- 审计日志查看

启动服务端后，直接打开：

```txt
http://localhost:8080
```

使用 `server/config.yaml` 中配置的 `auth.admin_token` 登录。

## CLI 使用

### 构建 CLI

```bash
cd cli
go build -o cerberus-client .
```

### 常用命令

```bash
# 采集当前机器指纹
cerberus-client fingerprint

# 激活 License
cerberus-client activate --license <license-id> --server http://localhost:8080

# 验证 License
cerberus-client verify --license <license-id> --server http://localhost:8080

# 发送心跳
cerberus-client heartbeat --license <license-id> --server http://localhost:8080

# 自助换绑
cerberus-client unbind --license <license-id> --old-fingerprint <old-fingerprint> --server http://localhost:8080
```

也可以通过环境变量设置默认服务地址：

```bash
export CERBERUS_SERVER_URL=http://localhost:8080
cerberus-client verify --license <license-id>
```

## API 概览

### 公开接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/activate` | 激活设备 |
| `POST` | `/api/v1/verify` | 验证 License |
| `POST` | `/api/v1/heartbeat` | 心跳上报 |
| `POST` | `/api/v1/unbind` | 自助解绑 / 换绑 |

### 认证接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/auth/login` | 管理员登录，返回 JWT |

### 管理接口

管理接口需要在请求头中携带认证信息：

```http
Authorization: Bearer <admin-token-or-jwt>
```

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/licenses/create` | 创建 License |
| `POST` | `/api/v1/licenses/get` | 查询 License 详情 |
| `POST` | `/api/v1/licenses/list` | 查询 License 列表 |
| `POST` | `/api/v1/licenses/delete` | 删除 License |
| `POST` | `/api/v1/licenses/revoke` | 撤销 License |
| `POST` | `/api/v1/licenses/renew` | 续期 License |
| `POST` | `/api/v1/licenses/unsuspend` | 解除暂停 |
| `POST` | `/api/v1/licenses/audit` | 查询审计日志 |
| `POST` | `/api/v1/licenses/machines/revoke` | 解绑 / 撤销设备 |

## API 示例

### 创建 License

```bash
curl -X POST http://localhost:8080/api/v1/licenses/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d '{
    "name": "企业版授权",
    "product": "my-app",
    "duration_sec": 31536000,
    "max_machines": 3,
    "max_unbind_count": 5,
    "ip_binding_enabled": false
  }'
```

### 激活设备

```bash
curl -X POST http://localhost:8080/api/v1/activate \
  -H "Content-Type: application/json" \
  -d '{
    "license_id": "<license-id>",
    "fingerprint": "<machine-fingerprint>",
    "hostname": "my-device",
    "os": "windows",
    "arch": "amd64"
  }'
```

### 验证 License

```bash
curl -X POST http://localhost:8080/api/v1/verify \
  -H "Content-Type: application/json" \
  -d '{
    "license_id": "<license-id>",
    "fingerprint": "<machine-fingerprint>"
  }'
```

## 地理位置策略

Cerberus 支持基于 IP 的地理位置检测，可用于记录用户激活位置、心跳位置变化以及异地登录告警。

配置示例：

```yaml
geoip:
  enabled: true
  policy: alert
  allowed_distance: 500
```

策略说明：

| 策略 | 说明 |
| --- | --- |
| `allow` | 允许异地登录，仅记录日志 |
| `alert` | 允许异地登录，并记录告警 |
| `deny` | 拒绝异地登录 |

## 技术栈

| 模块 | 技术 |
| --- | --- |
| 服务端 | Go, Gin |
| 数据库 | SQLite, GORM |
| Web 管理后台 | Vue, Axios |
| CLI | Cobra |
| 配置 | Viper |
| 认证 | Admin Token, JWT |
| 指纹采集 | Windows / Linux / macOS |

## 开发检查

在仓库根目录运行服务端检查：

```bash
go test ./server/...
```

## License

Copyright © 2026 xiay. All rights reserved.

本项目仅允许用于个人学习、研究、评估和非商业用途。

未经作者 xiay 明确书面授权，任何个人、组织或公司不得将本项目或其衍生作品用于商业用途，包括但不限于：

- 用于商业软件、商业服务或付费产品
- 用于 SaaS、云服务、托管服务或 managed service
- 用于企业内部生产环境或对外提供服务的生产系统
- 二次开发后销售、分发、转授权或集成到商业产品中
- 任何直接或间接产生商业收益的使用场景

如需商业授权，请先联系作者 xiay 并获得书面许可。

本项目按“现状”提供，不提供任何明示或暗示的担保。作者不对因使用本项目造成的任何损失或风险承担责任。
