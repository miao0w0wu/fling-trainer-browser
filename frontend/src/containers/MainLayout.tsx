import { Col, Row } from 'antd'
import ResultList from '../components/ResultList'
import DetailPanel from '../components/DetailPanel'
import { useAppStore } from '../store/appStore'

function MainLayout() {
  const results = useAppStore((state) => state.results)

  return (
    <Row className="main-layout" gutter={[24, 24]}>
      <Col xs={24} lg={10}>
        <ResultList results={results} />
      </Col>
      <Col xs={24} lg={14}>
        <DetailPanel />
      </Col>
    </Row>
  )
}

export default MainLayout
