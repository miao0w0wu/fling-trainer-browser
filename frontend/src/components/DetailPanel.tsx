import { DownloadOutlined } from '@ant-design/icons'
import {
  Button,
  Descriptions,
  Empty,
  Image,
  List,
  Skeleton,
  Tag,
  Typography,
} from 'antd'
import { useAppStore } from '../store/appStore'

function DetailPanel() {
  const detail = useAppStore((state) => state.selectedDetail)
  const loading = useAppStore((state) => state.loading.detail)

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

      <Button
        block
        disabled={!detail.downloadUrl}
        icon={<DownloadOutlined />}
        size="large"
        type="primary"
      >
        下载修改器
      </Button>
    </div>
  )
}

export default DetailPanel
