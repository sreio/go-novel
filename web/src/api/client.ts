import http, { ApiError } from './http'

export interface SearchItem { title: string; author: string; id: string; source: string }
export interface ChapterRow { title: string; url: string; index: number }

export async function apiHealth(): Promise<'ok'> {
  const { data } = await http.get<string>('/health')
  return data as any
}

export async function apiSearch(q: string): Promise<SearchItem[]> {
  try {
    const { data } = await http.get<{ items: SearchItem[] }>('/search', { params: { q } })
    return data.items || []
  } catch (e) {
    throw e as ApiError
  }
}

export async function apiChapters(url: string): Promise<ChapterRow[]> {
  try {
    const { data } = await http.get<{ chapters: ChapterRow[] }>('/books/chapters', { params: { url } })
    return data.chapters || []
  } catch (e) {
    throw e as ApiError
  }
}

export async function apiDownload(url: string, format: 'txt'|'epub'|'pdf'): Promise<Blob> {
  try {
    const { data } = await http.get(`/download`, {
      params: { url, format },
      responseType: 'blob',
    })
    if (data && (data as any).type === 'application/json') {
      const txt = await (data as any).text()
      try { const obj = JSON.parse(txt); throw { code: 0, message: obj?.error || 'Download error', detail: obj } } catch {}
    }
    return data
  } catch (e) {
    throw e as ApiError
  }
}

export async function apiChapter(url: string, opts?: { limit?: number; full?: boolean }): Promise<{url:string; title:string; content:string; full:boolean; limit:number}> {
  const { data } = await http.get('/chapter', {
    params: { url, limit: opts?.limit ?? 1000, full: opts?.full ? 1 : 0 },
  })
  return data
}
