# UnoAdmin

UnoAdmin 是面向多租户 SaaS 的管理框架：同一套代码同时服务**集团统一采购**与**单体公司采购**两类客户。前端基于 Vue 3 + UnoUI，后端基于 Go + Gin + MongoDB + Redis。

## 多租户模型

```text
Tenant 租户（采购 / 计费 / 隔离边界）
├── LegalEntity 经营主体（集团总部 / 经营公司）
│   └── OrgUnit 组织节点（部门 / 门店 / 班组）
├── Account 账户（登录身份 + 数据范围）
└── Role 角色（权限码 + 数据范围）
```

- **集团租户**（`mode = group`）：一个租户挂多个经营主体，集团总部账号可统一查看，各公司账号只看自己主体。
- **单体租户**（`mode = standalone`）：一个租户只有一个经营主体，新建租户时自动创建。
- **数据边界**：租户 → 经营主体 → 组织，依次收敛；边界由后端依据令牌强制，前端不参与推导。

详见 [docs/multi-tenant-model.md](docs/multi-tenant-model.md)。

## 快速开始

```sh
pnpm install

# 后端：写入种子数据后启动（默认使用本地 MongoDB / Redis）
cd packages/backend/admin-service
MONGO_DATABASE=unoadmin go run ./cmd/seed
HTTP_ADDR=:8282 REDIS_DB=10 go run ./cmd/server

# 前端
cd ../../..
pnpm dev
```

前端地址为 <http://localhost:5173/>，接口地址由 `admin-app/.env.development` 的 `VITE_API_BASE_URL` 指定（默认 `http://localhost:8282/api/v1/`）。

### 演示账号

| 账号 | 密码 | 形态 | 数据范围 |
| --- | --- | --- | --- |
| `lin.wei` | `admin123` | 平台运营 | 全部租户 |
| `chen.yu` | `admin123` | 汉唐控股集团（集团） | 租户全部主体 |
| `zhao.ming` | `admin123` | 汉唐控股集团（集团） | 仅华东公司 |
| `li.na` | `admin123` | 明远科技（单体） | 唯一主体 |

## Workspace

```text
packages/
├── backend/admin-service/     # REST + gRPC，多租户领域服务
└── frontend/
    ├── admin-app/             # 唯一 SPA 入口，组装 UnoUI、路由、会话与服务
    ├── admin-base/            # Layout、导航与基础管理页面
    ├── core/                  # TypeScript 合同、服务端口、注入令牌
    └── vite-plugins/routes-generator/
```

## 管理端页面

| 页面 | 路由 | 说明 |
| --- | --- | --- |
| 工作台 | `/dashboard` | 规模指标与当前上下文 |
| 租户 | `/system/tenants` | 租户形态、套餐、状态；单体租户自动建主体 |
| 经营主体 | `/system/entities` | 集团总部 + 经营公司层级 |
| 组织架构 | `/system/organizations` | 主体内组织树，账户授权粒度 |
| 账户 | `/system/accounts` | 数据范围、主体/组织授权、角色绑定 |
| 角色 | `/system/roles` | 权限码与数据范围，权限树授权 |
| 权限 | `/system/permissions` | 权限目录浏览、编辑与拖拽排序 |

UI 统一使用 [UnoUI](https://github.com/giacomohuang/UnoUI) 组件与 UnoCSS 语义色，布局、菜单与交互对齐 UnoMap 管理端。

## 检查

```sh
pnpm typecheck
pnpm test
pnpm build
pnpm lint

cd packages/backend/admin-service && go test ./...
```

架构说明见 [docs/frontend-architecture.md](docs/frontend-architecture.md)。
