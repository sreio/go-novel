<template>
  <div class="mt-4 p-3 border rounded" v-if="book" style="border:1px solid #eee;border-radius:.5rem">
    <div class="mb-2">已选：<strong>{{ book.Title }}</strong></div>
    <div class="flex gap-2 items-center">
      <select v-model="fmt" class="border rounded px-2 py-1">
        <option value="txt">TXT</option>
        <option value="epub">EPUB</option>
        <option value="pdf">PDF</option>
      </select>
      <button class="px-3 py-1" style="background:#111;color:#fff;border-radius:.375rem" :disabled="loading" @click="down">{{ loading ? '打包中…' : '下载' }}</button>
      <span v-if="err" style="color:#dc2626">{{ err }}</span>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { apiDownload } from '@/api/client'
import { saveBlob } from '@/utils/download'

const props = defineProps<{ book: { id: string; title?: string } | null }>()

const fmt = ref<'txt'|'epub'|'pdf'>('txt')
const loading = ref(false)
const err = ref('')

async function down(){
  if (!props.book) return
  err.value = ''
  loading.value = true
  try{
    const blob = await apiDownload(props.book.ID, fmt.value)
    const name = (props.book.Title || 'book') + '.' + fmt.value
    saveBlob(blob, name)
  }catch(e:any){
    err.value = e?.message || '下载失败'
  }finally{
    loading.value = false
  }
}
</script>
