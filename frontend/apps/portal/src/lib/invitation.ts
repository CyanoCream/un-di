import type { Invitation } from '@undangan/shared'
import { inject, type InjectionKey, type Ref } from 'vue'

export interface InvitationCtx {
  invitation: Ref<Invitation | null>
  setInvitation: (inv: Invitation) => void
  /** Muat ulang dari server tanpa menampilkan loading penuh. */
  reload: () => Promise<void>
}

export const INVITATION_CTX: InjectionKey<InvitationCtx> = Symbol('invitation')

export function useInvitationCtx() {
  const ctx = inject(INVITATION_CTX)
  if (!ctx) throw new Error('InvitationCtx tidak tersedia')
  return ctx
}
