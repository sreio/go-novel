<template>
  <main class="p-4 max-w-3xl mx-auto">
    <h1 class="text-2xl font-bold mb-3">go-novel</h1>

    <SearchBar @search="onSearch" />
    <BookList :items="books" @pick="onPick" />

    <section v-if="picked" class="mt-4 grid grid-cols-1 gap-4">
      <ChaptersList :book="picked" @pick-chapter="onPickChapter" @preview="onPreview" />
      <DownloadPanel :book="picked" />
    </section>

    <section v-if="preview" class="mt-4 p-3 border rounded" style="background:#f9fafb">
      <header class="font-semibold mb-2">预览：{{ preview.title }}</header>
      <pre class="whitespace-pre-wrap text-sm">{{ previewText }}</pre>
    </section>
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SearchBar from './components/SearchBar.vue'
import BookList from './components/BookList.vue'
import DownloadPanel from './components/DownloadPanel.vue'
import ChaptersList from './components/ChaptersList.vue'

const books = ref<any[]>([])
const picked = ref<any | null>(null)
const preview = ref<{ title: string; url: string } | null>(null)
const previewText = ref('')

function onSearch(list:any[]){
  books.value = list
  picked.value = null
  preview.value = null
  previewText.value = ''
}

function onPick(b:any){
  picked.value = b
  preview.value = null
  previewText.value = ''
}

function onPickChapter(_c: { Title: string; Url: string }){
  // placeholder for select action
}

function onPreview(p: { Title: string; Url: string; content: string }){
  preview.value = { title: p.Title, url: p.Url }
  previewText.value = p.content
}
</script>
