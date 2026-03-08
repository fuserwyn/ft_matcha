import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { matches, users } from '../api/client'

const PAGE_SIZE = 24

export default function Matches() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [nextCursor, setNextCursor] = useState('')
  const [error, setError] = useState('')
  const [blockingId, setBlockingId] = useState(null)

  const [flashMessage, setFlashMessage] = useState(null)

  useEffect(() => {
    const verified = searchParams.get('verified')
    const already = searchParams.get('already')
    const errParam = searchParams.get('error')
    if (verified) {
      setFlashMessage('verified')
      setSearchParams({}, { replace: true })
    } else if (already) {
      setFlashMessage('already')
      setSearchParams({}, { replace: true })
    } else if (errParam) {
      setFlashMessage('error')
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, setSearchParams])

  const load = async ({ append = false, cursor = '' } = {}) => {
    if (append) setLoadingMore(true)
    else setLoading(true)
    setError('')
    try {
      const params = { limit: PAGE_SIZE }
      if (cursor) params.cursor = cursor
      const data = await matches.list(params)
      const pageItems = Array.isArray(data) ? data : (data.items || [])
      const pageHasMore = Array.isArray(data) ? pageItems.length === PAGE_SIZE : !!data.has_more
      const pageNextCursor = Array.isArray(data) ? '' : (data.next_cursor || '')
      setItems((prev) => (append ? [...prev, ...pageItems] : pageItems))
      setNextCursor(pageNextCursor)
      setHasMore(pageHasMore)
    } catch (err) {
      setError(err.message || 'Failed to load matches')
      if (!append) setItems([])
      setNextCursor('')
      setHasMore(false)
    } finally {
      if (append) setLoadingMore(false)
      else setLoading(false)
    }
  }

  useEffect(() => {
    load({ append: false, cursor: '' })
  }, [])

  const loadMore = () => {
    if (loadingMore || loading || !hasMore) return
    load({ append: true, cursor: nextCursor })
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
      <h1 className="text-2xl font-bold text-slate-800 mb-6">Matches</h1>
      {flashMessage === 'verified' && (
        <div className="mb-4 p-4 rounded-lg bg-emerald-50 text-emerald-800 border border-emerald-200">
          Email verified successfully.
        </div>
      )}
      {flashMessage === 'already' && (
        <div className="mb-4 p-4 rounded-lg bg-slate-100 text-slate-700 border border-slate-200">
          Your email was already verified.
        </div>
      )}
      {flashMessage === 'error' && (
        <div className="mb-4 p-4 rounded-lg bg-amber-50 text-amber-800 border border-amber-200">
          Verification link is invalid or expired.
        </div>
      )}
      {error && <p className="text-rose-600 mb-4">{error}</p>}
      {items.length === 0 ? (
        <p className="text-slate-500">No matches yet.</p>
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
                    <button
                      onClick={async () => {
                        setBlockingId(u.id)
                        setError('')
                        try {
                          await users.block(u.id)
                          setItems((prev) => prev.filter((x) => x.id !== u.id))
                        } catch (err) {
                          setError(err.message || 'Failed to block user')
                        } finally {
                          setBlockingId(null)
                        }
                      }}
                      disabled={blockingId === u.id}
                      className="px-3 py-1.5 rounded-full bg-white/20 backdrop-blur-sm text-xs text-white hover:bg-white/30 transition disabled:opacity-60"
                    >
                      {blockingId === u.id ? '...' : 'Block'}
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
