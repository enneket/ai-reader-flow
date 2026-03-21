import {useState, useEffect} from 'react'
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
            setNoteContent(content)
        } catch (err: any) {
            setError(err.message || 'Failed to read note')
            setNoteContent('')
        }
    }

    const handleDeleteNote = async (noteId: number) => {
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
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        })
    }

    return (
        <div className="note-list">
            <div className="note-header">
                <h2>Notes</h2>
            </div>

            {error && <div className="error">{error}</div>}

            <div className="note-container">
                <div className="note-sidebar">
                    {notes.length === 0 ? (
                        <p className="empty-state">No notes yet. Save some articles first.</p>
                    ) : (
                        <ul className="notes">
                            {notes.map((note) => (
                                <li
                                    key={note.id}
                                    className={`note-item ${selectedNote?.id === note.id ? 'selected' : ''}`}
                                    onClick={() => handleSelectNote(note)}
                                >
                                    <h4>{note.title}</h4>
                                    <p className="note-date">{formatDate(note.created_at)}</p>
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation()
                                            handleDeleteNote(note.id)
                                        }}
                                        className="btn-delete-small"
                                    >
                                        Delete
                                    </button>
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
                <div className="note-content">
                    {selectedNote ? (
                        <div className="markdown-content">
                            <div dangerouslySetInnerHTML={{__html: formatMarkdown(noteContent)}} />
                        </div>
                    ) : (
                        <p className="empty-state">Select a note to view its content</p>
                    )}
                </div>
            </div>
        </div>
    )
}

// Simple markdown formatting
function formatMarkdown(text: string): string {
    if (!text) return ''

    let html = text
        // Headers
        .replace(/^### (.+)$/gm, '<h3>$1</h3>')
        .replace(/^## (.+)$/gm, '<h2>$1</h2>')
        .replace(/^# (.+)$/gm, '<h1>$1</h1>')
        // Bold
        .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
        // Italic
        .replace(/\*(.+?)\*/g, '<em>$1</em>')
        // Links
        .replace(/\[(.+?)\]\((.+?)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
        // Blockquotes
        .replace(/^> (.+)$/gm, '<blockquote>$1</blockquote>')
        // Horizontal rules
        .replace(/^---$/gm, '<hr>')
        // Line breaks
        .replace(/\n\n/g, '</p><p>')
        .replace(/\n/g, '<br>')

    // Wrap in paragraphs
    html = '<p>' + html + '</p>'

    // Clean up empty paragraphs
    html = html.replace(/<p><\/p>/g, '')
    html = html.replace(/<p>(<h[1-3]>)/g, '$1')
    html = html.replace(/(<\/h[1-3]>)<\/p>/g, '$1')
    html = html.replace(/<p>(<blockquote>)/g, '$1')
    html = html.replace(/(<\/blockquote>)<\/p>/g, '$1')
    html = html.replace(/<p>(<hr>)<\/p>/g, '$1')

    return html
}
