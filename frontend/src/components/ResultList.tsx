import { useState } from 'react'
import { Empty, List, Skeleton, Tag, Typography, message } from 'antd'
import type { SearchResult } from '../types/trainer'
import { useAppStore } from '../store/appStore'

interface ResultListProps {
  results: SearchResult[]
}

function ResultList({ results }: ResultListProps) {
  const selectedDetail = useAppStore((state) => state.selectedDetail)
  const loading = useAppStore((state) => state.loading.detail)
  const loadDetail = useAppStore((state) => state.loadDetail)
  const [selectedURL, setSelectedURL] = useState<string | null>(null)

  const handleSelect = async (result: SearchResult) => {
    setSelectedURL(result.url)
    try {
      await loadDetail(result.url)
    } catch {
      message.error('详情加载失败，请稍后重试')
    }
  }

  if (loading && !selectedDetail) {
    return (
      <div className="result-list-card">
        <Skeleton active paragraph={{ rows: 5 }} />
        <Skeleton active paragraph={{ rows: 4 }} />
      </div>
    )
  }

  if (results.length === 0) {
    return (
      <div className="result-list-card result-list-empty">
        <Empty description="暂无搜索结果" />
      </div>
    )
  }

  return (
    <div className="result-list-card">
      <Typography.Title level={4}>搜索结果</Typography.Title>
      <List
        dataSource={results}
        renderItem={(result) => {
          const selected = selectedURL === result.url
          return (
            <List.Item
              className={`result-item${selected ? ' result-item-selected' : ''}`}
              onClick={() => void handleSelect(result)}
            >
              <List.Item.Meta
                title={result.title}
                description={
                  <div className="result-meta">
                    {result.version && <Tag color="blue">{result.version}</Tag>}
                    {result.updated && <span>更新于 {result.updated}</span>}
                    {result.options && (
                      <Typography.Text type="secondary" ellipsis>
                        {result.options}
                      </Typography.Text>
                    )}
                  </div>
                }
              />
            </List.Item>
          )
        }}
      />
    </div>
  )
}

export default ResultList
