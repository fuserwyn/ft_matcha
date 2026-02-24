import { useState, useEffect } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { matches, presence, users } from '../api/client'

export default function UserProfile() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [isMatch, setIsMatch] = useState(false)
  const [presenceState, setPresenceState] = useState(null)
  const [liking, setLiking] = useState(false)

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const [u, m, p] = await Promise.all([
          users.getById(id),
          matches.list(),
          presence.get(id),
        ])
        if (!active) return
        setUser(u)
        setIsMatch(m.some((x) => x.id === id))
        setPresenceState(p)
      } catch {
        if (active) setError('User not found')
      } finally {
        if (active) setLoading(false)
      }
    })()
    return () => {
      active = false
    }
  }, [id])

  const age = (birthDate) => {
    if (!birthDate) return null
    const diff = Date.now() - new Date(birthDate).getTime()
    return Math.floor(diff / (365.25 * 24 * 60 * 60 * 1000))
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
      </div>
    )
  }

  if (error || !user) {
    return (
      <div className="text-center py-12">
        <p className="text-rose-600 mb-4">{error || 'User not found'}</p>
        <button
          onClick={() => navigate('/discover')}
          className="text-rose-600 hover:underline"
        >
          Back to Discover
        </button>
      </div>
    )
  }

  const onLike = async () => {
    setLiking(true)
    setInfo('')
    setError('')
    try {
      const res = await users.like(id)
      if (res?.is_match) {
        setIsMatch(true)
        setInfo("It's a match! You can open chat now.")
      } else {
        setInfo('Liked')
      }
    } catch (err) {
      setError(err.message || 'Failed to like user')
    } finally {
      setLiking(false)
    }
  }

  return (
    <div className="max-w-lg">
      <button
        onClick={() => navigate(-1)}
        className="text-slate-600 hover:text-rose-600 mb-4 text-sm"
      >
        ← Back
      </button>
      <div className="bg-white rounded-2xl shadow-lg p-8 border border-slate-100">
        {user.photos?.length > 0 && (
          <div className="grid grid-cols-2 gap-3 mb-5">
            {user.photos.map((p) => (
              <img
                key={p.id}
                src={p.url}
                alt={`${user.first_name} ${user.last_name}`}
                className={`w-full h-36 object-cover rounded ${p.is_primary ? 'ring-2 ring-rose-400' : ''}`}
              />
            ))}
          </div>
        )}
        <h1 className="text-2xl font-bold text-slate-800">
          {user.first_name} {user.last_name}
        </h1>
        <p className="text-slate-500">@{user.username}</p>
        {presenceState && (
          <p className="text-xs text-slate-500 mt-1">
            {presenceState.is_online
              ? 'Online now'
              : `Last seen: ${
                  presenceState.last_seen ? new Date(presenceState.last_seen).toLocaleString() : 'unknown'
                }`}
          </p>
        )}
        <div className="mt-4 flex gap-2 text-sm">
          {user.gender && (
            <span className="px-2 py-1 bg-slate-100 rounded">{user.gender}</span>
          )}
          {user.birth_date && (
            <span className="px-2 py-1 bg-slate-100 rounded">
              {age(user.birth_date)} years old
            </span>
          )}
          {user.sexual_preference && (
            <span className="px-2 py-1 bg-slate-100 rounded">
              Interested in {user.sexual_preference}
            </span>
          )}
        </div>
        {user.bio && (
          <p className="mt-4 text-slate-600">{user.bio}</p>
        )}
        {user.fame_rating > 0 && (
          <div className="mt-4 text-rose-500">★ Fame: {user.fame_rating}</div>
        )}
        {user.latitude != null && user.longitude != null && (
          <p className="mt-4 text-xs text-slate-500">
            Location: {user.latitude.toFixed(2)}, {user.longitude.toFixed(2)}
          </p>
        )}
        <div className="mt-6 flex gap-3">
          <button
            onClick={onLike}
            disabled={liking}
            className="px-4 py-2 bg-rose-500 text-white rounded hover:bg-rose-600 disabled:opacity-60"
          >
            {liking ? 'Liking...' : 'Like'}
          </button>
          {isMatch && (
            <Link
              to={`/chat/${id}`}
              className="px-4 py-2 border border-slate-300 text-slate-700 rounded hover:bg-slate-50"
            >
              Open chat
            </Link>
          )}
        </div>
        {info && <p className="mt-3 text-emerald-600 text-sm">{info}</p>}
        {error && <p className="mt-3 text-rose-600 text-sm">{error}</p>}
      </div>
    </div>
  )
}
