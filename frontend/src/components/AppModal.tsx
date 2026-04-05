import { useEffect, useRef, useCallback, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { AlertTriangle, XCircle, CheckCircle } from 'lucide-react'

interface AppModalProps {
  type: 'warning' | 'error' | 'success'
  title: string
  content: string
  onOk?: () => void
  autoClose?: number
}

const style = `
  .app-modal-overlay {
    position: fixed; inset: 0; background: rgba(0,0,0,0.6);
    display: flex; align-items: center; justify-content: center;
    z-index: 9999; font-family: var(--font-sans);
  }
  .app-modal-box {
    background: var(--surface); border-radius: var(--radius);
    padding: 24px; min-width: 320px; max-width: 400px;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
  }
  .app-modal-box--warning { border: 1px solid var(--accent); }
  .app-modal-box--error { border: 1px solid var(--danger); }
  .app-modal-box--success { border: 1px solid var(--success); }
  .app-modal-header { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
  .app-modal-icon { flex-shrink: 0; }
  .app-modal-box--warning .app-modal-icon { color: var(--accent); }
  .app-modal-box--error .app-modal-icon { color: var(--danger); }
  .app-modal-box--success .app-modal-icon { color: var(--success); }
  .app-modal-title { color: var(--text-primary); font-size: 16px; font-weight: 600; }
  .app-modal-content { color: var(--text-secondary); font-size: 14px; line-height: 1.5; margin-bottom: 20px; }
  .app-modal-footer { display: flex; justify-content: flex-end; }
  .app-modal-ok {
    background: var(--accent); color: #fff; border: none; border-radius: var(--radius);
    padding: 8px 20px; font-size: 14px; cursor: pointer; font-weight: 500;
    transition: background var(--transition-fast);
  }
  .app-modal-ok:hover { background: var(--accent-hover); }
  .app-modal-ok:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
` as const

export function AppModal({ type, title, content, onOk, autoClose }: AppModalProps): ReactNode {
  const overlayRef = useRef<HTMLDivElement>(null)
  const okButtonRef = useRef<HTMLButtonElement>(null)

  const close = useCallback(() => {
    onOk?.()
  }, [onOk])

  // Close on ESC
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [close])

  // Auto close
  useEffect(() => {
    if (!autoClose) return
    const timer = setTimeout(close, autoClose)
    return () => clearTimeout(timer)
  }, [autoClose, close])

  // Focus trap + initial focus
  useEffect(() => {
    okButtonRef.current?.focus()
    const handleTab = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return
      const focusable = overlayRef.current?.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      )
      if (!focusable || focusable.length === 0) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', handleTab)
    return () => document.removeEventListener('keydown', handleTab)
  }, [])

  const Icon = type === 'warning' ? AlertTriangle : type === 'error' ? XCircle : CheckCircle

  const modal = (
    <div
      className="app-modal-overlay"
      ref={overlayRef}
      onClick={(e) => { if (e.target === overlayRef.current) close() }}
    >
      <div className={`app-modal-box app-modal-box--${type}`} role="dialog" aria-modal="true" aria-labelledby="app-modal-title">
        <div className="app-modal-header">
          <Icon size={22} className="app-modal-icon" />
          <span id="app-modal-title" className="app-modal-title">{title}</span>
        </div>
        <p className="app-modal-content">{content}</p>
        {onOk && (
          <div className="app-modal-footer">
            <button className="app-modal-ok" ref={okButtonRef} onClick={close}>确定</button>
          </div>
        )}
      </div>
    </div>
  )

  return createPortal(modal, document.body)
}

// Inject styles once
let styleInjected = false
export function injectAppModalStyles() {
  if (styleInjected) return
  styleInjected = true
  const el = document.createElement('style')
  el.textContent = style
  document.head.appendChild(el)
}
