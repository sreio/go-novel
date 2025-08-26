import http from './http'

export interface SearchItem { title: string; author: string; id: string; source: string, category: string, update: string }
export interface ChapterRow { title: string; url: string; index: number }
export interface Chapters {chapters: ChapterRow[], source: string }

export async function apiSearch(q: string): Promise<SearchItem[]> {
  const { data } = await http.get<{ items: SearchItem[] }>('/search', { params: { q } })
  return data.items || []
}

export async function apiChapters(url: string): Promise<Chapters> {
  const { data } = await http.get('/books/chapters', { params: { url } })
  return data
}

export async function apiChapter(url: string, opts?: { limit?: number; full?: boolean }): Promise<{url:string; title:string; content:string; full:boolean; limit:number}> {
  const { data } = await http.get('/chapter', { params: { url, limit: opts?.limit ?? 1000, full: opts?.full ? 1 : 0 } })
  return data
}

export async function apiDownload(url: string, format: 'txt'|'epub'|'pdf'): Promise<Blob> {
  const { data } = await http.get(`/download`, { params: { url, format }, responseType: 'blob' })
  return data
}
