import { FolderOpenOutlined } from '@ant-design/icons'
import { Button, Progress, Space, Typography } from 'antd'
import { useAppStore } from '../store/appStore'

function DownloadBar() {
  const downloading = useAppStore((state) => state.loading.download)
  const progress = useAppStore((state) => state.downloadProgress)

  if (!downloading && progress.percent <= 0) {
    return null
  }

  const completed = !downloading && progress.percent >= 100

  return (
    <div className="download-bar">
      <div className="download-bar-content">
        <div className="download-bar-label">
          <Typography.Text strong>
            {completed ? '下载完成' : '正在下载修改器'}
          </Typography.Text>
          <Typography.Text type="secondary">{progress.speed || '准备中'}</Typography.Text>
        </div>
        <Progress
          percent={Math.min(100, Math.max(0, progress.percent))}
          status={completed ? 'success' : 'active'}
          strokeColor={completed ? '#52c41a' : undefined}
        />
        {completed && (
          <Space>
            <Button icon={<FolderOpenOutlined />} size="small">
              打开下载目录
            </Button>
          </Space>
        )}
      </div>
    </div>
  )
}

export default DownloadBar
