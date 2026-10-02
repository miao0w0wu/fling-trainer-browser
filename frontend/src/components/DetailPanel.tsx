import { DownloadOutlined } from '@ant-design/icons'
import {
  Button,
  Descriptions,
  Empty,
  Image,
  List,
  Skeleton,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import type { TableColumnsType } from 'antd'
import { useAppStore } from '../store/appStore'
import type { DownloadOption } from '../types/trainer'

const groupColors = ['purple', 'geekblue', 'cyan', 'orange']

function formatDownloadCount(value: string): string {
  if (!value || value.trim() === '') {
    return ''
  }
  const count = Number(value.replace(/,/g, ''))
  return Number.isFinite(count) ? count.toLocaleString() : value
}

function downloadOptionMeta(option: DownloadOption): string {
  const count = formatDownloadCount(option.downloads)
  return [option.dateAdded, option.fileSize, count ? `${count} 次下载` : '']
    .filter(Boolean)
    .join(' · ')
}

function DetailPanel() {
  const detail = useAppStore((state) => state.selectedDetail)
  const loading = useAppStore((state) => state.loading.detail)
  const downloading = useAppStore((state) => state.loading.download)
  const downloadTrainer = useAppStore((state) => state.downloadTrainer)

  if (loading) {
    return (
      <div className="detail-panel">
        <Skeleton active avatar paragraph={{ rows: 8 }} />
      </div>
    )
  }

  if (!detail) {
    return (
      <div className="detail-panel detail-panel-empty">
        <Empty description="选择一个修改器查看详情" />
      </div>
    )
  }

  const options = detail.options ?? []
  const images = detail.images ?? []
  const downloadOptions = detail.downloadOptions ?? []

  const groupToColor = new Map<string, string>()
  downloadOptions.forEach((option) => {
    if (option.group && !groupToColor.has(option.group)) {
      groupToColor.set(option.group, groupColors[groupToColor.size % groupColors.length])
    }
  })
  const showGroupTags = groupToColor.size > 1

  const handleDownload = async (downloadURL: string) => {
    try {
      const path = await downloadTrainer(downloadURL)
      message.success(`下载完成：${path}`)
    } catch {
      message.error('下载失败，请稍后重试')
    }
  }

  const downloadColumns: TableColumnsType<DownloadOption> = [
    {
      title: '文件',
      dataIndex: 'name',
      render: (_, option) => (
        <div className="download-option">
          {showGroupTags && option.group && (
            <Tag color={groupToColor.get(option.group)} style={{ marginBottom: 4 }}>
              {option.group}
            </Tag>
          )}
          <div className="download-option-name">{option.name}</div>
          <Typography.Text type="secondary" className="download-option-meta">
            {downloadOptionMeta(option)}
          </Typography.Text>
        </div>
      ),
    },
    {
      title: '',
      key: 'action',
      width: 88,
      align: 'right',
      render: (_, option) => (
        <Button
          disabled={downloading || !option.url}
          icon={<DownloadOutlined />}
          size="small"
          onClick={() => void handleDownload(option.url)}
        >
          下载
        </Button>
      ),
    },
  ]

  return (
    <div className="detail-panel">
      <div className="detail-images">
        {images.length > 0 ? (
          <Image.PreviewGroup>
            {images.map((image) => (
              <Image
                key={image}
                className="detail-cover"
                height={220}
                src={image}
                fallback="/wails.png"
                preview
              />
            ))}
          </Image.PreviewGroup>
        ) : (
          <div className="detail-cover-placeholder">暂无封面图</div>
        )}
      </div>

      <Typography.Title level={3}>{detail.title}</Typography.Title>
      <Descriptions
        bordered
        column={1}
        items={[
          { key: 'version', label: '游戏版本', children: detail.gameVersion || '未知' },
          { key: 'updated', label: '最后更新', children: detail.lastUpdated || '未知' },
          { key: 'source', label: '来源', children: detail.sourceUrl },
        ]}
      />

      <Typography.Title level={4}>修改选项</Typography.Title>
      {options.length > 0 ? (
        <List
          bordered
          size="small"
          dataSource={options}
          renderItem={(option) => (
            <List.Item>
              <Tag color="geekblue">{option}</Tag>
            </List.Item>
          )}
        />
      ) : (
        <Typography.Text type="secondary">暂无选项信息</Typography.Text>
      )}

      <Typography.Title level={4}>描述</Typography.Title>
      <Typography.Paragraph ellipsis={{ rows: 5, expandable: true, symbol: '展开' }}>
        {detail.description || '暂无描述'}
      </Typography.Paragraph>

      <Typography.Title level={4}>下载链接</Typography.Title>
      {downloadOptions.length > 0 ? (
        <Table<DownloadOption>
          className="download-options-table"
          size="small"
          rowKey={(option) => option.url}
          dataSource={downloadOptions}
          pagination={false}
          columns={downloadColumns}
        />
      ) : detail.downloadUrl ? (
        <Button
          block
          disabled={!detail.downloadUrl}
          icon={<DownloadOutlined />}
          loading={downloading}
          size="large"
          type="primary"
          onClick={() => void handleDownload(detail.downloadUrl)}
        >
          下载修改器
        </Button>
      ) : (
        <Typography.Text type="secondary">暂无下载链接</Typography.Text>
      )}
    </div>
  )
}

export default DetailPanel
