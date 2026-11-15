/**
 * 品牌类型（Branded Type）的「不可伪造」标记键。
 *
 * @remarks
 * - 声明为 `unique symbol`，保证该键只属于本模块，外部无法伪造同名字段；
 * - 仅存在于类型层面，运行时并不存在该属性，不会被序列化，也没有任何运行时开销；
 * - 建议不对外导出，避免使用者绕过校验直接构造 `Opaque`。
 */
declare const __opaqueMarker: unique symbol

/**
 * 为类型 `T` 打上名为 `N` 的「品牌」标签，得到语义上不可互换的派生类型。
 *
 * @remarks
 * 结构上它仍然是 `T`，可以照常读取原字段、参与运算；
 * 但因为多了一个 phantom（幻影）只读属性 `[__opaqueMarker]: N`，
 * TypeScript 会判定它既不等价于裸的 `T`，也不等价于其它品牌的 `N`，
 * 从而在编译期拦下「把 UserId 当成 OrderId 传」这类低级错误。
 *
 * @template T - 被打品牌的基础类型，如 `string`、`number` 或某个对象类型。
 * @template N - 品牌名（字符串字面量类型），用于区分同一基础类型下的不同业务含义。
 *
 * @example
 * type UserId  = Opaque<string, 'UserId'>
 * type OrderId = Opaque<string, 'OrderId'>
 *
 * const uid = 'u_1' as UserId
 * const oid: OrderId = uid          // ❌ 编译报错，语义不同不能直传
 * const oid2 = uid as unknown as OrderId  // ✅ 需显式「盖章」才能转换
 *
 * @example
 * // 推荐配套：只暴露构造器，把类型转换收敛到一处，顺便做格式校验
 * function asUserId(raw: string): UserId {
 *   if (!raw.startsWith('u_')) throw new Error('invalid user id')
 *   return raw as UserId
 * }
 *
 * @see {@link https://www.typescriptlang.org/docs/handbook/2/narrowing.html#type-predicates} 类型收窄与断言相关用法
 */
type Opaque<T, N extends string> = T & { readonly [__opaqueMarker]: N }

/** Stable identifier for users. */
export type UserId = Opaque<string, 'UserId'>

/** Stable identifier for roles. */
export type RoleId = Opaque<string, 'RoleId'>

/** Stable identifier for permissions. */
export type PermissionId = Opaque<string, 'PermissionId'>

/** Stable identifier for tenants. */
export type TenantId = Opaque<string, 'TenantId'>
