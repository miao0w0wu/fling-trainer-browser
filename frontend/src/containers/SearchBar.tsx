import { SearchOutlined } from '@ant-design/icons'
import { Input, message } from 'antd'
import { useAppStore } from '../store/appStore'

function SearchBar() {
  const keyword = useAppStore((state) => state.searchKeyword)
  const loading = useAppStore((state) => state.loading.search)
  const setSearchKeyword = useAppStore((state) => state.setSearchKeyword)
  const search = useAppStore((state) => state.search)

  const handleSearch = async (value: string) => {
    try {
      await search(value)
    } catch {
      message.error('搜索失败，请检查网络连接后重试')
    }
  }

  return (
    <Input.Search
      allowClear
      enterButton={<SearchOutlined />}
      loading={loading}
      placeholder="搜索游戏名称，例如 Elden Ring"
      size="large"
      value={keyword}
      onChange={(event) => setSearchKeyword(event.target.value)}
      onSearch={handleSearch}
    />
  )
}

export default SearchBar
