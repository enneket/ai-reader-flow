import {Routes, Route} from 'react-router-dom'
import {ArticleList} from './components/ArticleList'
import {FeedList} from './components/FeedList'
import {NoteList} from './components/NoteList'
import {Settings} from './components/Settings'

function App() {
  return (
    <Routes>
      {/* Main reading view — The Magazine layout */}
      <Route path="/" element={<ArticleList />} />
      <Route path="/articles" element={<ArticleList />} />
      <Route path="/articles/:feedId" element={<ArticleList />} />

      {/* Feeds page — manage subscriptions */}
      <Route path="/feeds" element={<FeedList />} />

      {/* Notes page — standalone layout */}
      <Route path="/notes" element={<NoteList />} />

      {/* Settings page — standalone layout */}
      <Route path="/settings" element={<Settings />} />
    </Routes>
  )
}

export default App
