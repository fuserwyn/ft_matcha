import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { likes, users } from '../api/client'

const TABS = [
  { id: 'by-me', label: 'I liked', fetch: likes.listByMe },
  { id: 'liked-me', label: 'Liked me', fetch: likes.listLikedMe },
]
const PAGE_SIZE = 24

export default function Likes() {
  const [activeTab, setActiveTab] = useState('by-me')
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [offset, setOffset] = useState(0)
  const [error, setError] = useState('')
  const [actionId, setActionId] = useState(null)

  const currentTab = TABS.find((t) => t.id === activeTab)
  const isByMe = activeTab === 'by-me'

  const load = async ({ append = false, currentOffset = 0 } = {}) => {
    if (append) setLoadingMore(true)
    else setLoading(true)
    setError('')
    try {
      const data = await currentTab.fetch({ limit: PAGE_SIZE, offset: currentOffset })
      setItems((prev) => (append ? [...prev, ...data] : data))
      setOffset(currentOffset + data.length)
      setHasMore(data.length === PAGE_SIZE)
    } catch (err) {
      setError(err.message || 'Failed to load likes')
      if (!append) setItems([])
      setHasMore(false)
    } finally {
      if (append) setLoadingMore(false)
      else setLoading(false)
    }
  }

  useEffect(() => {
    setOffset(0)
    setHasMore(true)
    load({ append: false, currentOffset: 0 })
  }, [activeTab])

  const loadMore = () => {
    if (loadingMore || loading || !hasMore) return
    load({ append: true, currentOffset: offset })
  }

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
        <>
          <div className="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3 sm:gap-4">
            {items.map((u) => (
              <div key={u.id} className="group relative rounded-2xl overflow-hidden aspect-[3/4] bg-slate-100 hover:shadow-xl transition-shadow">
                {u.primary_photo_url ? (
                  <img
                    src={u.primary_photo_url}
                    alt={`${u.first_name} ${u.last_name}`}
                    className="absolute inset-0 w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
                    referrerPolicy="no-referrer"
                  />
                ) : (
                  <div className="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-slate-100 to-slate-200">
                    <span className="text-7xl font-bold text-slate-300">{(u.first_name?.[0] || u.username?.[0] || '?').toUpperCase()}</span>
                  </div>
                )}
                <div className="absolute inset-0 bg-gradient-to-t from-black/75 via-black/10 to-transparent" />
                <div className="absolute bottom-0 inset-x-0 p-4 text-white">
                  <p className="font-bold text-lg leading-tight truncate drop-shadow">{u.first_name} {u.last_name}</p>
                  <p className="text-xs text-white/70 mb-3">@{u.username}</p>
                  <div className="flex flex-wrap gap-2">
                    <Link to={`/users/${u.id}`} className="px-3 py-1.5 rounded-full bg-white/20 backdrop-blur-sm text-xs text-white hover:bg-white/30 transition">
                      Profile
                    </Link>
                    <Link to={`/chat/${u.id}`} className="px-3 py-1.5 rounded-full bg-rose-500 text-xs text-white hover:bg-rose-600 transition">
                      💬 Chat
                    </Link>
                    {isByMe ? (
                      <button onClick={() => handleUnlike(u.id)} disabled={!!actionId}
                        className="px-3 py-1.5 rounded-full bg-white/20 backdrop-blur-sm text-xs text-white hover:bg-white/30 transition disabled:opacity-60">
                        {actionId === u.id ? '...' : 'Unlike'}
                      </button>
                    ) : (
                      <button onClick={() => handleLike(u.id)} disabled={!!actionId}
                        className="px-3 py-1.5 rounded-full bg-emerald-500 text-xs text-white hover:bg-emerald-600 transition disabled:opacity-60">
                        {actionId === u.id ? '...' : '♡ Like back'}
                      </button>
                    )}
                    <button onClick={() => handleBlock(u.id)} disabled={!!actionId}
                      className="px-3 py-1.5 rounded-full bg-white/20 backdrop-blur-sm text-xs text-white hover:bg-white/30 transition disabled:opacity-60">
                      {actionId === u.id ? '...' : 'Block'}
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
          {hasMore && (
            <div className="mt-5 flex justify-center">
              <button
                type="button"
                onClick={loadMore}
                disabled={loadingMore}
                className="px-4 py-2 rounded-lg border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-60"
              >
                {loadingMore ? 'Loading...' : 'Load more'}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
