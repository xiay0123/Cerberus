# Cerberus 软件授权管理系统使用手册

> 适用对象：系统管理员、运维人员、软件发行方、客户端集成开发者。  
> 文档范围：服务部署、后台使用、许可证管理、设备绑定、CLI 调用、API 调用、Go SDK 集成、运维与故障排查。

---

## 目录

1. [系统简介](#1-系统简介)
2. [核心概念](#2-核心概念)
3. [快速启动](#3-快速启动)
4. [配置说明](#4-配置说明)
5. [Web 管理后台使用](#5-web-管理后台使用)
6. [许可证完整生命周期](#6-许可证完整生命周期)
7. [客户端 CLI 使用](#7-客户端-cli-使用)
8. [HTTP API 使用](#8-http-api-使用)
9. [Go SDK 集成](#9-go-sdk-集成)
10. [典型接入流程](#10-典型接入流程)
11. [运维管理](#11-运维管理)
12. [安全建议](#12-安全建议)
13. [故障排查](#13-故障排查)
14. [附录：常用命令速查](#14-附录常用命令速查)

---

## 1. 系统简介

Cerberus 是一个轻量级软件许可证管理系统，用于给客户端软件提供在线授权、设备绑定、心跳上报、验证审计和后台管理能力。

系统由四部分组成：

| 模块      | 路径        | 用途                               |
|---------|-----------|----------------------------------|
| 服务端     | `server/` | 提供 HTTP API、Web 管理后台、SQLite 数据存储 |
| CLI 客户端 | `cli/`    | 命令行方式激活、验证、心跳、解绑、采集机器指纹          |
| Go SDK  | `sdk/`    | 在 Go 应用中直接集成授权验证能力               |
| 公共包     | `pkg/`    | 机器指纹、GeoIP、Token、类型定义等公共能力       |

### 1.1 系统能力

- 许可证创建、查询、续期、撤销、重新启用、删除。
- 按许可证限制最大绑定设备数量。
- 客户端在线激活和在线验证。
- 客户端心跳上报，用于记录最后活跃时间。
- 用户自助解绑或换绑，并可限制最大解绑次数。
- 可选 IP 绑定验证。
- 可选 GeoIP 异地检测策略。
- 后台可查看产品分布、状态分布、许可证列表、设备总览、操作审计。
- CLI 输出 JSON，适合 Java、Python、Electron、C#、Shell、安装器等程序通过子进程集成。

### 1.2 系统架构

```text
┌─────────────────────┐
│   软件发行方/管理员  │
│   Web 管理后台       │
└──────────┬──────────┘
           │ Admin Token / JWT
           ▼
┌────────────────────────────────────┐
│          Cerberus Server            │
│  - License 管理 API                 │
│  - 客户端公开 API                   │
│  - Web Console                      │
│  - 审计日志                         │
└──────────┬─────────────────────────┘
           │
           ▼
┌─────────────────────┐
│       SQLite         │
│  License / Machine   │
│  Audit Log           │
└─────────────────────┘
           ▲
           │ Public API
┌──────────┴──────────┐
│     客户端软件       │
│  CLI / SDK / HTTP    │
└─────────────────────┘
```

---

## 2. 核心概念

### 2.1 License / 许可证

License 是 Cerberus 中最核心的授权对象。每个 License 都有唯一 ID，客户端需要使用该 ID 完成激活和验证。

常见字段：

| 字段                   | 说明                             |
|----------------------|--------------------------------|
| `id`                 | 许可证唯一 ID，也就是客户端使用的 License Key |
| `name`               | 许可证名称，便于管理员识别                  |
| `product`            | 产品标识，用于区分不同软件或版本               |
| `issuer`             | 签发方                            |
| `max_machines`       | 最大绑定设备数                        |
| `valid_from`         | 生效时间                           |
| `valid_until`        | 到期时间                           |
| `status`             | 当前状态：活跃、已过期、已撤销、已暂停等           |
| `max_unbind_count`   | 最大自助解绑次数                       |
| `unbind_count`       | 已使用解绑次数                        |
| `ip_binding_enabled` | 是否启用 IP 绑定校验                   |

### 2.2 Machine / 设备

设备是客户端激活 License 后绑定到服务端的一条机器记录。

常见字段：

| 字段            | 说明           |
|---------------|--------------|
| `id`          | 设备记录 ID      |
| `license_id`  | 绑定的许可证 ID    |
| `fingerprint` | 机器指纹         |
| `hostname`    | 主机名          |
| `os`          | 操作系统         |
| `arch`        | CPU 架构       |
| `ip`          | 激活或心跳时上报的 IP |
| `last_seen`   | 最后活跃时间       |
| `status`      | 设备状态         |

### 2.3 机器指纹

机器指纹用于识别客户端设备。CLI 和 SDK 支持自动采集，也可以由业务系统自行生成后传入。

建议：

- 默认使用 CLI/SDK 自动采集。
- 如果软件有自己的设备识别体系，可以通过 `--fingerprint` 或 `sdk.WithFingerprint()` 传入自定义指纹。
- 指纹应保持稳定，避免因重启、升级、小硬件变化导致频繁变更。

### 2.4 状态说明

| 状态          | 含义   | 常见处理               |
|-------------|------|--------------------|
| `active`    | 正常可用 | 客户端可激活、验证、心跳       |
| `expired`   | 已过期  | 管理员续期后恢复使用         |
| `revoked`   | 已撤销  | 管理员可重新启用，或保持禁用     |
| `suspended` | 已暂停  | 通常由换绑次数超限触发，管理员可解封 |

---

## 3. 快速启动

### 3.1 环境要求

| 项目   | 要求                             |
|------|--------------------------------|
| 操作系统 | Windows / Linux / macOS        |
| Go   | 1.21 或更高版本                     |
| 数据库  | SQLite，服务端自动创建数据库文件            |
| 浏览器  | Chrome / Edge / Firefox 等现代浏览器 |

### 3.2 启动服务端

进入服务端目录：

```bash
cd server
```

直接运行：

```bash
go run main.go
```

或者构建二进制：

```bash
go build -o cerberus-server .
./cerberus-server
```

当前仓库默认配置的端口是 `8010`，启动后访问：

```text
http://localhost:8010
```

健康检查：

```bash
curl http://localhost:8010/health
```

预期响应：

```json
{
  "status": "ok"
}
```

### 3.3 构建 CLI 客户端

```bash
cd cli
go build -o cerberus-client .
```

Windows 可以构建为：

```bash
go build -o cerberus-client.exe .
```

---

## 4. 配置说明

服务端配置文件位于：

```text
server/config.yaml
```

当前配置示例：

```yaml
server:
  port: 8010
  mode: release
  read_timeout: 10s
  write_timeout: 10s

database:
  path: ./data/cerberus.db

license:
  private_key_path: ./keys/private.pem
  public_key_path: ./keys/public.pem
  max_machines: 1
  heartbeat_ttl: 5m

auth:
  admin_token: xiay
  jwt_secret: cerberus-jwt-secret-xiay
  jwt_ttl: 24h

rate_limit:
  enabled: true
  rps: 20
  burst: 40

geoip:
  enabled: true
  policy: allow
  allowed_distance: 50
  database_path: ./data/GeoLite2-City.mmdb
```

### 4.1 server

| 参数              | 类型       | 说明                              |
|-----------------|----------|---------------------------------|
| `port`          | int      | HTTP 服务监听端口                     |
| `mode`          | string   | Gin 运行模式，常用 `debug` 或 `release` |
| `read_timeout`  | duration | 请求读取超时时间                        |
| `write_timeout` | duration | 响应写入超时时间                        |

### 4.2 database

| 参数     | 类型     | 说明                       |
|--------|--------|--------------------------|
| `path` | string | SQLite 数据库文件路径，首次启动会自动创建 |

### 4.3 license

| 参数                 | 类型       | 说明                   |
|--------------------|----------|----------------------|
| `private_key_path` | string   | Ed25519 私钥路径，仅服务端保存  |
| `public_key_path`  | string   | Ed25519 公钥路径，可随客户端分发 |
| `max_machines`     | int      | 创建许可证时默认最大设备数        |
| `heartbeat_ttl`    | duration | 心跳超时时间配置，当前作为预留配置    |

### 4.4 auth

| 参数            | 类型       | 说明                                |
|---------------|----------|-----------------------------------|
| `admin_token` | string   | 管理员登录令牌，也可直接作为管理 API Bearer Token |
| `jwt_secret`  | string   | JWT 签名密钥                          |
| `jwt_ttl`     | duration | 登录后 JWT 有效期                       |

生产环境必须修改：

```yaml
auth:
  admin_token: "请改成强随机字符串"
  jwt_secret: "请改成另一个强随机字符串"
```

### 4.5 rate_limit

| 参数        | 类型   | 说明     |
|-----------|------|--------|
| `enabled` | bool | 是否启用限流 |
| `rps`     | int  | 每秒请求速率 |
| `burst`   | int  | 突发请求上限 |

### 4.6 geoip

| 参数                 | 类型     | 说明                           |
|--------------------|--------|------------------------------|
| `enabled`          | bool   | 是否启用地理位置检测                   |
| `policy`           | string | 策略：`allow` / `warn` / `deny` |
| `allowed_distance` | int    | 允许距离阈值，单位公里                  |
| `database_path`    | string | GeoLite2 City 数据库路径          |

策略说明：

| 策略      | 行为           |
|---------|--------------|
| `allow` | 允许异地登录，仅记录信息 |
| `warn`  | 允许操作，但返回告警信息 |
| `deny`  | 超过距离阈值时拒绝操作  |

---

## 5. Web 管理后台使用

启动服务端后，打开：

```text
http://localhost:8010
```

### 5.1 登录后台

登录页如下：

![登录页](images/01-login.png)

操作步骤：

1. 打开 Web 管理后台。
2. 输入 `server/config.yaml` 中的 `auth.admin_token`。
3. 点击“登录”。
4. 登录成功后系统会将 JWT 保存到浏览器本地存储中，用于后续管理 API 请求。

当前示例配置中的管理员令牌为：

```text
xiay
```

生产环境请不要使用弱口令。

### 5.2 仪表盘

登录后进入仪表盘：

![仪表盘](images/02-dashboard.png)

仪表盘包含：

- 许可证总数。
- 活跃许可证数量。
- 7 天内即将过期许可证数量。
- 已过期 / 已撤销许可证数量。
- 产品分布图。
- 状态分布图。
- 最近许可证列表。

常用操作：

| 操作            | 说明            |
|---------------|---------------|
| 点击统计卡片        | 快速跳转或按状态筛选许可证 |
| 点击“新建许可证”     | 打开创建许可证弹窗     |
| 点击最近许可证中的详情按钮 | 查看许可证详情       |
| 点击“查看全部”      | 跳转到许可证管理页面    |

### 5.3 新建许可证

点击右上角“新建许可证”，打开创建弹窗：

![新建许可证](images/03-create-license.png)

字段说明：

| 字段         | 是否必填 | 说明                                   |
|------------|------|--------------------------------------|
| 许可证名称      | 是    | 便于管理员识别，如“企业版年度授权”                   |
| 产品标识       | 是    | 建议使用稳定标识，如 `my-app-pro`              |
| 有效期        | 是    | 可选 1 天、3 天、7 天、30 天、90 天、180 天、365 天 |
| 最大设备数      | 是    | 此许可证最多可绑定几台机器                        |
| 最大解绑次数     | 否    | 用户可自助换绑次数                            |
| 启用 IP 绑定验证 | 否    | 开启后验证时会检查 IP 是否与绑定记录一致               |

创建完成后，系统会返回许可证 ID。这个 ID 就是客户端激活和验证时使用的 License Key。

建议流程：

1. 管理员创建许可证。
2. 复制许可证 ID。
3. 将许可证 ID 发给用户或写入交付系统。
4. 用户在客户端输入许可证 ID 完成激活。

### 5.4 许可证管理

许可证管理页用于查看、筛选、续期、撤销和重新启用许可证：

![许可证列表](images/04-license-list.png)

页面能力：

| 功能   | 说明                |
|------|-------------------|
| 搜索   | 按许可证名称或 ID 搜索     |
| 产品筛选 | 按产品标识筛选           |
| 状态筛选 | 按活跃、过期、撤销、暂停筛选    |
| 刷新   | 重新加载服务端数据         |
| 查看详情 | 查看许可证完整信息、设备、审计日志 |
| 续期   | 为许可证追加有效期         |
| 撤销   | 禁用许可证及其设备         |
| 重新启用 | 将已撤销许可证恢复为可用状态    |

列表字段：

| 列      | 说明               |
|--------|------------------|
| 许可证 ID | 简短展示，点击可复制完整 ID  |
| 名称     | 许可证名称            |
| 产品     | 产品标识             |
| 状态     | 当前许可证状态          |
| 设备     | 当前绑定设备数量 / 最大设备数 |
| 剩余时间   | 到期倒计时            |
| 操作     | 查看、续期、撤销、重新启用等   |

### 5.5 许可证详情

点击许可证列表中的查看按钮，打开详情弹窗：

![许可证详情](images/05-license-detail.png)

详情页包含：

- 基本信息：ID、名称、产品、状态、IP 绑定状态。
- 有效期：开始时间、结束时间、剩余时间、进度条。
- 配额信息：设备配额、解绑次数。
- 已绑定设备：设备 ID、机器指纹、主机名、系统、IP、最后活跃时间。
- 操作记录：创建、验证、激活、续期、撤销等审计日志。
- 操作区：续期、复制 ID、撤销、重新启用、解除暂停。

### 5.6 Key 验证工具

Key 验证页用于管理员快速检查某个许可证是否有效：

![Key 验证](images/06-key-verify.png)

操作步骤：

1. 进入“Key 验证”。
2. 输入许可证 ID。
3. 点击“验证”。
4. 查看许可证状态、剩余时间、设备配额和已绑定设备。

适用场景：

- 客服收到用户反馈“许可证不可用”时快速排查。
- 发货前检查许可证是否处于活跃状态。
- 续期或撤销后确认状态是否已更新。

### 5.7 设备总览

设备总览页展示所有已激活设备：

![设备总览](images/07-devices.png)

字段说明：

| 列     | 说明                     |
|-------|------------------------|
| 设备 ID | 设备记录唯一 ID              |
| 许可证   | 所属许可证名称或 ID            |
| 产品    | 所属产品                   |
| 主机名   | 客户端机器名                 |
| 系统    | 操作系统和架构                |
| IP 地址 | 激活或心跳上报 IP             |
| 位置    | GeoIP 解析位置，未配置数据库时可能为空 |
| 最后活跃  | 最近一次激活或心跳时间            |
| 状态    | 设备当前状态                 |

点击设备行可以查看设备详情。对于异常设备，可以在许可证详情中执行“解绑此设备”。

---

## 6. 许可证完整生命周期

### 6.1 创建

管理员在后台或 API 创建 License。

```text
创建许可证 → 获得 License ID → 交付给用户
```

### 6.2 激活

用户首次使用客户端时，客户端采集机器指纹并请求激活。

```text
客户端采集机器指纹
        │
        ▼
POST /api/v1/activate
        │
        ▼
服务端检查 License 状态和设备配额
        │
        ▼
写入 Machine 绑定记录
        │
        ▼
返回激活成功
```

### 6.3 验证

客户端启动时或关键功能使用前调用验证接口。

```text
客户端提供 License ID + 机器指纹
        │
        ▼
POST /api/v1/verify
        │
        ▼
服务端检查状态、有效期、设备绑定、IP 绑定
        │
        ├── 有效：允许使用软件
        └── 无效：提示过期、撤销、设备不匹配等原因
```

### 6.4 心跳

客户端运行期间定期上报心跳。

```text
客户端定时调用 POST /api/v1/heartbeat
        │
        ▼
服务端更新 last_seen
        │
        ▼
后台设备总览显示最后活跃时间
```

建议心跳频率：

| 软件类型 | 建议频率             |
|------|------------------|
| 桌面软件 | 5 到 30 分钟一次      |
| 长驻服务 | 1 到 10 分钟一次      |
| 短时工具 | 启动时验证即可，必要时退出前上报 |

### 6.5 续期

许可证到期前，管理员可以在后台续期。

续期逻辑：

- 如果许可证未过期，通常在原到期时间基础上追加时长。
- 如果许可证已过期，系统会按服务端逻辑更新新的有效期。

### 6.6 撤销

撤销用于永久或临时禁用某个许可证。

效果：

- License 状态变为 `revoked`。
- 客户端验证会失败。
- 管理员仍可在后台重新启用。

### 6.7 重新启用

已撤销许可证可以重新启用。

适用场景：

- 误操作撤销。
- 用户补缴费用后恢复服务。
- 内部测试许可证临时关闭后恢复。

### 6.8 设备解绑

管理员可以在许可证详情中解绑某台设备。

适用场景：

- 用户更换电脑。
- 旧设备损坏。
- 检测到异常设备。

客户端也可以调用自助解绑接口，但会受到最大解绑次数限制。

---

## 7. 客户端 CLI 使用

CLI 位于：

```text
cli/
```

构建：

```bash
cd cli
go build -o cerberus-client .
```

全局参数：

| 参数                  | 说明                      |
|---------------------|-------------------------|
| `--server, -s`      | Cerberus Server 地址      |
| `--fingerprint, -f` | 机器指纹，默认 `auto` 自动采集     |
| `--output, -o`      | 输出格式，支持 `json` / `text` |

也可以通过环境变量指定服务地址：

```bash
export CERBERUS_SERVER_URL=http://localhost:8010
```

Windows PowerShell：

```powershell
$env:CERBERUS_SERVER_URL="http://localhost:8010"
```

### 7.1 采集机器指纹

```bash
cerberus-client fingerprint
```

输出示例：

```json
{
  "fingerprint": "机器指纹字符串",
  "hostname": "DESKTOP-EXAMPLE",
  "os": "windows",
  "arch": "amd64"
}
```

### 7.2 激活许可证

```bash
cerberus-client activate \
  --license 35d0dc02-0951-4011-89a2-ae56b1b65333 \
  --server http://localhost:8010
```

指定自定义指纹：

```bash
cerberus-client activate \
  --license 35d0dc02-0951-4011-89a2-ae56b1b65333 \
  --fingerprint demo-machine-001 \
  --server http://localhost:8010
```

### 7.3 验证许可证

```bash
cerberus-client verify \
  --license 35d0dc02-0951-4011-89a2-ae56b1b65333 \
  --server http://localhost:8010
```

业务程序应根据 JSON 中的 `valid` 字段决定是否允许继续运行。

### 7.4 心跳上报

```bash
cerberus-client heartbeat \
  --license 35d0dc02-0951-4011-89a2-ae56b1b65333 \
  --server http://localhost:8010
```

建议客户端软件启动后定时调用。

### 7.5 自助解绑 / 换绑

```bash
cerberus-client unbind \
  --license 35d0dc02-0951-4011-89a2-ae56b1b65333 \
  --old-fingerprint old-machine-fingerprint \
  --server http://localhost:8010
```

解绑成功后，用户可以在新机器上重新激活。

---

## 8. HTTP API 使用

服务端统一返回 JSON：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

`code = 0` 表示成功，非 0 或 HTTP 错误码表示失败。

### 8.1 管理认证

管理接口支持两种认证方式：

1. 直接使用 Admin Token。
2. 先登录获取 JWT，再使用 JWT。

请求头格式：

```http
Authorization: Bearer <admin-token-or-jwt>
```

### 8.2 管理员登录

```bash
curl -X POST http://localhost:8010/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"token":"xiay"}'
```

响应示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "token": "jwt-token"
  }
}
```

### 8.3 创建许可证

```bash
curl -X POST http://localhost:8010/api/v1/licenses/create \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{
    "name": "Business Annual License",
    "product": "Cerberus Enterprise",
    "issuer": "Cerberus Team",
    "duration_sec": 31536000,
    "max_machines": 5,
    "max_unbind_count": 2,
    "ip_binding_enabled": false
  }'
```

### 8.4 查询许可证详情

```bash
curl -X POST http://localhost:8010/api/v1/licenses/get \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{"id":"35d0dc02-0951-4011-89a2-ae56b1b65333"}'
```

### 8.5 查询许可证列表

```bash
curl -X POST http://localhost:8010/api/v1/licenses/list \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{"page":1,"size":20,"status":"active"}'
```

可选筛选字段：

| 字段        | 说明   |
|-----------|------|
| `page`    | 页码   |
| `size`    | 每页数量 |
| `status`  | 状态筛选 |
| `product` | 产品筛选 |

### 8.6 续期许可证

```bash
curl -X POST http://localhost:8010/api/v1/licenses/renew \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{"id":"35d0dc02-0951-4011-89a2-ae56b1b65333","duration_sec":31536000}'
```

### 8.7 撤销许可证

```bash
curl -X POST http://localhost:8010/api/v1/licenses/revoke \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{"id":"35d0dc02-0951-4011-89a2-ae56b1b65333"}'
```

### 8.8 重新启用许可证

```bash
curl -X POST http://localhost:8010/api/v1/licenses/reactivate \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{"id":"35d0dc02-0951-4011-89a2-ae56b1b65333"}'
```

### 8.9 解除暂停

```bash
curl -X POST http://localhost:8010/api/v1/licenses/unsuspend \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{"id":"35d0dc02-0951-4011-89a2-ae56b1b65333","reset_unbind_count":true}'
```

### 8.10 查询审计日志

```bash
curl -X POST http://localhost:8010/api/v1/licenses/audit \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{"id":"35d0dc02-0951-4011-89a2-ae56b1b65333"}'
```

### 8.11 管理员解绑设备

```bash
curl -X POST http://localhost:8010/api/v1/licenses/machines/revoke \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{
    "license_id":"35d0dc02-0951-4011-89a2-ae56b1b65333",
    "machine_id":"856df093-2857-49a0-9d41-3493749f089e"
  }'
```

### 8.12 客户端激活

公开接口，不需要管理员认证。

```bash
curl -X POST http://localhost:8010/api/v1/activate \
  -H 'Content-Type: application/json' \
  -d '{
    "license_id":"35d0dc02-0951-4011-89a2-ae56b1b65333",
    "fingerprint":"demo-machine-001",
    "hostname":"demo-workstation",
    "os":"windows",
    "arch":"amd64",
    "ip":"127.0.0.1"
  }'
```

### 8.13 客户端验证

```bash
curl -X POST http://localhost:8010/api/v1/verify \
  -H 'Content-Type: application/json' \
  -d '{
    "license_id":"35d0dc02-0951-4011-89a2-ae56b1b65333",
    "fingerprint":"demo-machine-001"
  }'
```

响应中重点关注：

| 字段           | 说明      |
|--------------|---------|
| `valid`      | 是否有效    |
| `message`    | 失败或提示原因 |
| `license_id` | 许可证 ID  |
| `product`    | 产品标识    |
| `expires_in` | 剩余有效秒数  |

### 8.14 客户端心跳

```bash
curl -X POST http://localhost:8010/api/v1/heartbeat \
  -H 'Content-Type: application/json' \
  -d '{
    "license_id":"35d0dc02-0951-4011-89a2-ae56b1b65333",
    "fingerprint":"demo-machine-001"
  }'
```

### 8.15 客户端自助解绑

```bash
curl -X POST http://localhost:8010/api/v1/unbind \
  -H 'Content-Type: application/json' \
  -d '{
    "license_id":"35d0dc02-0951-4011-89a2-ae56b1b65333",
    "old_fingerprint":"demo-machine-001"
  }'
```

---

## 9. Go SDK 集成

SDK 位于：

```text
sdk/
```

### 9.1 创建客户端

```go
package main

import (
	"context"
	"fmt"

	"cerberus.dev/sdk"
)

func main() {
	ctx := context.Background()
	client := sdk.NewClient("http://localhost:8010")

	licenseID := "35d0dc02-0951-4011-89a2-ae56b1b65333"

	result, err := client.Verify(ctx, licenseID, sdk.WithFingerprintAuto())
	if err != nil {
		panic(err)
	}

	if !result.Valid {
		fmt.Println("license invalid:", result.Message)
		return
	}

	fmt.Println("license valid")
}
```

### 9.2 激活

```go
resp, err := client.Activate(ctx, licenseID, sdk.WithFingerprintAuto())
if err != nil {
return err
}
fmt.Println("activated machine:", resp.Machine.ID)
```

### 9.3 验证

```go
result, err := client.Verify(ctx, licenseID, sdk.WithFingerprintAuto())
if err != nil {
return err
}
if !result.Valid {
return fmt.Errorf("license invalid: %s", result.Message)
}
```

### 9.4 心跳

```go
err := client.Heartbeat(ctx, licenseID, sdk.WithFingerprintAuto())
if err != nil {
return err
}
```

### 9.5 定时心跳示例

```go
ticker := time.NewTicker(5 * time.Minute)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return ctx.Err()
case <-ticker.C:
if err := client.Heartbeat(ctx, licenseID, sdk.WithFingerprintAuto()); err != nil {
log.Println("heartbeat failed:", err)
}
}
}
```

### 9.6 自定义机器指纹

```go
result, err := client.Verify(
ctx,
licenseID,
sdk.WithFingerprint("custom-machine-id"),
)
```

### 9.7 指定 IP

```go
result, err := client.Verify(
ctx,
licenseID,
sdk.WithFingerprintAuto(),
sdk.WithIP("203.0.113.10"),
)
```

---

## 10. 典型接入流程

### 10.1 桌面软件接入

推荐流程：

```text
软件启动
  │
  ├─ 本地读取已保存的 License ID
  │
  ├─ 如果没有 License ID：弹出输入框
  │
  ├─ 调用 activate 或 verify
  │
  ├─ valid=true：进入主界面
  │
  └─ valid=false：提示用户续费、换绑或联系管理员
```

建议：

- 首次输入 License ID 后保存到本地配置。
- 每次启动调用 `verify`。
- 软件运行时每隔 5 到 30 分钟调用 `heartbeat`。
- 对关键功能可以再次调用 `verify`。

### 10.2 服务端软件接入

推荐流程：

```text
服务启动 → 读取 License ID → verify → 启动业务服务 → 定时 heartbeat
```

如果验证失败，应阻止服务启动或进入受限模式。

### 10.3 使用 CLI 作为跨语言集成桥

对于 Java、Python、Electron、C# 等应用，可以打包 `cerberus-client`，通过子进程调用：

```text
业务程序
  │
  ├─ 执行 cerberus-client verify --license xxx --server xxx
  │
  ├─ 读取 stdout JSON
  │
  ├─ 判断 valid 字段
  │
  └─ 决定是否允许继续使用
```

优点：

- 不需要每种语言都实现机器指纹采集。
- CLI 输出 JSON，解析简单。
- 便于在安装器、脚本、桌面应用中复用。

### 10.4 简单 Python 调用示例

```python
import json
import subprocess

result = subprocess.run(
    [
        "cerberus-client",
        "verify",
        "--license", "35d0dc02-0951-4011-89a2-ae56b1b65333",
        "--server", "http://localhost:8010",
    ],
    capture_output=True,
    text=True,
    check=False,
)

if result.returncode != 0:
    raise RuntimeError(result.stderr)

data = json.loads(result.stdout)
if not data.get("valid"):
    raise RuntimeError("license invalid")
```

---

## 11. 运维管理

### 11.1 数据目录

默认数据文件：

```text
server/data/cerberus.db
```

默认密钥文件：

```text
server/keys/private.pem
server/keys/public.pem
```

建议备份：

- SQLite 数据库文件。
- 私钥文件。
- 配置文件。

### 11.2 备份 SQLite

服务运行中建议使用 SQLite 在线备份方式；简单部署也可以在停服后复制数据库文件。

停服备份：

```bash
cp server/data/cerberus.db backup/cerberus-$(date +%F).db
```

### 11.3 日志查看

前台运行时日志直接输出到终端。

后台运行可重定向：

```bash
cd server
./cerberus-server > cerberus.log 2>&1
```

查看日志：

```bash
tail -f cerberus.log
```

Windows PowerShell：

```powershell
Get-Content .\cerberus.log -Wait
```

### 11.4 反向代理部署

生产环境建议使用 Nginx、Caddy、IIS 或其他网关代理 HTTPS。

示例结构：

```text
Internet
   │ HTTPS
   ▼
Reverse Proxy
   │ HTTP localhost:8010
   ▼
Cerberus Server
```

代理时请保留真实客户端 IP 头：

```text
X-Forwarded-For
X-Real-IP
```

服务端会按如下优先级获取 IP：

1. 请求体中的 `ip`。
2. `X-Forwarded-For`。
3. `X-Real-IP`。
4. 连接来源 IP。

### 11.5 升级建议

升级前：

1. 停止服务。
2. 备份 `server/data/cerberus.db`。
3. 备份 `server/config.yaml`。
4. 备份 `server/keys/`。
5. 替换二进制或拉取新代码构建。
6. 启动服务并检查 `/health`。
7. 登录后台确认许可证和设备数据正常。

---

## 12. 安全建议

### 12.1 修改默认密钥

生产环境必须修改：

```yaml
auth:
  admin_token: "强随机管理员令牌"
  jwt_secret: "强随机 JWT 密钥"
```

不要将生产环境配置提交到公开仓库。

### 12.2 使用 HTTPS

License ID、机器指纹和管理 Token 都应通过 HTTPS 传输。

生产环境不要让客户端直接访问明文 HTTP 地址。

### 12.3 限制后台访问

建议：

- 后台放在内网或 VPN 后。
- 使用反向代理增加 IP 白名单。
- 不要暴露弱口令 Admin Token。
- 定期轮换管理员令牌。

### 12.4 保护私钥

`private.pem` 只应存在于服务端，不应随客户端分发。

如果私钥泄露，应立即：

1. 停止服务。
2. 生成新的密钥对。
3. 评估是否需要重新签发许可证。
4. 检查历史访问日志。

### 12.5 合理设置限流

公网部署时建议开启限流：

```yaml
rate_limit:
  enabled: true
  rps: 20
  burst: 40
```

如果客户端规模较大，应根据并发量提高 `rps` 和 `burst`。

---

## 13. 故障排查

### 13.1 无法打开后台

检查服务是否启动：

```bash
curl http://localhost:8010/health
```

如果失败：

- 检查服务端进程是否运行。
- 检查端口是否与配置一致。
- 检查防火墙是否拦截。
- 查看启动日志。

### 13.2 登录失败

可能原因：

| 原因                   | 处理                                           |
|----------------------|----------------------------------------------|
| Admin Token 输入错误     | 检查 `server/config.yaml` 中 `auth.admin_token` |
| 配置文件不是当前运行目录下的文件     | 确认从 `server` 目录启动                            |
| JWT Secret 变更导致旧登录失效 | 重新登录                                         |

### 13.3 创建许可证失败

检查：

- 是否已登录。
- 请求头是否包含 `Authorization`。
- `duration_sec` 是否大于 0。
- `name` 和 `product` 是否为空。
- 服务端日志中是否有数据库错误。

### 13.4 客户端验证失败

常见原因：

| 提示                    | 可能原因          | 处理                |
|-----------------------|---------------|-------------------|
| license not found     | License ID 错误 | 复制后台完整 ID 再试      |
| license expired       | 许可证已过期        | 后台续期              |
| license revoked       | 许可证已撤销        | 后台重新启用或重新发证       |
| machine not activated | 当前机器未激活       | 先调用 activate      |
| machine mismatch      | 指纹不匹配         | 检查指纹是否变化，必要时解绑旧设备 |
| max machines reached  | 设备数达到上限       | 提高设备上限或解绑旧设备      |

### 13.5 设备未出现在后台

检查：

- 客户端是否调用了 `/api/v1/activate`。
- 激活请求是否成功返回。
- License ID 是否正确。
- 机器指纹是否为空。
- 后台是否点击刷新。

### 13.6 中文显示乱码

在 Windows Git Bash 或 CMD 中使用 curl 发送中文 JSON 时，终端编码可能导致中文乱码。

建议：

- Web 后台直接输入中文。
- PowerShell 使用 UTF-8。
- API 调试时先使用英文或从 UTF-8 文件读取 JSON。

### 13.7 端口被占用

修改 `server/config.yaml`：

```yaml
server:
  port: 8011
```

然后重启服务。

---

## 14. 附录：常用命令速查

### 服务端

```bash
cd server
go run main.go
```

```bash
curl http://localhost:8010/health
```

### CLI

```bash
cd cli
go build -o cerberus-client .
```

```bash
cerberus-client fingerprint
```

```bash
cerberus-client activate --license <license-id> --server http://localhost:8010
```

```bash
cerberus-client verify --license <license-id> --server http://localhost:8010
```

```bash
cerberus-client heartbeat --license <license-id> --server http://localhost:8010
```

```bash
cerberus-client unbind --license <license-id> --old-fingerprint <fingerprint> --server http://localhost:8010
```

### 管理 API

```bash
curl -X POST http://localhost:8010/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"token":"xiay"}'
```

```bash
curl -X POST http://localhost:8010/api/v1/licenses/list \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer xiay' \
  -d '{"page":1,"size":20}'
```

### 公开 API

```bash
curl -X POST http://localhost:8010/api/v1/activate \
  -H 'Content-Type: application/json' \
  -d '{"license_id":"<license-id>","fingerprint":"<fingerprint>"}'
```

```bash
curl -X POST http://localhost:8010/api/v1/verify \
  -H 'Content-Type: application/json' \
  -d '{"license_id":"<license-id>","fingerprint":"<fingerprint>"}'
```

---

## 15. 推荐日常操作流程

### 管理员发放许可证

```text
登录后台 → 新建许可证 → 复制许可证 ID → 发给用户 → 用户激活 → 后台确认设备绑定
```

### 用户更换设备

```text
用户申请换绑 → 管理员查看旧设备 → 后台解绑旧设备 → 用户新设备重新激活
```

### 用户续费

```text
搜索许可证 → 打开详情 → 点击续期 → 选择时长 → 保存 → Key 验证确认有效期
```

### 处理滥用或退款

```text
搜索许可证 → 查看设备和审计记录 → 撤销许可证 → 客户端下次验证失败
```

---

文档截图保存在：

```text
docs/images/
```

如果后台界面样式或功能变更，重新打开服务端并更新截图即可。
