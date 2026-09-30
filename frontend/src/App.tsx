import { Layout, Typography } from 'antd'
import SearchBar from './containers/SearchBar'
import './app.css'

const { Header, Content } = Layout

function App() {
  return (
    <Layout className="app-layout">
      <Header className="app-header">
        <div className="app-brand">
          <Typography.Title level={3}>FLiNG Trainer Browser</Typography.Title>
          <Typography.Text type="secondary">
            Search PC game trainers
          </Typography.Text>
        </div>
        <div className="app-search">
          <SearchBar />
        </div>
      </Header>
      <Content className="app-content">
        <div className="empty-content">
          <Typography.Title level={4}>搜索游戏修改器</Typography.Title>
          <Typography.Paragraph type="secondary">
            输入游戏名称开始搜索。
          </Typography.Paragraph>
        </div>
      </Content>
    </Layout>
  )
}

export default App
