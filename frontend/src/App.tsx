import {Routes, Route} from 'react-router-dom'
import {Layout} from './components/Layout'
import {FeedList} from './components/FeedList'
import {ArticleList} from './components/ArticleList'
import {NoteList} from './components/NoteList'
import {Settings} from './components/Settings'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<FeedList />} />
        <Route path="/articles" element={<ArticleList />} />
        <Route path="/articles/:feedId" element={<ArticleList />} />
        <Route path="/notes" element={<NoteList />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Layout>
  )
}

export default App
