import { useEffect, useState } from 'react'
import { notifications } from '../api/client'

function formatDate(ts) {
  if (!ts) return '—'
  return new Date(ts).toLocaleString()
}

export default function Notifications() {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [onlyUnread, setOnlyUnread] = useState(false)

  const load = async (unreadOnly = onlyUnread) => {
    setLoading(true)
    setError('')
    try {
      const data = await notifications.list({ unread_only: unreadOnly })
      setItems(data)
    } catch (err) {
      setError(err.message || 'Failed to load notifications')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load(onlyUnread)
  }, [onlyUnread])

  const markAllRead = async () => {
    try {
      await notifications.markAllRead()
      await load(onlyUnread)
    } catch (err) {
      setError(err.message || 'Failed to mark notifications read')
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold text-slate-800">Notifications</h1>
        <div className="flex items-center gap-3">
          <label className="text-sm text-slate-600 flex items-center gap-2">
            <input
              type="checkbox"
              checked={onlyUnread}
              onChange={(e) => setOnlyUnread(e.target.checked)}
            />
            Unread only
          </label>
          <button
            onClick={markAllRead}
            className="px-3 py-2 text-sm rounded border border-slate-300 hover:bg-slate-50"
          >
            Mark all read
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
        </div>
      ) : (
        <>
          {error && <p className="text-rose-600 mb-4">{error}</p>}
          {items.length === 0 ? (
            <p className="text-slate-500">No notifications.</p>
          ) : (
            <div className="space-y-3">
              {items.map((n) => (
                <div
                  key={n.id}
                  className={`rounded-lg border p-4 ${
                    n.is_read ? 'bg-white border-slate-200' : 'bg-rose-50 border-rose-200'
                  }`}
                >
                  <p className="text-sm text-slate-500 mb-1">
                    <span className="font-semibold text-slate-700">{n.type}</span> • {formatDate(n.created_at)}
                  </p>
                  <p className="text-slate-800">{n.content}</p>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
