<script setup lang="ts">
import { api, confirm, copyText, defaultWhatsappTemplate, fillShareTemplate, toast, type Invitation } from '@undangan/shared'
import CustomDomainPanel from '@undangan/shared/components/CustomDomainPanel.vue'
import InvitationSettingsPanel from '@undangan/shared/components/InvitationSettingsPanel.vue'
import SubdomainField from '@undangan/shared/components/SubdomainField.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiInput from '@undangan/shared/components/ui/UiInput.vue'
import { computed, ref } from 'vue'
import { useInvitationCtx } from '../../lib/invitation'
import { BASE_DOMAIN } from '../../lib/labels'

const ctx = useInvitationCtx()
const inv = computed(() => ctx.invitation.value as Invitation)

// ---------- Checklist publikasi ----------
const checklist = computed(() => {
  const c = inv.value.content
  return [
    { ok: !!inv.value.theme, label: 'Tema sudah dipilih', to: 'tema' },
    { ok: !!inv.value.subdomain, label: 'Subdomain sudah diatur', to: null },
    { ok: c.events.some((e) => !!e.date), label: 'Minimal 1 acara dengan tanggal', to: 'data' },
    { ok: !!c.groom.full_name.trim() && !!c.bride.full_name.trim(), label: 'Nama kedua mempelai terisi', to: 'data' },
  ]
})
const ready = computed(() => checklist.value.every((i) => i.ok))
const published = computed(() => inv.value.status === 'published')
const suspended = computed(() => inv.value.status === 'suspended')
const subInactive = computed(() => inv.value.subscription_status !== null && inv.value.subscription_status !== 'active')
const busy = ref(false)

async function publish() {
  busy.value = true
  try {
    const updated = await api.post<Invitation>(`/invitations/${inv.value.id}/publish`)
    ctx.setInvitation(updated)
    toast.success('Undangan berhasil dipublikasikan!')
  } catch (e) {
    toast.error(e)
  } finally {
    busy.value = false
  }
}

async function unpublish() {
  const ok = await confirm({
    title: 'Batalkan publikasi?',
    message: 'Undangan tidak bisa diakses tamu sampai dipublikasikan lagi. Link yang sudah dibagikan tidak akan berfungsi sementara.',
    confirmText: 'Ya, batalkan publikasi',
    danger: true,
  })
  if (!ok) return
  busy.value = true
  try {
    const updated = await api.post<Invitation>(`/invitations/${inv.value.id}/unpublish`)
    ctx.setInvitation(updated)
    toast.success('Publikasi dibatalkan. Undangan kembali menjadi draf.')
  } catch (e) {
    toast.error(e)
  } finally {
    busy.value = false
  }
}

// ---------- Link ----------
async function copy(text: string, msg = 'Link disalin') {
  if (await copyText(text)) toast.success(msg)
  else toast.error('Gagal menyalin')
}

const genericName = ref('')
const genericLink = computed(() => {
  if (!inv.value.url) return ''
  const name = genericName.value.trim()
  return name ? `${inv.value.url.replace(/\/$/, '')}/?to=${encodeURIComponent(name).replace(/%20/g, '+')}` : inv.value.url
})
const genericMessage = computed(() => {
  const tpl = inv.value.content.share.whatsapp_template || defaultWhatsappTemplate(inv.value.content)
  return fillShareTemplate(tpl, genericName.value.trim() || 'Bapak/Ibu/Saudara/i', genericLink.value)
})
const shareWaHref = computed(() => `https://wa.me/?text=${encodeURIComponent(genericMessage.value)}`)
</script>

<template>
  <div class="grid gap-5 pb-6 lg:grid-cols-2">
    <!-- Status publikasi -->
    <section class="card p-5 sm:p-6 lg:col-span-2">
      <div class="flex flex-col gap-5 md:flex-row md:items-start">
        <div class="flex-1">
          <div class="flex items-center gap-3">
            <span
              class="flex size-11 items-center justify-center rounded-full"
              :class="published ? 'bg-success/12 text-success' : suspended ? 'bg-danger/10 text-danger' : 'bg-accent/10 text-accent'"
            >
              <UiIcon :name="published ? 'globe' : suspended ? 'lock' : 'send'" :size="20" />
            </span>
            <div>
              <h2 class="heading text-2xl sm:text-3xl">
                {{ published ? 'Undangan sudah terbit' : suspended ? 'Undangan dinonaktifkan' : 'Siap dipublikasikan?' }}
              </h2>
              <p class="text-sm text-muted">
                {{
                  published
                    ? 'Tamu sudah bisa membuka undangan melalui link di bawah.'
                    : suspended
                      ? 'Masa aktif paket habis. Perpanjang paket untuk mengaktifkan kembali.'
                      : 'Lengkapi daftar berikut, lalu publikasikan undangan.'
                }}
              </p>
            </div>
          </div>

          <ul v-if="!published" class="mt-5 space-y-2">
            <li v-for="item in checklist" :key="item.label" class="flex items-center gap-2.5 text-sm">
              <span
                class="flex size-5 shrink-0 items-center justify-center rounded-full"
                :class="item.ok ? 'bg-success text-accent-ink' : 'border-2 border-line'"
              >
                <UiIcon v-if="item.ok" name="check" :size="12" :stroke-width="3" />
              </span>
              <span :class="item.ok ? 'text-ink' : 'text-muted'">{{ item.label }}</span>
              <RouterLink v-if="!item.ok && item.to" :to="`/undangan/${inv.id}/${item.to}`" class="text-xs font-medium text-accent hover:underline">
                Lengkapi
              </RouterLink>
            </li>
          </ul>
        </div>

        <div class="flex flex-col gap-2 md:w-64">
          <template v-if="suspended || subInactive">
            <UiButton href="/paket" @click.prevent="$router.push('/paket')"><UiIcon name="gift" :size="16" /> Perpanjang paket</UiButton>
          </template>
          <template v-if="!suspended">
            <UiButton v-if="!published" :disabled="!ready || subInactive" :loading="busy" class="h-11!" @click="publish">
              <UiIcon name="send" :size="16" /> Publikasikan
            </UiButton>
            <template v-else>
              <UiButton v-if="inv.url" :href="inv.url" target="_blank" rel="noopener" class="h-11!">
                <UiIcon name="external" :size="16" /> Buka undangan
              </UiButton>
              <UiButton variant="ghost" :loading="busy" @click="unpublish">Batalkan publikasi</UiButton>
            </template>
          </template>
          <p v-if="!published && !ready" class="text-center text-xs text-muted">Lengkapi semua poin untuk publikasi.</p>
        </div>
      </div>
    </section>

    <!-- Subdomain -->
    <section class="card p-5 sm:p-6">
      <h3 class="heading mb-1 text-2xl">Alamat undangan</h3>
      <p class="mb-4 text-sm text-muted">Pilih subdomain yang mudah diingat, mis. nama panggilan kedua mempelai.</p>
      <SubdomainField
        :invitation="inv"
        :editable="inv.status !== 'published'"
        :base-domain-hint="BASE_DOMAIN"
        locked-hint="Batalkan publikasi terlebih dahulu untuk mengganti subdomain."
        @saved="ctx.setInvitation"
      />

      <div v-if="inv.url" class="mt-5 border-t border-line pt-5">
        <p class="mb-2 text-xs tracking-wider text-muted uppercase">Link undangan</p>
        <div class="flex items-center gap-2 rounded-xl border border-line bg-surface-2 p-1.5 pl-3">
          <span class="min-w-0 flex-1 truncate text-sm font-medium text-ink">{{ inv.url }}</span>
          <UiButton variant="secondary" size="sm" @click="copy(inv.url)"><UiIcon name="copy" :size="14" /> Salin</UiButton>
          <UiButton variant="secondary" size="sm" :href="inv.url" target="_blank" rel="noopener" aria-label="Buka">
            <UiIcon name="external" :size="14" />
          </UiButton>
        </div>
        <p v-if="!published" class="mt-2 text-xs text-warning">Link belum bisa dibuka tamu sebelum undangan dipublikasikan.</p>
      </div>
    </section>

    <!-- Link umum -->
    <section class="card p-5 sm:p-6">
      <h3 class="heading mb-1 text-2xl">Bagikan ke grup</h3>
      <p class="mb-4 text-sm text-muted">
        Buat link dengan nama penerima umum, mis. untuk grup WhatsApp keluarga. Untuk tamu perorangan, gunakan link personal di tab
        <RouterLink :to="`/undangan/${inv.id}/tamu`" class="font-medium text-accent hover:underline">Tamu</RouterLink>.
      </p>
      <template v-if="inv.url">
        <label class="mb-1.5 block text-sm font-medium text-ink">Nama penerima</label>
        <UiInput v-model="genericName" placeholder="mis. Keluarga Besar Alumni SMA 1" />
        <div class="mt-3 flex items-center gap-2 rounded-xl border border-line bg-surface-2 p-1.5 pl-3">
          <span class="min-w-0 flex-1 truncate text-sm text-ink">{{ genericLink }}</span>
          <UiButton variant="secondary" size="sm" @click="copy(genericLink)"><UiIcon name="copy" :size="14" /> Salin</UiButton>
        </div>
        <div class="mt-3 flex flex-wrap gap-2">
          <UiButton variant="secondary" size="sm" :href="shareWaHref" target="_blank" rel="noopener">
            <UiIcon name="message" :size="14" class="text-success" /> Kirim via WhatsApp
          </UiButton>
          <UiButton variant="ghost" size="sm" @click="copy(genericMessage, 'Pesan undangan disalin')"><UiIcon name="copy" :size="14" /> Salin pesan</UiButton>
        </div>
      </template>
      <p v-else class="rounded-xl bg-surface-2 px-3 py-3 text-sm text-muted">Atur subdomain terlebih dahulu untuk membuat link.</p>
    </section>

    <!-- Domain sendiri -->
    <section class="card p-5 sm:p-6 lg:col-span-2">
      <h3 class="heading mb-1 text-2xl">Domain sendiri</h3>
      <p class="mb-4 text-sm text-muted">Hubungkan domain milik Anda agar undangan bisa dibuka lewat alamat pribadi, mis. www.rakadannadia.com.</p>
      <CustomDomainPanel :invitation="inv" :is-admin="false" upgrade-to="/paket" @saved="ctx.setInvitation" />
    </section>

    <!-- Akses & check-in -->
    <section class="lg:col-span-2">
      <h3 class="heading mb-1 text-2xl">Akses & check-in</h3>
      <p class="mb-4 text-sm text-muted">Batasi siapa yang bisa membuka undangan dan siapkan penerimaan tamu dengan QR di venue.</p>
      <InvitationSettingsPanel :invitation="inv" :is-admin="false" upgrade-to="/paket" @saved="ctx.setInvitation" />
    </section>
  </div>
</template>
