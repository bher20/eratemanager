import { Routes, Route } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import { DashboardPage, ElectricPage, WaterPage, SettingsPage } from '@/pages'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<DashboardPage />} />
        <Route path="/electric" element={<ElectricPage />} />
        <Route path="/water" element={<WaterPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Routes>
    </Layout>
  )
}

export default App
