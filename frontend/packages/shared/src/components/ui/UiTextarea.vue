<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'

const model = defineModel<string>({ default: '' })

const props = withDefaults(
  defineProps<{ rows?: number; placeholder?: string; disabled?: boolean; invalid?: boolean; autogrow?: boolean; maxlength?: number }>(),
  { rows: 3, placeholder: undefined, disabled: false, invalid: false, autogrow: true, maxlength: undefined },
)

const el = ref<HTMLTextAreaElement | null>(null)

function grow() {
  if (!props.autogrow || !el.value) return
  el.value.style.height = 'auto'
  el.value.style.height = `${Math.min(el.value.scrollHeight + 2, 480)}px`
}

watch(model, () => nextTick(grow))
onMounted(grow)

defineExpose({ el })
</script>

<template>
  <textarea
    ref="el"
    v-model="model"
    :rows="rows"
    :placeholder="placeholder"
    :disabled="disabled"
    :maxlength="maxlength"
    :aria-invalid="invalid || undefined"
    class="block w-full min-w-0 resize-y rounded-xl border bg-surface-2 px-3 py-2 text-sm leading-relaxed text-ink transition outline-none placeholder:text-muted/70 focus:border-accent focus:bg-surface focus:ring-3 focus:ring-accent/15 disabled:opacity-60"
    :class="invalid ? 'border-danger' : 'border-line'"
  />
</template>
