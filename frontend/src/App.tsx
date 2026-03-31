import {Routes, Route, Navigate} from 'react-router-dom'
import {Briefing} from './components/Briefing'
import {BriefingDetail} from './components/BriefingDetail'
import {FeedList} from './components/FeedList'
import {Settings} from './components/Settings'

function App() {
  return (
    <Routes>
      {/* Default redirect to feeds */}
      <Route path="/" element={<Navigate to="/feeds" replace />} />

      {/* Feeds page — manage subscriptions */}
      <Route path="/feeds" element={<FeedList />} />

      {/* Briefing page — AI-generated daily briefing */}
      <Route path="/briefing" element={<Briefing />} />

      {/* Briefing detail page */}
      <Route path="/briefing/:id" element={<BriefingDetail />} />

      {/* Settings page */}
      <Route path="/settings" element={<Settings />} />
    </Routes>
  )
}

export default App
