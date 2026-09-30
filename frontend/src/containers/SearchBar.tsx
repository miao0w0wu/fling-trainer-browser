import { SearchOutlined } from '@ant-design/icons'
import { Input, message, Tag, Tooltip, type InputRef } from 'antd'
import { useEffect, useRef } from 'react'
import { useAppStore } from '../store/appStore'
import { useDebounce } from '../hooks/useDebounce'

function SearchBar() {
  const keyword = useAppStore((state) => state.searchKeyword)
  const loading = useAppStore((state) => state.loading.search)
  const setSearchKeyword = useAppStore((state) => state.setSearchKeyword)
  const search = useAppStore((state) => state.search)
  const recentSearches = useAppStore((state) => state.recentSearches)
  const debouncedKeyword = useDebounce(keyword, 300)
  const inputRef = useRef<InputRef>(null)

  useEffect(() => {
    const handleShortcut = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        inputRef.current?.focus()
      }
    }
    window.addEventListener('keydown', handleShortcut)
    return () => window.removeEventListener('keydown', handleShortcut)
  }, [])

  const handleSearch = async () => {
    try {
      await search(debouncedKeyword)
    } catch {
      message.error('搜索失败，请检查网络连接后重试')
    }
  }

  return (
    <div>
      <Tooltip title={`按 ${navigator.platform.includes('Mac') ? '⌘' : 'Ctrl'}+K 聚焦搜索`}>
        <Input.Search
          ref={inputRef}
          allowClear
          enterButton={<SearchOutlined />}
          loading={loading}
          placeholder="搜索游戏名称，例如 Elden Ring"
          size="large"
          value={keyword}
          onChange={(event) => setSearchKeyword(event.target.value)}
          onSearch={handleSearch}
        />
      </Tooltip>
      {recentSearches.length > 0 && (
        <div className="recent-searches">
          <span>最近：</span>
          {recentSearches.slice(0, 5).map((recent) => (
            <Tag
              key={recent}
              className="recent-search-tag"
              onClick={() => {
                setSearchKeyword(recent)
                void search(recent)
              }}
            >
              {recent}
            </Tag>
          ))}
        </div>
      )}
    </div>
  )
}

export default SearchBar
