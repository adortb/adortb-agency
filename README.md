# adortb-agency

代理商自助平台（第十三期）。支持多客户管理、RBAC 权限控制、批量操作和返佣计算。

## 功能概览

- 代理商创建与管理（白标域名/Logo/主题色）
- 多广告主层级管理（一代理商可管理多个广告主）
- 3 级角色 × 5 种权限的 RBAC 矩阵，支持广告主级细粒度授权
- 批量暂停/恢复/预算更新（并发信号量控制）
- 月度返佣计算与结算
- JWT 认证（24h 有效期）
- Prometheus 监控指标

## 快速启动

```bash
export DATABASE_URL="postgres://user:pass@localhost/adortb_agency"
export PORT="8109"

go run ./cmd/agency
```

## API 端点

**代理商管理**
```
POST   /v1/agencies
GET    /v1/agencies/:id
PUT    /v1/agencies/:id/white-label-config
GET    /v1/agencies/:id/white-label-config
```

**用户与权限**
```
POST   /v1/agencies/:id/users
GET    /v1/agencies/:id/users
POST   /v1/agencies/:id/advertisers/:adv_id/permissions
```

**广告主管理**
```
POST   /v1/agencies/:id/advertisers
GET    /v1/agencies/:id/advertisers
GET    /v1/agencies/:id/campaigns
```

**批量操作**
```
POST   /v1/agencies/:id/batch/pause-campaigns
POST   /v1/agencies/:id/batch/budget-update
```

**报表与返佣**
```
GET    /v1/agencies/:id/reports/aggregated
GET    /v1/agencies/:id/commissions
POST   /v1/agencies/:id/commissions/settle
GET    /v1/agencies/:id/commissions/:period
```

**认证**
```
POST   /v1/auth/login
POST   /v1/auth/switch-advertiser
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DATABASE_URL` | `postgres://localhost/adortb_agency` | PostgreSQL 连接串 |
| `PORT` | `8109` | 监听端口 |

## 技术栈

- Go 1.25.3
- PostgreSQL
- Prometheus
- golang.org/x/crypto（bcrypt）
