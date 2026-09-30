import { Layout, message, Typography } from 'antd'
import { useEffect } from 'react'
import SearchBar from './containers/SearchBar'
import MainLayout from './containers/MainLayout'
import DownloadBar from './components/DownloadBar'
import { useDownloadProgress } from './hooks/useDownloadProgress'
import { useAppStore } from './store/appStore'
import './app.css'

const { Header, Content } = Layout

function App() {
  useDownloadProgress()
  const error = useAppStore((state) => state.error)
  const clearError = useAppStore((state) => state.clearError)

  useEffect(() => {
    if (error) {
      message.error(error)
      clearError()
    }
  }, [clearError, error])

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
      <DownloadBar />
    </Layout>
  )
}

export default App
