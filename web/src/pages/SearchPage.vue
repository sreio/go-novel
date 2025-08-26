<template>
  <div class="maxw mx">
    <el-card shadow="never">
      <template #header>
        <div class="row">
          <el-input v-model="q" placeholder="输入关键词" clearable @keyup.enter="doSearch" style="max-width:420px" />
          <el-button type="primary" :loading="loading" @click="doSearch">搜索</el-button>
        </div>
      </template>

      <el-skeleton v-if="loading && books.length===0" :rows="6" animated />
      <el-empty v-else-if="!loading && books.length===0" description="无结果" />

      <el-table v-else :data="books" size="small" border stripe>
        <el-table-column type="index" width="60" label="#" />
        <el-table-column prop="source" label="来源" width="160" />
        <el-table-column prop="title" label="书名" min-width="220" />
        <el-table-column prop="category" label="分类" width="160" />
        <el-table-column prop="author" label="作者" width="160" />
        <el-table-column prop="update" label="更新时间" width="160" />
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button size="small" @click="goDetail(row)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { apiSearch, type SearchItem } from '@/api/client'
import { ElMessage } from 'element-plus'

const q = ref('')
const loading = ref(false)
const books = ref<SearchItem[]>([])
const router = useRouter()

async function doSearch(){
  loading.value = true
  try{
    books.value = await apiSearch(q.value.trim())
  }catch(e:any){
    ElMessage.error(e?.message || '搜索失败')
  }finally{
    loading.value = false
  }
}

function goDetail(row: SearchItem){
  router.push({ name: 'book', query: { id: row.id, title: row.title, source: row.source } })
}
</script>

<style scoped>
.maxw{max-width:72rem}.mx{margin:0 auto}
.row{display:flex; gap:.75rem; align-items:center}
</style>
