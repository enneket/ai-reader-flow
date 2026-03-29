import {Routes, Route} from 'react-router-dom'
import {Briefing} from './components/Briefing'
import {FeedList} from './components/FeedList'
import {Settings} from './components/Settings'

function App() {
  return (
    <Routes>
      {/* Briefing page — AI-generated daily briefing */}
      <Route path="/" element={<Briefing />} />

      {/* Feeds page — manage subscriptions */}
      <Route path="/feeds" element={<FeedList />} />

      {/* Settings page */}
      <Route path="/settings" element={<Settings />} />
    </Routes>
  )
}

export default App
