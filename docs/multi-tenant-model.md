# UnoAdmin 多租户模型

> 适用范围：`packages/frontend/*` 与 `packages/backend/admin-service`
>
> 目标：同一套代码同时服务「集团统一采购」和「单体公司采购」两类客户。

## 1. 三层结构

```text
Platform 平台
└── Tenant 租户（采购 / 计费 / 数据隔离边界）
    ├── LegalEntity 经营主体（法人，租户内隔离单元）
    │   └── OrgUnit 组织节点（部门 / 门店 / 班组）
    ├── Account 账户（管理端登录身份）
    ├── Role 角色（权限码 + 数据范围）
    └── Permission 权限（代码声明，按角色授予）
```

| 层级 | 职责 | 关键字段 |
| --- | --- | --- |
| Tenant | 采购与隔离的最外层；一个客户一个租户 | `mode = group \| standalone`、`plan`、`status` |
| LegalEntity | 租户内相互隔离的经营实体 | `tenantId`、`parentId`、`type = GROUP \| COMPANY` |
| OrgUnit | 主体内部组织，账户授权的粒度 | `legalEntityId`、`parentId`、`path` |
| Account | 登录身份与授权范围 | `tenantId`、`dataScope`、`legalEntityIds`、`orgIds`、`roleIds` |
| Role | 权限码与数据范围的组合 | `tenantId`、`legalEntityId`、`dataScope`、`permissionCodes` |

## 2. 两类客户形态

### 2.1 集团统一采购（`mode = group`）

- 一个租户下挂 1 个集团总部主体（`type = GROUP`）和 n 个经营公司（`type = COMPANY`）。
- 集团层面通过 `dataScope = tenant` 的账户统一查看全部主体数据。
- 各经营公司用 `dataScope = entity` 的账户独立管理，账户只被授权到本公司主体。

### 2.2 单体公司采购（`mode = standalone`）

- 一个租户下**有且只有一个**经营主体（`type = COMPANY`）。
- 新建租户时后端自动创建该主体，后续不允许再新增；前端「经营主体」页只保留编辑入口。
- 若客户后续需要多主体经营，把租户形态改为 `group` 后再添加主体。

### 2.3 各主体单独采购

每个经营主体各自开通一个 `standalone` 租户，平台运营管理员在租户之间切换；租户之间默认不可见，也不共享账户与角色。

## 3. 数据边界

三层边界依次收敛，越靠内优先级越高：

1. **租户边界**：`tenantId` 只能来自令牌，请求参数一律被服务层覆盖。
2. **主体边界**：`dataScope = entity / self` 的账户只能访问 `legalEntityIds` 内的主体。
3. **组织边界**：`dataScope = self` 的账户只能访问 `orgIds` 及其下级组织。

```go
// 主体是否可见：平台管理员与租户级范围放行，其余按授权列表判断
func CanAccessEntity(principal *Principal, entityID string) bool
```

角色与账户的数据范围取**更宽**的一方（`WidenScope`），避免角色配置把账户显式声明的主体范围收窄到不可用。

列表查询同样遵守边界：

- 主体级账户只能看到与自己授权主体有交集的账户（账户列表、工作台账户数一致）；
- 主体级操作者新建/编辑账户时，只能授权自己范围内的主体，且不能创建租户级账户；
- 非平台账号访问租户列表只会得到自己的租户。

| 场景 | 账户 dataScope | 角色 dataScope | 实际范围 |
| --- | --- | --- | --- |
| 集团总部管理员 | `tenant` | `tenant` | 租户全部主体 |
| 经营公司管理员 | `entity` | `entity` | 授权主体 |
| 部门负责人 | `self` | `self` | 本人组织及下级 |
| 权限收紧 | `entity` | `tenant` | 租户全部主体（角色优先放宽） |

## 4. 上下文与令牌

会话同时携带两个上下文：

- `currentTenantId`：当前租户；
- `currentLegalEntityId`：当前经营主体。

登录、刷新、恢复都会重新计算可切换范围；切换上下文时后端重新签发访问令牌，前端刷新整页以避免残留上一个上下文的数据。

| 接口 | 作用 |
| --- | --- |
| `POST /auth/login` | 账号密码登录，返回令牌 + 会话（租户、主体、权限、数据范围） |
| `POST /auth/restore` | 依据令牌重建会话，令牌中的上下文优先 |
| `POST /auth/refresh` | 轮换刷新令牌，沿用原租户与主体 |
| `POST /auth/switch-tenant` | 平台侧跨租户切换；普通账户只能切换到自身租户 |
| `POST /auth/switch-legal-entity` | 租户内切换主体，校验主体是否在授权范围内 |

请求头约定：

- `Authorization: Bearer <token>`
- `X-Tenant-Id`：必须与令牌一致，否则返回 403；
- `X-Legal-Entity-Id`：必须在令牌授权范围内，否则返回 403。

## 5. 管理接口

| 资源 | 接口 | 权限码 |
| --- | --- | --- |
| 租户 | `GET/POST /tenants`、`PUT /tenants/:id` | `tenant:view/create/update` |
| 经营主体 | `GET/POST /legal-entities`、`PUT /legal-entities/:id` | `entity:view/create/update` |
| 组织 | `GET /orgs?legalEntityId=`、`POST /orgs`、`PUT /orgs/:id` | `org:view/create/update` |
| 账户 | `GET/POST /accounts`、`PUT /accounts/:id`、`PATCH /accounts/:id/status` | `account:view/create/update` |
| 角色 | `GET/POST /roles`、`PUT /roles/:id` | `role:view/create/update` |
| 权限 | `GET /permissions` | `permission:view` |
| 指标 | `GET /stats` | 登录即可 |

非平台账号调用 `GET /tenants` 只会得到自己的租户；`POST/PUT /tenants` 仅平台运营管理员可用。

## 6. 校验规则（服务层）

| 规则 | 说明 |
| --- | --- |
| 租户形态 | `standalone` 最多一个主体；`group` 无上限 |
| 集团总部唯一 | 同一租户只允许一个 `type = GROUP` 主体 |
| 公司挂靠 | 经营公司必须挂在集团总部主体下，不能挂在另一个公司主体下 |
| 组织路径 | `path` 保存从根到自身的 ID 链，禁止把节点移动到自己的后代 |
| 账户主体 | `legalEntityIds` 必须属于账户所在租户；`entity/self` 范围至少选一个主体 |
| 角色归属 | 账户只能分配到同租户的角色 |
| 权限码 | 角色授权只能引用权限目录中已声明的权限码 |

## 7. 种子数据

`cmd/seed` 覆盖两类客户与三种数据范围：

| 账号 | 密码 | 租户 | 范围 |
| --- | --- | --- | --- |
| `lin.wei` | `admin123` | 平台 | 全部租户 |
| `chen.yu` | `admin123` | 汉唐控股集团（集团） | 租户全部主体 |
| `zhao.ming` | `admin123` | 汉唐控股集团（集团） | 仅华东公司 |
| `li.na` | `admin123` | 明远科技（单体） | 唯一主体 |

种子对「权限目录 + 内置角色」执行覆盖写入，保证升级后权限码同步；账户、租户与主体只做幂等插入，不覆盖业务数据。

## 8. 存储约定

- MongoDB 所有集合的 `_id` 均为原生 **ObjectId**；引用字段（`tenantId`、`legalEntityId`、`parentId`、`roleIds`、`legalEntityIds`、`orgIds`、`path`、`permissions.parentId`）同样以 ObjectId 存储，保证 `$lookup` 与聚合可直接关联。
- 外部接口（REST / gRPC）统一使用 ObjectId 的 24 位十六进制字符串，前端无需感知 BSON 类型；空引用返回空字符串，落库为 `NilObjectID`。
- 种子数据按自然键（租户编码、主体编码、权限码、角色编码、用户名、组织名称）查重，缺失时用 `bson.NewObjectID()` 生成主键，重复执行不会产生重复记录。
- 历史字符串主键由 `EnsureIndexes` 自动迁移：为每条旧记录生成新 ObjectId，同步改写引用字段，并用 `_legacyId` 标记保证中断后可重跑。
