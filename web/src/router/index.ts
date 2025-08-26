import { createRouter, createWebHistory } from 'vue-router'
import SearchPage from '@/pages/SearchPage.vue'
import BookDetail from '@/pages/BookDetail.vue'

const routes = [
  { path: '/', name: 'home', component: SearchPage, meta: { keepAlive: true } },
  { path: '/book', name: 'book', component: BookDetail, props: route => ({ id: route.query.id, title: route.query.title, source: route.query.source }) }
]

export default createRouter({ history: createWebHistory(), routes })
