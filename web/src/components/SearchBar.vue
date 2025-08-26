<template>
  <form @submit.prevent="go" class="flex gap-2">
    <input v-model="q" placeholder="关键词" class="border px-3 py-2 rounded w-64" />
    <button class="px-4 py-2 rounded" style="background:#111;color:#fff" :disabled="loading">{{ loading ? '搜索中…' : '搜索' }}</button>
    <span v-if="err" style="color:#dc2626">{{ err }}</span>
  </form>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { apiSearch } from '@/api/client'
const emit = defineEmits<{ (e: 'search', items: any[]): void }>()
const q = ref('')
const loading = ref(false)
const err = ref('')

async function go(){
  err.value = ''
  loading.value = true
  try{
    const items = await apiSearch(q.value.trim())
    emit('search', items)
  }catch(e:any){
    err.value = e?.message || '请求失败'
  }finally{
    loading.value = false
  }
}
</script>
