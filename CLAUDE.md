# CLAUDE.md — adortb-agency

## 项目概述

代理商自助平台。语言 Go 1.25.3，监听端口 `:8109`。

## 目录结构

```
cmd/agency/main.go             # 入口：DB 连接池（max 50）+ 组件初始化 + HTTP
internal/
  agency/
    repo.go                    # 代理商 CRUD
    hierarchy.go               # 代理商-广告主关系（HasAccess 检查）
  user/
    rbac.go                    # 角色 + 权限定义 + 检查逻辑
    subaccount.go              # 子账户管理
  batch/
    ops.go                     # 批量操作（并发信号量 + WaitGroup）
  commission/
    calculator.go              # 返佣计算（TotalSpend × CommissionRate）
    settlement.go              # 月度结算（UPSERT 幂等）
  reporting/
    aggregator.go              # 跨广告主聚合报表
    white_label.go             # 白标配置 CRUD
  api/
    handler.go                 # 路由处理器
    auth.go                    # JWT 生成/验证（HMAC-SHA256）
    response.go                # 统一响应格式
  metrics/metrics.go           # Prometheus 指标
migrations/001_agency.up.sql
```

## RBAC 权限矩阵

| 角色 | view_campaigns | edit_campaigns | view_reports | manage_users | manage_billing |
|------|:-:|:-:|:-:|:-:|:-:|
| `agency_admin` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `media_buyer` | ✓ | ✓ | ✓ | ✗ | ✗ |
| `analyst` | ✗ | ✗ | ✓ | ✗ | ✗ |

- `user_permissions` 表中 `advertiser_id IS NULL` 表示全局权限
- `advertiser_id IS NOT NULL` 表示对该广告主的细粒度授权
- `CanAccessAdvertiser(userID, advertiserID)` 检查代理商是否可访问该广告主

## 批量操作并发模型

```go
// ops.go 核心逻辑
sem := make(chan struct{}, o.maxWorkers)  // 默认 maxWorkers = 10
var wg sync.WaitGroup

for _, id := range req.CampaignIDs {
    wg.Add(1)
    go func(campaignID int64) {
        defer wg.Done()
        sem <- struct{}{}    // 占用许可
        defer func() { <-sem }()
        // 执行操作
    }(id)
}
wg.Wait()
```

- 支持操作：`pause` / `resume` / `budget_update`
- 结果按 Campaign ID 一一对应，部分失败不影响其他

## 返佣计算规则

```
CommissionEarned = TotalAdvertiserSpend × CommissionRate
```

- `CommissionRate` 存储在 `agencies.commission_rate`（DECIMAL 5,4，默认 0.10）
- 结算周期：自然月，`period_month` 取月初日期
- 结算幂等：`INSERT ON CONFLICT (agency_id, period_month) DO UPDATE`
- 状态：`pending` → `invoiced`

## JWT 认证

```
Token 格式：base64(payload).hmac_sha256_signature
Payload：AgencyUserID / AgencyID / Role / ExpiresAt（24h）
```

**生产环境必须修改** JWT 密钥（当前硬编码为 `agency-jwt-secret-change-in-prod`）。

## 关键约定

- DB 连接池：`MaxOpenConns=50, MaxIdleConns=10`
- 批量操作的 DSP 客户端当前为 `NoopCampaignClient`（Noop），生产需对接 adortb-dsp
- 聚合报表框架已就绪，实际数据需调用 adortb-reporting 服务
- 新增角色时同步更新：`rbac.go` rolePermissions 映射表

## 数据库

五张核心表：`agencies` / `agency_users` / `agency_advertisers` / `agency_commissions` / `user_permissions`

```bash
psql $DATABASE_URL < migrations/001_agency.up.sql
```
