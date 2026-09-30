import { Layout, Typography } from 'antd'
import SearchBar from './containers/SearchBar'
import MainLayout from './containers/MainLayout'
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
        <MainLayout />
      </Content>
    </Layout>
  )
}

export default App
