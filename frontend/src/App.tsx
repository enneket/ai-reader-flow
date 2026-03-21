import {Routes, Route} from 'react-router-dom'
import {Layout} from './components/Layout'
import {FeedList} from './components/FeedList'
import {ArticleList} from './components/ArticleList'
import {NoteList} from './components/NoteList'
import {Settings} from './components/Settings'

function App() {
  return (
    <Routes>
      {/* Articles page uses its own full-width layout */}
      <Route path="/articles" element={<ArticleList />} />
      <Route path="/articles/:feedId" element={<ArticleList />} />

      {/* Home page with Layout */}
      <Route path="/" element={<Layout><FeedList /></Layout>} />

      {/* Notes page with Layout */}
      <Route path="/notes" element={<Layout><NoteList /></Layout>} />

      {/* Settings page with Layout */}
      <Route path="/settings" element={<Layout><Settings /></Layout>} />
    </Routes>
  )
}

export default App
