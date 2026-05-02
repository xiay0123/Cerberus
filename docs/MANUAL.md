# Cerberus 操作手册

## 目录

1. [系统概述](#1-系统概述)
2. [部署指南](#2-部署指南)
3. [配置详解](#3-配置详解)
4. [服务端操作](#4-服务端操作)
5. [客户端使用](#5-客户端使用)
6. [SDK 集成](#6-sdk-集成)
7. [运维管理](#7-运维管理)
8. [故障排查](#8-故障排查)

---

## 1. 系统概述

### 1.1 什么是 Cerberus

Cerberus 是一个轻量级的软件授权管理系统，提供：

- **License 管理**：创建、查询、吊销、续期许可证
- **机器绑定**：基于硬件指纹限制授权机器数量
- **在线验证**：实时验证 License 有效性
- **心跳监控**：监控客户端在线状态
- **地理位置检测**：检测异地登录并告警
- **审计日志**：记录所有操作历史

### 1.2 系统架构

```
┌─────────────────┐     HTTP API      ┌─────────────────┐
│   客户端应用    │ ◄───────────────► │   Cerberus      │
│  (集成 SDK)     │                   │   Server        │
└─────────────────┘                   └────────┬────────┘
                                               │
                                               ▼
                                      ┌─────────────────┐
                                      │    SQLite       │
                                      │   数据库        │
                                      └─────────────────┘
```

### 1.3 核心流程

#### 激活流程

```
用户首次启动应用
       │
       ▼
采集机器指纹 (CPU/磁盘/主板/MAC)
       │
       ▼
向服务端发送激活请求
       │
       ▼
服务端验证 License 有效性
       │
       ▼
检查机器数量限制
       │
       ▼
记录机器信息，返回成功
```

#### 验证流程

```
应用启动或定期检查
       │
       ▼
向服务端发送验证请求
       │
       ▼
服务端检查 License 状态
       │
       ├─ 有效 ──► 返回验证成功
       │
       └─ 无效 ──► 返回失败原因
                    (吊销/过期/机器不匹配)
```

---

## 2. 部署指南

### 2.1 环境要求

| 项目 | 要求 |
|------|------|
| 操作系统 | Windows / Linux / macOS |
| Go 版本 | 1.21 或更高 |
| 内存 | 最低 64MB |
| 磁盘 | 最低 100MB（数据库会增长） |

### 2.2 从源码构建

```bash
# 克隆仓库
git clone https://github.com/your-org/cerberus.git
cd cerberus

# 构建服务端
cd server
go build -o cerberus-server .

# 构建 CLI 客户端
cd ../cli
go build -o cerberus-client .
```

### 2.3 目录结构准备

```bash
# 创建必要目录
mkdir -p data keys

# 目录结构
cerberus/
├── cerberus-server     # 服务端可执行文件
├── cerberus-client     # CLI 客户端
├── config.yaml         # 配置文件
├── data/               # 数据库目录
│   └── cerberus.db     # SQLite 数据库（自动创建）
└── keys/               # 密钥目录（可选）
```

### 2.4 生成配置文件

创建 `config.yaml`：

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

### 2.5 启动服务

```bash
# 前台运行
./cerberus-server

# 后台运行（Linux/macOS）
nohup ./cerberus-server > cerberus.log 2>&1 &

# 后台运行（Windows，使用 PowerShell）
Start-Process -NoNewWindow ./cerberus-server
```

### 2.6 验证部署

```bash
# 健康检查
curl http://localhost:8080/health

# 预期输出
{"status":"ok"}
```

---

## 3. 配置详解

### 3.1 服务器配置 (server)

```yaml
server:
  port: 8080           # 监听端口
  mode: release        # 运行模式：debug / release
  read_timeout: 10s    # 读取超时
  write_timeout: 10s   # 写入超时
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| port | int | 8080 | HTTP 服务监听端口 |
| mode | string | release | debug 启用日志输出，release 静默运行 |
| read_timeout | duration | 10s | 请求读取超时时间 |
| write_timeout | duration | 10s | 响应写入超时时间 |

### 3.2 数据库配置 (database)

```yaml
database:
  path: ./data/cerberus.db
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| path | string | ./data/cerberus.db | SQLite 数据库文件路径 |

### 3.3 License 配置 (license)

```yaml
license:
  max_machines: 3        # 默认最大机器数
  heartbeat_ttl: 5m      # 心跳超时时间
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| max_machines | int | 3 | 每个 License 默认最大绑定机器数 |
| heartbeat_ttl | duration | 5m | 心跳超时时间，超过后机器状态变为 stale |

### 3.4 认证配置 (auth)

```yaml
auth:
  admin_token: your-secure-token-change-me
  jwt_secret: your-jwt-secret-change-me
  jwt_ttl: 24h
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| admin_token | string | cerberus-admin-token-change-me | 管理员令牌，**生产环境必须修改** |
| jwt_secret | string | cerberus-jwt-secret-change-me | JWT 签名密钥，**生产环境必须修改** |
| jwt_ttl | duration | 24h | JWT 有效期 |

### 3.5 地理位置检测配置 (geoip)

```yaml
geoip:
  enabled: true
  policy: alert
  allowed_distance: 500
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| enabled | bool | false | 是否启用地理位置检测 |
| policy | string | allow | 检测策略：allow / alert / deny |
| allowed_distance | float | 500 | 允许的最大距离（公里） |

**策略说明：**

| 策略 | 行为 |
|------|------|
| allow | 允许异地登录，仅记录日志 |
| alert | 异地登录时返回告警信息但允许操作 |
| deny | 拒绝异地登录，验证失败 |

### 3.6 限流配置 (rate_limit)

```yaml
rate_limit:
  enabled: true
  rps: 20
  burst: 40
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| enabled | bool | true | 是否启用限流 |
| rps | int | 20 | 每秒允许的请求数 |
| burst | int | 40 | 突发请求数上限 |

### 3.7 CORS 配置 (cors_origins)

```yaml
cors_origins:
  - "*"                    # 允许所有来源（不推荐生产使用）
  # 或者指定具体域名
  # - "https://example.com"
  # - "https://admin.example.com"
```

---

## 4. 服务端操作

### 4.1 创建 License

**请求：**

```bash
curl -X POST http://localhost:8080/api/v1/licenses/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{
    "name": "企业版授权",
    "product": "my-app",
    "issuer": "admin",
    "duration_sec": 31536000,
    "max_machines": 5,
    "max_unbind_count": 5,
    "ip_binding_enabled": false
  }'
```

**参数说明：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | License 名称（显示用） |
| product | string | 是 | 产品标识 |
| issuer | string | 否 | 发行者名称 |
| duration_sec | int64 | 是 | 有效时长（秒），如 31536000 = 1 年 |
| max_machines | int | 否 | 最大机器数，默认使用配置值 |
| max_unbind_count | int | 否 | 最大换绑次数，默认 3 |
| ip_binding_enabled | bool | 否 | 是否启用 IP 绑定，默认 false |

**响应示例：**

```json
{
  "code": 0,
  "message": "created",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "企业版授权",
    "product": "my-app",
    "issuer": "admin",
    "max_machines": 5,
    "valid_from": 1693526400,
    "valid_until": 1725062400,
    "status": "active"
  }
}
```

### 4.2 查询 License 详情

```bash
curl -X POST http://localhost:8080/api/v1/licenses/get \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{"id": "550e8400-e29b-41d4-a716-446655440000"}'
```

### 4.3 查询 License 列表

```bash
curl -X POST http://localhost:8080/api/v1/licenses/list \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{
    "page": 1,
    "size": 20,
    "status": "active",
    "product": "my-app"
  }'
```

**参数说明：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，从 1 开始 |
| size | int | 每页数量，最大 100 |
| status | string | 状态筛选：active / revoked / expired / suspended |
| product | string | 产品筛选 |

### 4.4 续期 License

```bash
curl -X POST http://localhost:8080/api/v1/licenses/renew \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "duration_sec": 31536000
  }'
```

### 4.5 吊销 License

```bash
curl -X POST http://localhost:8080/api/v1/licenses/revoke \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{"id": "550e8400-e29b-41d4-a716-446655440000"}'
```

### 4.6 重新启用已吊销 License

重新启用会将 License 恢复为 active，并同步恢复该 License 下已吊销的机器，不会重置换绑计数。

```bash
curl -X POST http://localhost:8080/api/v1/licenses/reactivate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{"id": "550e8400-e29b-41d4-a716-446655440000"}'
```

### 4.7 解封 License

当 License 因换绑次数超限被暂停时：

```bash
curl -X POST http://localhost:8080/api/v1/licenses/unsuspend \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "reset_unbind_count": true
  }'
```

### 4.8 吊销机器

管理员可以主动吊销单个机器：

```bash
curl -X POST http://localhost:8080/api/v1/licenses/machines/revoke \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{
    "license_id": "550e8400-e29b-41d4-a716-446655440000",
    "machine_id": "660e8400-e29b-41d4-a716-446655440001"
  }'
```

### 4.9 查询审计日志

```bash
curl -X POST http://localhost:8080/api/v1/licenses/audit \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-admin-token" \
  -d '{"id": "550e8400-e29b-41d4-a716-446655440000"}'
```

**响应示例：**

```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": "xxx",
      "license_id": "550e8400-e29b-41d4-a716-446655440000",
      "action": "activate",
      "detail": "machine activated: abc123...",
      "ip": "192.168.1.100",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

## 5. 客户端使用

### 5.1 CLI 工具安装

```bash
cd cli
go build -o cerberus-client .
```

### 5.2 采集机器指纹

```bash
cerberus-client fingerprint
```

**输出示例：**

```json
{
  "fingerprint": "a1b2c3d4e5f6...",
  "hostname": "MyPC",
  "os": "windows",
  "arch": "amd64"
}
```

### 5.3 激活 License

```bash
# 使用自动采集的指纹激活
cerberus-client activate \
  --license 550e8400-e29b-41d4-a716-446655440000 \
  --server http://localhost:8080

# 使用指定指纹激活（测试用）
cerberus-client activate \
  --license 550e8400-e29b-41d4-a716-446655440000 \
  --server http://localhost:8080 \
  --fingerprint custom-fingerprint
```

**输出示例：**

```json
{
  "machine": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "license_id": "550e8400-e29b-41d4-a716-446655440000",
    "fingerprint": "a1b2c3d4e5f6...",
    "status": "active"
  },
  "is_new_machine": true
}
```

### 5.4 验证 License

```bash
cerberus-client verify \
  --license 550e8400-e29b-41d4-a716-446655440000 \
  --server http://localhost:8080
```

**输出示例：**

```json
{
  "valid": true,
  "license_id": "550e8400-e29b-41d4-a716-446655440000",
  "product": "my-app",
  "expires_in": 31536000,
  "max_machines": 5
}
```

### 5.5 发送心跳

建议在应用程序运行期间定期发送心跳（如每 5 分钟）：

```bash
cerberus-client heartbeat \
  --license 550e8400-e29b-41d4-a716-446655440000 \
  --server http://localhost:8080
```

### 5.6 自助换绑

当需要在新设备上使用 License 时，可以先解绑旧设备：

```bash
# 先查看已绑定的机器指纹
cerberus-client fingerprint

# 解绑旧设备（需要旧设备的指纹）
cerberus-client unbind \
  --license 550e8400-e29b-41d4-a716-446655440000 \
  --old-fingerprint old-device-fingerprint \
  --server http://localhost:8080
```

**输出示例：**

```json
{
  "success": true,
  "machine_revoked": "660e8400-e29b-41d4-a716-446655440001",
  "remaining": 2,
  "message": "machine unbound successfully, 2 unbind operations remaining"
}
```

**注意：每个 License 有换绑次数限制，超限后 License 会被暂停！**

### 5.7 使用环境变量

设置默认服务器地址，避免每次输入：

```bash
# Linux/macOS
export CERBERUS_SERVER_URL=http://localhost:8080

# Windows PowerShell
$env:CERBERUS_SERVER_URL = "http://localhost:8080"

# Windows CMD
set CERBERUS_SERVER_URL=http://localhost:8080

# 之后可以省略 --server 参数
cerberus-client verify --license 550e8400-...
```

---

## 6. SDK 集成

### 6.1 安装 SDK

```bash
go get cerberus.dev/sdk
```

### 6.2 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "cerberus.dev/sdk"
)

func main() {
    // 创建客户端
    client := sdk.NewClient("http://localhost:8080")
    ctx := context.Background()
    licenseID := "550e8400-e29b-41d4-a716-446655440000"

    // 激活（自动采集指纹）
    activateResp, err := client.Activate(ctx, licenseID, sdk.WithFingerprintAuto())
    if err != nil {
        log.Fatalf("激活失败: %v", err)
    }
    fmt.Printf("激活成功: %s\n", activateResp.Machine.ID)

    // 验证
    verifyResult, err := client.Verify(ctx, licenseID, sdk.WithFingerprintAuto())
    if err != nil {
        log.Fatalf("验证失败: %v", err)
    }
    if verifyResult.Valid {
        fmt.Printf("License 有效，剩余 %d 秒\n", verifyResult.ExpiresIn)
    } else {
        fmt.Printf("License 无效: %s\n", verifyResult.Reason)
    }
}
```

### 6.3 使用指定指纹

```go
// 使用自定义指纹（测试场景）
result, err := client.Verify(ctx, licenseID, sdk.WithFingerprint("custom-fingerprint"))
```

### 6.4 心跳循环

```go
package main

import (
    "context"
    "log"
    "time"

    "cerberus.dev/sdk"
)

func main() {
    client := sdk.NewClient("http://localhost:8080")
    ctx := context.Background()
    licenseID := "550e8400-e29b-41d4-a716-446655440000"

    // 每 5 分钟发送一次心跳
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if err := client.Heartbeat(ctx, licenseID, sdk.WithFingerprintAuto()); err != nil {
                log.Printf("心跳失败: %v", err)
            } else {
                log.Println("心跳成功")
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### 6.5 错误处理

```go
result, err := client.Verify(ctx, licenseID, sdk.WithFingerprintAuto())
if err != nil {
    // API 错误
    log.Printf("API 调用失败: %v", err)
    return
}

if !result.Valid {
    // License 无效
    switch result.Reason {
    case "license revoked":
        // License 被吊销
    case "license expired":
        // License 已过期
    case "machine not activated or revoked":
        // 机器未激活或已被吊销
    case "IP mismatch":
        // IP 地址不匹配
    default:
        // 其他原因
    }
}
```

---

## 7. 运维管理

### 7.1 数据备份

```bash
# SQLite 数据库是单文件，直接复制即可备份
cp data/cerberus.db backups/cerberus-$(date +%Y%m%d).db

# 定时备份脚本（crontab）
# 每天凌晨 2 点备份
0 2 * * * cp /path/to/cerberus/data/cerberus.db /path/to/backups/cerberus-$(date +\%Y\%m\%d).db
```

### 7.2 日志管理

```bash
# 重定向日志到文件
./cerberus-server >> logs/cerberus.log 2>&1

# 日志轮转（使用 logrotate）
# /etc/logrotate.d/cerberus
/path/to/cerberus/logs/cerberus.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
}
```

### 7.3 监控健康状态

```bash
# 健康检查脚本
#!/bin/bash
HEALTH_URL="http://localhost:8080/health"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)

if [ "$RESPONSE" != "200" ]; then
    echo "Cerberus 服务异常！HTTP 状态码: $RESPONSE"
    # 发送告警...
    exit 1
fi

echo "Cerberus 服务正常"
```

### 7.4 性能优化

**数据库优化：**

```bash
# 定期执行 VACUUM 清理空间
sqlite3 data/cerberus.db "VACUUM;"

# 分析查询性能
sqlite3 data/cerberus.db "ANALYZE;"
```

**配置优化：**

```yaml
# 高并发场景
rate_limit:
  enabled: true
  rps: 100
  burst: 200

# 长时间运行场景
license:
  heartbeat_ttl: 30m  # 延长心跳超时
```

---

## 8. 故障排查

### 8.1 常见错误

#### License 激活失败

| 错误信息 | 原因 | 解决方案 |
|----------|------|----------|
| license not found or not active | License 不存在或已吊销 | 检查 License ID，确认 License 状态 |
| license expired | License 已过期 | 联系管理员续期 |
| max machines (X) reached | 已达到最大机器数 | 换绑或管理员吊销旧机器 |

#### 验证失败

| 错误信息 | 原因 | 解决方案 |
|----------|------|----------|
| machine not activated or revoked | 机器未激活或已吊销 | 先执行激活操作 |
| IP mismatch | IP 地址与绑定 IP 不符 | 关闭 IP 绑定或从原 IP 访问 |
| license revoked | License 已被吊销 | 联系管理员 |
| license expired | License 已过期 | 联系管理员续期 |

#### 换绑失败

| 错误信息 | 原因 | 解决方案 |
|----------|------|----------|
| machine not found or already revoked | 机器不存在或已吊销 | 检查指纹是否正确 |
| unbind limit exceeded | 换绑次数超限 | 联系管理员解封 License |
| license is suspended | License 已被暂停 | 联系管理员解封 |

### 8.2 日志分析

```bash
# 查看最近错误
tail -100 logs/cerberus.log | grep -i error

# 统计 API 调用
grep -o "\[POST\]" logs/cerberus.log | wc -l

# 查看特定 License 操作
grep "550e8400-e29b-41d4-a716-446655440000" logs/cerberus.log
```

### 8.3 数据库诊断

```bash
# 检查数据库完整性
sqlite3 data/cerberus.db "PRAGMA integrity_check;"

# 查看 License 状态
sqlite3 data/cerberus.db "SELECT id, name, status, valid_until FROM licenses;"

# 查看机器绑定
sqlite3 data/cerberus.db "SELECT license_id, fingerprint, status FROM machines WHERE license_id = 'xxx';"

# 查看审计日志
sqlite3 data/cerberus.db "SELECT * FROM audit_logs WHERE license_id = 'xxx' ORDER BY created_at DESC LIMIT 10;"
```

### 8.4 网络诊断

```bash
# 检查端口监听
netstat -tlnp | grep 8080

# 检查服务可达性
curl -v http://localhost:8080/health

# 检查防火墙
# Linux
iptables -L -n | grep 8080
# Windows
netsh advfirewall firewall show rule name=all | findstr 8080
```

---

## 附录

### A. 时间转换参考

| 时长 | 秒数 |
|------|------|
| 1 天 | 86400 |
| 1 周 | 604800 |
| 1 月（30天） | 2592000 |
| 1 年（365天） | 31536000 |

### B. 状态码说明

| HTTP 状态码 | 说明 |
|-------------|------|
| 200 | 成功 |
| 201 | 资源创建成功 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |

### C. License 状态说明

| 状态 | 说明 |
|------|------|
| active | 活跃，正常使用 |
| revoked | 已吊销，不可恢复 |
| expired | 已过期，可续期 |
| suspended | 已暂停，可解封 |

### D. Machine 状态说明

| 状态 | 说明 |
|------|------|
| active | 活跃，正常使用 |
| revoked | 已吊销 |
| stale | 长时间无心跳 |
