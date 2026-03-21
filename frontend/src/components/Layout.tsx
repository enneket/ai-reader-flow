import {Link, useLocation} from 'react-router-dom'
import {Rss, FileText, Settings, LayoutGrid} from 'lucide-react'

interface LayoutProps {
  children: React.ReactNode
}

export function Layout({children}: LayoutProps) {
  const location = useLocation()

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="sidebar-header">
          <div className="sidebar-logo">
            <Rss size={24} />
            <span>AI RSS</span>
          </div>
        </div>

        <nav className="sidebar-nav">
          <Link
            to="/"
            className={`nav-item ${isActive('/') && location.pathname === '/' ? 'active' : ''}`}
          >
            <LayoutGrid />
            <span>Feeds</span>
          </Link>
          <Link
            to="/articles"
            className={`nav-item ${isActive('/articles') ? 'active' : ''}`}
          >
            <FileText />
            <span>Articles</span>
          </Link>
          <Link
            to="/notes"
            className={`nav-item ${isActive('/notes') ? 'active' : ''}`}
          >
            <FileText />
            <span>Notes</span>
          </Link>
          <Link
            to="/settings"
            className={`nav-item ${isActive('/settings') ? 'active' : ''}`}
          >
            <Settings />
            <span>Settings</span>
          </Link>
        </nav>

        <div className="sidebar-footer">
          <div style={{fontSize: '12px', color: 'var(--text-muted)'}}>
            AI RSS Reader v1.0
          </div>
        </div>
      </aside>

      <main className="app-main">
        {children}
      </main>
    </div>
  )
}
