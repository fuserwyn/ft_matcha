import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { likes, users } from '../api/client'

const TABS = [
  { id: 'by-me', label: 'I liked', fetch: likes.listByMe },
  { id: 'liked-me', label: 'Liked me', fetch: likes.listLikedMe },
]

export default function Likes() {
  const [activeTab, setActiveTab] = useState('by-me')
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionId, setActionId] = useState(null)

  const currentTab = TABS.find((t) => t.id === activeTab)
  const isByMe = activeTab === 'by-me'

  useEffect(() => {
    let active = true
    setLoading(true)
    setError('')
    currentTab
      .fetch({ limit: 100 })
      .then((data) => {
        if (active) setItems(data)
      })
      .catch((err) => {
        if (active) setError(err.message || 'Failed to load likes')
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [activeTab])

  const handleUnlike = async (userId) => {
    setActionId(userId)
    setError('')
    try {
      await users.unlike(userId)
      setItems((prev) => prev.filter((u) => u.id !== userId))
    } catch (err) {
      setError(err.message || 'Failed to unlike')
    } finally {
      setActionId(null)
    }
  }

  const handleLike = async (userId) => {
    setActionId(userId)
    setError('')
    try {
      await users.like(userId)
      setItems((prev) => prev.filter((u) => u.id !== userId))
    } catch (err) {
      setError(err.message || 'Failed to like')
    } finally {
      setActionId(null)
    }
  }

  const handleBlock = async (userId) => {
    setActionId(userId)
    setError('')
    try {
      await users.block(userId)
      setItems((prev) => prev.filter((u) => u.id !== userId))
    } catch (err) {
      setError(err.message || 'Failed to block')
    } finally {
      setActionId(null)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-slate-800 mb-6">Likes</h1>

      <div className="flex gap-2 mb-6 border-b border-slate-200">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition ${
              activeTab === tab.id
                ? 'bg-rose-50 text-rose-600 border-b-2 border-rose-500 -mb-px'
                : 'text-slate-600 hover:text-rose-600 hover:bg-slate-50'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {error && <p className="text-rose-600 mb-4">{error}</p>}

      {items.length === 0 ? (
        <p className="text-slate-500">
          {isByMe ? 'You have not liked anyone yet.' : 'No one has liked you yet.'}
        </p>
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
                </div>
              </div>
              <div className="flex gap-2 flex-wrap">
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
                {isByMe ? (
                  <button
                    onClick={() => handleUnlike(u.id)}
                    disabled={!!actionId}
                    className="px-3 py-2 rounded border border-rose-300 text-sm text-rose-700 hover:bg-rose-50 disabled:opacity-60"
                  >
                    {actionId === u.id ? 'Removing...' : 'Unlike'}
                  </button>
                ) : (
                  <button
                    onClick={() => handleLike(u.id)}
                    disabled={!!actionId}
                    className="px-3 py-2 rounded border border-rose-300 text-sm text-rose-700 hover:bg-rose-50 disabled:opacity-60"
                  >
                    {actionId === u.id ? 'Liking...' : 'Like back'}
                  </button>
                )}
                <button
                  onClick={() => handleBlock(u.id)}
                  disabled={!!actionId}
                  className="px-3 py-2 rounded border border-slate-300 text-sm text-slate-600 hover:bg-slate-50 disabled:opacity-60"
                >
                  {actionId === u.id ? 'Blocking...' : 'Block'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
