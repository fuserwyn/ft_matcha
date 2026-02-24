import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { matches } from '../api/client'

export default function Matches() {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      setLoading(true)
      setError('')
      try {
        const data = await matches.list()
        if (active) setItems(data)
      } catch (err) {
        if (active) setError(err.message || 'Failed to load matches')
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
      <h1 className="text-2xl font-bold text-slate-800 mb-6">Matches</h1>
      {error && <p className="text-rose-600 mb-4">{error}</p>}
      {items.length === 0 ? (
        <p className="text-slate-500">No matches yet.</p>
      ) : (
        <div className="space-y-3">
          {items.map((u) => (
            <div
              key={u.id}
              className="bg-white rounded-lg border border-slate-200 p-4 flex items-center justify-between gap-3"
            >
              <div>
                <p className="font-semibold text-slate-800">
                  {u.first_name} {u.last_name}
                </p>
                <p className="text-sm text-slate-500">@{u.username}</p>
              </div>
              <div className="flex gap-2">
                <Link
                  to={`/users/${u.id}`}
                  className="px-3 py-2 rounded border border-slate-300 text-sm text-slate-700 hover:bg-slate-50"
                >
                  Profile
                </Link>
                <Link
                  to={`/chat/${u.id}`}
                  className="px-3 py-2 rounded bg-rose-500 text-sm text-white hover:bg-rose-600"
                >
                  Chat
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
