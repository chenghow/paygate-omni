import { ElMessage } from 'element-plus'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'
const REQUEST_TIMEOUT = 10000 // 10s

/**
 * 统一的 API 请求服务类
 * 包含：请求拦截、响应处理、错误捕获、token 管理
 */
class ApiService {
  private baseURL: string

  constructor() {
    this.baseURL = API_BASE_URL
  }

  /**
   * 获取 token
   */
  private getToken(): string | null {
    return localStorage.getItem('admin_token')
  }

  /**
   * 清除 token（退出登录）
   */
  clearToken(): void {
    localStorage.removeItem('admin_token')
  }

  /**
   * 统一的请求方法
   */
  async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`
    const token = this.getToken()

    // 准备请求头
    const headers = new Headers(options.headers || {})
    headers.set('Content-Type', 'application/json')
    
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }

    // 构建请求
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT)

    try {
      const response = await fetch(url, {
        ...options,
        headers,
        signal: controller.signal
      })

      clearTimeout(timeoutId)

      // 解析响应
      const data = await response.json()

      // 处理 401（token 过期或无效）
      if (response.status === 401) {
        this.clearToken()
        window.location.hash = '#/login'
        ElMessage.error('登录已过期，请重新登录')
        throw new Error('Unauthorized')
      }

      // 处理错误响应
      if (!response.ok || data.code !== 'SUCCESS') {
        const errorMsg = data.message || `请求失败: ${response.status}`
        ElMessage.error(errorMsg)
        throw new Error(errorMsg)
      }

      return data.data || data
    } catch (error: any) {
      clearTimeout(timeoutId)

      // 处理超时
      if (error.name === 'AbortError') {
        ElMessage.error('请求超时，请重试')
        throw new Error('Request timeout')
      }

      // 处理网络错误
      if (error instanceof TypeError) {
        ElMessage.error('网络连接失败，请检查网络')
      }

      throw error
    }
  }

  // GET 方法
  get<T>(endpoint: string, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'GET'
    })
  }

  // POST 方法
  post<T>(endpoint: string, body?: any, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: JSON.stringify(body)
    })
  }

  // PUT 方法
  put<T>(endpoint: string, body?: any, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(body)
    })
  }

  // DELETE 方法
  delete<T>(endpoint: string, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'DELETE'
    })
  }
}

export const apiService = new ApiService()
