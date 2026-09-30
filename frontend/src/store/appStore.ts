import { create } from 'zustand'
import {
  GetTrainerDetail,
  SearchTrainer,
} from '../../bindings/changeme/backend/services/trainerservice'
import type { SearchResult, TrainerDetail } from '../types/trainer'

interface LoadingState {
  search: boolean
  detail: boolean
  download: boolean
}

interface DownloadProgressState {
  percent: number
  speed: string
}

interface AppState {
  searchKeyword: string
  results: SearchResult[]
  selectedDetail: TrainerDetail | null
  loading: LoadingState
  downloadProgress: DownloadProgressState
  error: string | null
  setSearchKeyword: (keyword: string) => void
  search: (keyword?: string) => Promise<void>
  loadDetail: (detailURL: string) => Promise<void>
  setDownloadProgress: (progress: DownloadProgressState) => void
  clearError: () => void
}

export const useAppStore = create<AppState>((set, get) => ({
  searchKeyword: '',
  results: [],
  selectedDetail: null,
  loading: {
    search: false,
    detail: false,
    download: false,
  },
  downloadProgress: {
    percent: 0,
    speed: '',
  },
  error: null,
  setSearchKeyword: (keyword) => set({ searchKeyword: keyword }),
  search: async (keyword) => {
    const query = (keyword ?? get().searchKeyword).trim()
    if (!query) {
      set({ error: '请输入游戏名称' })
      return
    }

    set({
      searchKeyword: query,
      loading: { ...get().loading, search: true },
      error: null,
      selectedDetail: null,
    })
    try {
      const results = await SearchTrainer(query)
      set({ results: results ?? [] })
    } catch (error) {
      const message = error instanceof Error ? error.message : '搜索失败，请稍后重试'
      set({ results: [], error: message })
      throw error
    } finally {
      set({ loading: { ...get().loading, search: false } })
    }
  },
  loadDetail: async (detailURL) => {
    set({
      loading: { ...get().loading, detail: true },
      error: null,
    })
    try {
      const detail = await GetTrainerDetail(detailURL)
      set({ selectedDetail: detail })
    } catch (error) {
      const message = error instanceof Error ? error.message : '详情加载失败，请稍后重试'
      set({ error: message })
      throw error
    } finally {
      set({ loading: { ...get().loading, detail: false } })
    }
  },
  setDownloadProgress: (progress) => set({ downloadProgress: progress }),
  clearError: () => set({ error: null }),
}))
