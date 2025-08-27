<template>
  <div class="maxw mx">
    <el-page-header @back="goBack" content="书籍详情" />

    <el-card class="mt" shadow="never">
      <template #header>
        <div class="hdr">
          <div class="ttl" :title="title">
            <el-tag>{{ source }}</el-tag>
            {{ title || '未命名' }}
          </div>
          <div class="row">
            <el-select v-model="fmt" placeholder="选择格式" style="width:120px">
              <el-option label="TXT" value="txt" />
              <el-option label="EPUB" value="epub" />
              <el-option label="PDF" value="pdf" />
            </el-select>
            <el-button type="primary" :loading="downloading" @click="download">下载</el-button>
          </div>
        </div>
        
        <!-- 下载进度显示 -->
        <div v-if="downloading" class="progress-container">
          <div class="progress-info">
            <span>下载进度: {{ downloadProgress }}%</span>
            <span>线程: {{ activeThreads }}/8</span>
            <span>章节: {{ completedNum }} / {{ totalChapters }} 章</span>
          </div>
          <el-progress 
            :percentage="downloadProgress" 
            :stroke-width="15" 
            :text-inside="true"
            status="success"
          />
        </div>
      </template>

      <el-table :data="pagedChapters" size="small" stripe border height="60vh" v-loading="loading">
        <el-table-column type="index" width="60" label="#" :index="indexMethod" />
        <el-table-column prop="title" label="章节" min-width="300" />
        <el-table-column width="120" label="预览">
          <template #default="{ row }">
            <el-button size="small" @click="openPreview(row)">预览</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pg-wrap">
        <el-pagination
          background
          layout="prev, pager, next, sizes, jumper, total"
          :total="chapters.length"
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[20, 50, 100, 200]"
          @current-change="(p:number)=>{ page=p }"
          @size-change="(s:number)=>{ pageSize=s; page=1 }"
        />
      </div>
    </el-card>

    <!-- Drawer Preview -->
    <el-drawer v-model="drawer" size="40%" :title="previewTitle || '章节预览'">
      <template #default>
        <el-skeleton v-if="previewLoading" :rows="12" animated />
        <pre v-else class="preview" v-html="previewText"></pre>
      </template>
      <template #footer>
        <div class="row" style="justify-content:space-between; width:100%">
          <el-switch v-model="full" inactive-text="前1000字" active-text="完整" />
          <el-button @click="drawer=false">关闭</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiChapters, apiChapter, apiDownload, type ChapterRow } from '@/api/client'
import { saveBlob } from '@/utils/download'
import { ElMessage } from 'element-plus'

const route = useRoute(); const router = useRouter()
const id = ref<string>(String(route.query.id || ''))
const title = ref<string>(String(route.query.title || ''))
const chapters = ref<ChapterRow[]>([])
const loading = ref(false)
const source = ref(String(route.query.source || ''))

// pagination
const page = ref(1)
const pageSize = ref(50)
const pagedChapters = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return chapters.value.slice(start, start + pageSize.value)
})
function indexMethod(index:number){ return (page.value - 1) * pageSize.value + index + 1 }

// drawer preview
const drawer = ref(false)
const previewText = ref('')
const previewTitle = ref('')
const previewLoading = ref(false)
const full = ref(false)

// download
const fmt = ref<'txt'|'epub'|'pdf'>('txt')
const downloading = ref(false)
const downloadProgress = ref(0)
const activeThreads = ref(0)
const totalChapters = ref(0)
const completedNum = ref(0)
let eventSource: EventSource | null = null

function goBack(){ router.push({ name: 'home' }) }

async function loadChapters(){
  if (!id.value) return
  loading.value = true
  try {
    const data = await apiChapters(id.value);
    chapters.value = data.chapters
    source.value = data.source
    page.value = 1
  } catch (e: any) {
    ElMessage.error(e?.message || '加载章节失败')
  } finally {
    loading.value = false
  }
}

async function openPreview(c: ChapterRow){
  drawer.value = true
  previewLoading.value = true
  previewTitle.value = c.title
  previewText.value = ''
  try{
    const data = await apiChapter(c.url, { limit: full.value ? undefined : 1000, full: full.value })
    previewText.value = data.content
  }catch(e:any){
    ElMessage.error(e?.message || '预览失败')
  }finally{ previewLoading.value = false }
}

function setupProgressListener() {
  if (eventSource) {
    eventSource.close()
  }
  
  eventSource = new EventSource('/api/progress')
  
  eventSource.addEventListener('connected', (event) => {
    console.log('SSE connected:', event)
  })
  
  eventSource.addEventListener('progress', (event) => {
    try {
      const data = JSON.parse(event.data)
      downloadProgress.value = Math.round(data.percentage)
      activeThreads.value = data.activeThreads
      totalChapters.value = data.totalChapters
      completedNum.value = data.completed
    } catch (e) {
      console.error('Failed to parse progress event:', e)
    }
  })
  
  eventSource.onerror = (error) => {
    console.error('SSE error:', error)
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
  }
}

function cleanupProgressListener() {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  downloadProgress.value = 0
  activeThreads.value = 0
  totalChapters.value = 0
  completedNum.value = 0
}

async function download(){
  if (!id.value) return
  downloading.value = true
  setupProgressListener()
  
  try{
    const blob = await apiDownload(id.value, fmt.value)
    saveBlob(blob, `${title.value || 'book'}.${fmt.value}`)
  }catch(e:any){ 
    ElMessage.error(e?.message || '下载失败') 
  }
  finally{ 
    downloading.value = false
    cleanupProgressListener()
  }
}

onMounted(loadChapters)
watch(() => route.query, () => { id.value = String(route.query.id||''); title.value = String(route.query.title||''); loadChapters() })
watch(full, () => {
  // 切换完整/截断时，如果抽屉打开且有标题，则重新拉取
  if (drawer.value && previewTitle.value) {
    const row = chapters.value.find(c => c.title === previewTitle.value)
    if (row) openPreview(row)
  }
})
</script>

<style scoped>
.maxw{max-width:80rem}.mx{margin:0 auto}.mt{margin-top:.75rem}
.hdr{display:flex; align-items:center; justify-content:space-between}
.row{display:flex; gap:.5rem; align-items:center}
.ttl{font-weight:700; font-size:18px; min-width:0; max-width:40vw; overflow:hidden; white-space:nowrap; text-overflow:ellipsis;}
.preview{white-space:pre-wrap; font-size:13px; line-height:1.6;}
.pg-wrap{display:flex; justify-content:flex-end; padding:.5rem 0;}

.progress-container {
  margin: 1rem 0;
  padding: 1rem;
  background: #f5f7fa;
  border-radius: 4px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.5rem;
  font-size: 14px;
  color: #606266;
}

.progress-info span {
  margin-right: 1rem;
}
</style>
