import type { Account, DataScope, LegalEntity, Tenant } from './types/entities'
import type { LegalEntityId, TenantId } from './types/identifiers'

/**
 * 认证适配器恢复的会话状态。
 *
 * 同时携带租户与经营主体两个上下文：租户决定采购边界与数据隔离范围，
 * 经营主体决定当前操作的具体公司。
 */
export interface AuthSession {
  account: Account
  permissions: string[]
  tenants: Tenant[]
  currentTenantId: TenantId | null
  legalEntities: LegalEntity[]
  currentLegalEntityId: LegalEntityId | null
  dataScope: DataScope
}

/** 真实认证后端接受的凭据。 */
export interface SignInCredentials {
  username: string
  password: string
}

/**
 * 认证适配器：进入、恢复与清理会话，并在租户/主体之间切换上下文。
 *
 * 凭据可选，便于本地适配器无需输入即可登录；HTTP 适配器缺少凭据时会抛出错误。
 */
export interface AuthPort {
  signIn(credentials?: SignInCredentials): Promise<AuthSession>
  restore(): Promise<AuthSession | null>
  signOut(): Promise<void>
  /** 切换租户：集团统一采购的客户跨租户属于平台侧操作。 */
  switchTenant?(id: TenantId): Promise<AuthSession | null>
  /** 在租户内切换经营主体，返回新的上下文。 */
  switchLegalEntity?(id: LegalEntityId): Promise<AuthSession | null>
}
