import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { matches, users } from '../api/client'

export default function Matches() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
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
                  className="px-3 py-2 rounded border border-rose-300 text-sm text-rose-700 hover:bg-rose-50 disabled:opacity-60"
                >
                  {blockingId === u.id ? 'Blocking...' : 'Block'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
