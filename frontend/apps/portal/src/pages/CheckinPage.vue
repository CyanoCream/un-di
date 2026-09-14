<script setup lang="ts">
import CheckinStation from '@undangan/shared/components/CheckinStation.vue'
import { computed } from 'vue'

const props = defineProps<{ id: string }>()
const loginUrl = computed(() => `/masuk?next=${encodeURIComponent(`/checkin/${props.id}`)}`)
</script>

<template>
  <CheckinStation :invitation-id="id" :login-url="loginUrl" />
</template>
