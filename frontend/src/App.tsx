import {Routes, Route} from 'react-router-dom'
import {Layout} from './components/Layout'
import {ArticleList} from './components/ArticleList'
import {NoteList} from './components/NoteList'
import {Settings} from './components/Settings'

function App() {
  return (
    <Routes>
      {/* Main reading view — The Magazine layout */}
      <Route path="/" element={<ArticleList />} />
      <Route path="/articles" element={<ArticleList />} />
      <Route path="/articles/:feedId" element={<ArticleList />} />

      {/* Notes page — keeps Layout with sidebar */}
      <Route path="/notes" element={<Layout><NoteList /></Layout>} />

      {/* Settings page — keeps Layout with sidebar */}
      <Route path="/settings" element={<Layout><Settings /></Layout>} />
    </Routes>
  )
}

export default App
