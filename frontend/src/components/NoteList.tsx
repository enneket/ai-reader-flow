import {useState, useEffect} from 'react'
import {FileText, Trash2} from 'lucide-react'
import {GetNotes, ReadNote, DeleteNote} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function NoteList() {
  const [notes, setNotes] = useState<models.Note[]>([])
  const [selectedNote, setSelectedNote] = useState<models.Note | null>(null)
  const [noteContent, setNoteContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    loadNotes()
  }, [])

  const loadNotes = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await GetNotes()
      setNotes(data || [])
    } catch (err: any) {
      setError(err.message || 'Failed to load notes')
    } finally {
      setLoading(false)
    }
  }

  const handleSelectNote = async (note: models.Note) => {
    setSelectedNote(note)
    try {
      const content = await ReadNote(note.id)
      setNoteContent(content || '')
    } catch (err: any) {
      setError(err.message || 'Failed to read note')
      setNoteContent('')
    }
  }

  const handleDeleteNote = async (noteId: number, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await DeleteNote(noteId)
      if (selectedNote?.id === noteId) {
        setSelectedNote(null)
        setNoteContent('')
      }
      await loadNotes()
    } catch (err: any) {
      setError(err.message || 'Failed to delete note')
    }
  }

  const formatDate = (dateStr: string) => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  // Simple markdown formatting
  const formatMarkdown = (text: string): string => {
    if (!text) return ''

    let html = text
      .replace(/^### (.+)$/gm, '<h3>$1</h3>')
      .replace(/^## (.+)$/gm, '<h2>$1</h2>')
      .replace(/^# (.+)$/gm, '<h1>$1</h1>')
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/\*(.+?)\*/g, '<em>$1</em>')
      .replace(/\[(.+?)\]\((.+?)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
      .replace(/^> (.+)$/gm, '<blockquote>$1</blockquote>')
      .replace(/^---$/gm, '<hr>')
      .replace(/\n\n/g, '</p><p>')
      .replace(/\n/g, '<br>')

    html = '<p>' + html + '</p>'
    html = html.replace(/<p><\/p>/g, '')
    html = html.replace(/<p>(<h[1-3]>)/g, '$1')
    html = html.replace(/(<\/h[1-3]>)<\/p>/g, '$1')
    html = html.replace(/<p>(<blockquote>)/g, '$1')
    html = html.replace(/(<\/blockquote>)<\/p>/g, '$1')
    html = html.replace(/<p>(<hr>)<\/p>/g, '$1')

    return html
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">Notes</h1>
      </header>

      <div className="page-content" style={{padding: 0, height: 'calc(100vh - 73px)'}}>
        {error && (
          <div className="alert alert-error" style={{margin: 'var(--space-4)'}}>
            <span>{error}</span>
            <button className="alert-close" onClick={() => setError('')}>×</button>
          </div>
        )}

        <div className="notes-layout">
          <aside className="notes-sidebar">
            <div className="notes-sidebar-header">
              {notes.length} {notes.length === 1 ? 'Note' : 'Notes'}
            </div>
            <div className="notes-list">
              {notes.length === 0 ? (
                <div className="empty-state" style={{padding: 'var(--space-8)'}}>
                  <FileText />
                  <p>No notes yet. Save articles to create notes.</p>
                </div>
              ) : (
                notes.map((note) => (
                  <div
                    key={note.id}
                    className={`note-item ${selectedNote?.id === note.id ? 'selected' : ''}`}
                    onClick={() => handleSelectNote(note)}
                  >
                    <h4>{note.title || 'Untitled Note'}</h4>
                    <p className="note-date">{formatDate(note.created_at)}</p>
                    <button
                      onClick={(e) => handleDeleteNote(note.id, e)}
                      className="btn btn-ghost btn-sm btn-icon note-delete-btn"
                      aria-label="Delete note"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                ))
              )}
            </div>
          </aside>

          <div className="notes-content">
            {selectedNote ? (
              <div className="markdown-content" dangerouslySetInnerHTML={{__html: formatMarkdown(noteContent)}} />
            ) : (
              <div className="empty-state">
                <FileText />
                <p>Select a note to view its content</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  )
}
