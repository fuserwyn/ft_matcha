import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { profile } from '../api/client'

export default function Views() {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      setLoading(true)
      setError('')
      try {
        const data = await profile.getViewedHistory({ limit: 100 })
        if (active) setItems(data)
      } catch (err) {
        if (active) setError(err.message || 'Failed to load history')
      } finally {
        if (active) setLoading(false)
      }
    })()
    return () => {
      active = false
    }
  }, [])

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-slate-800 mb-6">Profile views</h1>
      {error && <p className="text-rose-600 mb-4">{error}</p>}
      {items.length === 0 ? (
        <p className="text-slate-500">No profiles viewed yet.</p>
      ) : (
        <div className="space-y-3">
          {items.map((u) => (
            <div
              key={u.id}
              className="bg-white rounded-lg border border-slate-200 p-4 flex items-center justify-between gap-3"
            >
              <div className="flex items-center gap-3">
                {u.primary_photo_url ? (
                  <img
                    src={u.primary_photo_url}
                    alt={`${u.first_name} ${u.last_name}`}
                    className="w-14 h-14 object-cover rounded-full shrink-0"
                    referrerPolicy="no-referrer"
                  />
                ) : (
                  <div className="w-14 h-14 rounded-full bg-slate-200 shrink-0 flex items-center justify-center text-slate-500 text-lg font-medium">
                    {(u.first_name?.[0] || u.username?.[0] || '?').toUpperCase()}
                  </div>
                )}
                <div>
                  <p className="font-semibold text-slate-800">
                    {u.first_name} {u.last_name}
                  </p>
                  <p className="text-sm text-slate-500">@{u.username}</p>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Viewed {new Date(u.last_viewed_at).toLocaleString()}
                  </p>
                </div>
              </div>
              <Link
                to={`/users/${u.id}`}
                className="px-3 py-2 rounded bg-rose-500 text-sm text-white hover:bg-rose-600"
              >
                View profile
              </Link>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
