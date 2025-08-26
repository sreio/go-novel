<template>
  <section class="mt-4" v-if="book" style="border:1px solid #eee;border-radius:.5rem;padding:12px">
    <header class="flex items-center justify-between mb-2">
      <h2 style="font-weight:600">章节列表</h2>
      <div style="font-size:.875rem;color:#6b7280">共 {{ chapters.length }} 章</div>
    </header>

    <div v-if="loading" style="color:#6b7280">加载章节中…</div>
    <div v-else-if="err" style="color:#dc2626">{{ err }}</div>

    <ul v-else class="max-h-80 overflow-auto" style="max-height:20rem;overflow:auto;border-top:1px solid #f3f4f6">
      <li v-for="c in chapters" :key="c.Index" class="py-2 flex items-center justify-between" style="border-bottom:1px solid #f3f4f6">
        <div class="truncate">{{ c.Title }}</div>
        <div class="flex gap-2">
          <button class="px-2 py-1" style="border:1px solid #ddd;border-radius:.375rem" @click="$emit('pick-chapter', c)">选择</button>
          <button class="px-2 py-1" style="border:1px solid #ddd;border-radius:.375rem" :disabled="previewing === c.URL" @click="onPreview(c)">
            {{ previewing === c.URL ? '预览中…' : '预览' }}
          </button>
        </div>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { apiChapters, type ChapterRow, apiChapter } from '@/api/client'

const props = defineProps<{ book: { id: string; title?: string } | null }>()
const emit = defineEmits<{ (e: 'pick-chapter', c: ChapterRow): void; (e:'preview', p:{title:string;url:string;content:string}): void }>()

const chapters = ref<ChapterRow[]>([])
const loading = ref(false)
const err = ref('')
const previewing = ref('')

async function fetchChapters(){
  if (!props.book) return
  loading.value = true
  err.value = ''
  chapters.value = []
  try{
    chapters.value = await apiChapters(props.book.ID)
  }catch(e:any){
    err.value = e?.message || '加载失败'
  }finally{
    loading.value = false
  }
}

async function onPreview(c: ChapterRow){
  previewing.value = c.URL
  try{
    const data = await apiChapter(c.URL, { limit: 800 })
    emit('preview', { title: data.title || c.Title, url: c.URL, content: data.content })
  }catch(e:any){
    err.value = e?.message || '预览失败'
  }finally{
    previewing.value = ''
  }
}

watch(() => props.book?.ID, () => fetchChapters())
onMounted(() => fetchChapters())
</script>
