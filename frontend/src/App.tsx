import { Button, Space, Typography } from 'antd'

function App() {
  return (
    <Space
      direction="vertical"
      size="large"
      style={{
        display: 'flex',
        minHeight: '100vh',
        justifyContent: 'center',
        alignItems: 'center',
      }}
    >
      <Typography.Title level={2}>FLiNG Trainer Browser</Typography.Title>
      <Button type="primary">Ant Design 已就绪</Button>
    </Space>
  )
}

export default App
