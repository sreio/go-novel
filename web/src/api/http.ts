import axios, { AxiosError } from 'axios'

export interface ApiError { code: number; message: string; detail?: any }

const http = axios.create({ baseURL: '/api', timeout: 60000 })

http.interceptors.response.use(
  resp => resp,
  (error: AxiosError) => {
    const err: ApiError = {
      code: error.response?.status || 0,
      message: (error.response?.data as any)?.error || error.message || 'Network Error',
      detail: error.response?.data
    }
    return Promise.reject(err)
  }
)

export default http
