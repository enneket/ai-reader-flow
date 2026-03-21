import {Routes, Route, Link, useLocation} from 'react-router-dom'
import {FeedList} from './components/FeedList'
import {ArticleList} from './components/ArticleList'
import {NoteList} from './components/NoteList'
import {Settings} from './components/Settings'
import './App.css'

function App() {
  const location = useLocation()

  return (
      <div className="app">
        <header className="app-header">
          <h1>AI RSS Reader</h1>
          <nav>
            <Link to="/" className={location.pathname === '/' ? 'active' : ''}>Feeds</Link>
            <Link to="/articles" className={location.pathname === '/articles' ? 'active' : ''}>Articles</Link>
            <Link to="/notes" className={location.pathname === '/notes' ? 'active' : ''}>Notes</Link>
            <Link to="/settings" className={location.pathname === '/settings' ? 'active' : ''}>Settings</Link>
          </nav>
        </header>
        <main className="app-main">
          <Routes>
            <Route path="/" element={<FeedList />} />
            <Route path="/articles" element={<ArticleList />} />
            <Route path="/articles/:feedId" element={<ArticleList />} />
            <Route path="/notes" element={<NoteList />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
  )
}

export default App
