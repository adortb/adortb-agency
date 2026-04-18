# Architecture — adortb-agency

## 系统定位

代理商自助平台，处于代理商（多人/多角色团队）与 adortb 核心业务系统（adortb-admin、adortb-dsp、adortb-reporting）之间的中间层。

## 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        adortb-agency (:8109)                     │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  HTTP API（corsMiddleware → loggingMiddleware → JWT）     │   │
│  └──────┬───────────────────────────────────────────────────┘   │
│         │                                                        │
│  ┌──────▼──────────────────────────────────────────────────┐   │
│  │  Handler（路由分发 + RBAC 检查）                          │   │
│  └──────┬────────────┬──────────────┬──────────────────────┘   │
│         │            │              │                            │
│  ┌──────▼───┐ ┌──────▼──────┐ ┌──────▼────────────────────┐   │
│  │  Agency  │ │   User /    │ │ Commission / Reporting /   │   │
│  │  Repo    │ │   RBAC      │ │ Batch / WhiteLabel         │   │
│  └──────────┘ └─────────────┘ └────────────────────────────┘   │
│                                                                  │
│  PostgreSQL（连接池 max=50）                                       │
└─────────────────────────────────────────────────────────────────┘
         │                     │                     │
   adortb-admin         adortb-dsp           adortb-reporting
  （暂无直接集成）     （BatchOp 调用目标）   （聚合报表数据源）
```

## 核心模块

### 多客户层级关系

```
agencies (代理商)
    │ 1:N
    ├── agency_users（子账户）
    │       └── user_permissions（细粒度权限，可选 advertiser_id）
    │ 1:N
    └── agency_advertisers（代理商管理的广告主）
```

`HierarchyRepo.HasAccess(agencyID, advertiserID)` 确保跨客户数据隔离。

### RBAC 权限模型

```
┌──────────────┬────────────────────────────────────────────────┐
│    角色        │                  权限集合                       │
├──────────────┼────────────────────────────────────────────────┤
│ agency_admin │ view_campaigns + edit_campaigns + view_reports  │
│              │ + manage_users + manage_billing                 │
├──────────────┼────────────────────────────────────────────────┤
│ media_buyer  │ view_campaigns + edit_campaigns + view_reports  │
├──────────────┼────────────────────────────────────────────────┤
│ analyst      │ view_reports                                    │
└──────────────┴────────────────────────────────────────────────┘

权限可降级到广告主级别（user_permissions.advertiser_id IS NOT NULL）
```

### 批量操作并发模型

```
HTTP Request → BatchOp.Execute(req)
                   │
                   ▼
          sem := make(chan struct{}, 10)  [信号量，最大 10 并发]
                   │
          ┌────────┼────────┐
          ▼        ▼        ▼   (goroutines)
        op(1)    op(2)    op(3) ...
          │
          ▼
     CampaignClient
     [生产环境对接 adortb-dsp]
          │
          ▼
     []BatchResult  [汇总所有结果，部分失败不中断]
```

### 返佣计算与结算

```
结算时序：
  1. GET agencies.commission_rate
  2. SELECT SUM(spend) FROM [adortb-reporting] WHERE period=month
  3. commission_earned = spend × rate
  4. UPSERT agency_commissions
     ON CONFLICT (agency_id, period_month) DO UPDATE

状态：pending → invoiced（人工操作）
```

## 数据库 Schema 概要

```
agencies              ─── 代理商主体（白标配置内联）
  │ 1:N
agency_users          ─── 子账户（角色 + bcrypt 密码）
  │ 1:N
user_permissions      ─── 细粒度权限（advertiser_id 可 NULL）
  
agencies (1:N) agency_advertisers ─── 代理商-广告主关系

agencies (1:N) agency_commissions ─── 月度返佣记录
```

## JWT 认证流程

```
POST /v1/auth/login (email + password)
  │
  ├── 验证 bcrypt(password, stored_hash)
  │
  └── 签发 Token
        Payload: { AgencyUserID, AgencyID, Role, ExpiresAt: now+24h }
        签名: HMAC-SHA256(base64(payload), JWT_SECRET)

每个受保护路由：
  requireAuth(handler) middleware
    │
    └── 解析 Token → 注入 Claims 到 context
```

## 关键设计决策

1. **细粒度权限两层**：全局权限（`advertiser_id IS NULL`）+ 广告主级权限，灵活满足不同代理商组织结构
2. **批量操作信号量**：限制最大 10 并发，防止 DSP API 被打爆，结果独立收集不互相影响
3. **返佣 UPSERT**：月度结算幂等，重复触发不重复计费
4. **白标内联**：白标域名/Logo/主色调直接存 `agencies` 表，避免关联查询
5. **聚合报表延迟**：`Aggregator` 框架已就绪，调用 adortb-reporting 作为数据源（当前返回空框架）

## 可观测性

| 指标 | 含义 |
|------|------|
| `agency_http_requests_total{method,path,status}` | 各接口调用量 |
| `agency_http_request_duration_seconds{method,path}` | 接口延迟分布 |
| `agency_batch_ops_total{action,result}` | 批量操作成功/失败计数 |

## 部署拓扑

```
┌─────────────────────────────────────┐
│  adortb-agency (:8109)              │
│  ├── PostgreSQL                     │
│  ├── adortb-dsp（批量操作目标，计划）  │
│  ├── adortb-reporting（报表数据，计划）│
│  └── Prometheus                     │
└─────────────────────────────────────┘
```
