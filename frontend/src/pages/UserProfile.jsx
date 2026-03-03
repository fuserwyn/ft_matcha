import { useState, useEffect } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { presence, users, photos } from '../api/client'

export default function UserProfile() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [isMatch, setIsMatch] = useState(false)
  const [likedMe, setLikedMe] = useState(false)
  const [iLiked, setILiked] = useState(false)
  const [presenceState, setPresenceState] = useState(null)
  const [liking, setLiking] = useState(false)
  const [blocking, setBlocking] = useState(false)
  const [blocked, setBlocked] = useState(false)
  const [hasPrimaryPhoto, setHasPrimaryPhoto] = useState(false)

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const [u, p, myPhotos] = await Promise.all([
          users.getById(id),
          presence.get(id),
          photos.listMe().catch(() => []),
        ])
        if (!active) return
        setUser(u)
        setIsMatch(Boolean(u.is_match))
        setLikedMe(Boolean(u.liked_me))
        setILiked(Boolean(u.i_liked))
        setPresenceState(p)
        setHasPrimaryPhoto(Array.isArray(myPhotos) && myPhotos.some((ph) => ph.is_primary))
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
    if (blocked || !hasPrimaryPhoto) return
    setLiking(true)
    setInfo('')
    setError('')
    try {
      const res = await users.like(id)
      if (res?.is_match) {
        setIsMatch(true)
        setILiked(true)
        setInfo("It's a match! You can open chat now.")
      } else {
        setILiked(true)
        setInfo('Liked')
      }
    } catch (err) {
      setError(err.message || 'Failed to like user')
    } finally {
      setLiking(false)
    }
  }

  const onUnlike = async () => {
    if (blocked) return
    setLiking(true)
    setInfo('')
    setError('')
    try {
      await users.unlike(id)
      setILiked(false)
      setIsMatch(false)
      setInfo('Like removed')
    } catch (err) {
      setError(err.message || 'Failed to unlike user')
    } finally {
      setLiking(false)
    }
  }

  const onToggleBlock = async () => {
    setBlocking(true)
    setInfo('')
    setError('')
    try {
      if (blocked) {
        await users.unblock(id)
        setBlocked(false)
        setInfo('User unblocked')
      } else {
        await users.block(id)
        setBlocked(true)
        setInfo('User blocked')
      }
    } catch (err) {
      setError(err.message || 'Failed to update block state')
    } finally {
      setBlocking(false)
    }
  }

  return (
    <div className="max-w-lg w-full min-w-0">
      <button
        onClick={() => navigate(-1)}
        className="text-slate-600 hover:text-rose-600 mb-4 text-sm"
      >
        ← Back
      </button>
      <div className="bg-white rounded-2xl shadow-lg p-4 sm:p-8 border border-slate-100">
        {user.photos?.length > 0 && (
          <div className="grid grid-cols-2 gap-3 mb-5">
            {user.photos.map((p) => (
              <img
                key={p.id}
                src={p.url}
                alt={`${user.first_name} ${user.last_name}`}
                className={`w-full h-36 object-cover rounded ${p.is_primary ? 'ring-2 ring-rose-400' : ''}`}
                referrerPolicy="no-referrer"
              />
            ))}
          </div>
        )}
        <h1 className="text-2xl font-bold text-slate-800">
          {user.first_name} {user.last_name}
        </h1>
        <p className="text-slate-500">@{user.username}</p>
        {presenceState && (
          <p className="text-sm mt-1">
            {presenceState.is_online ? (
              <span className="text-emerald-600 font-medium">● Online now</span>
            ) : (
              <span className="text-slate-500">
                Last connection: {presenceState.last_seen ? new Date(presenceState.last_seen).toLocaleString() : 'unknown'}
              </span>
            )}
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
        <div className="mt-3 flex flex-wrap gap-2 text-xs">
          {isMatch && <span className="px-2 py-1 rounded bg-emerald-50 text-emerald-700">Connected (match)</span>}
          {likedMe && <span className="px-2 py-1 rounded bg-blue-50 text-blue-700">Liked you</span>}
          {iLiked && <span className="px-2 py-1 rounded bg-rose-50 text-rose-700">You liked</span>}
        </div>
        {user.bio && (
          <p className="mt-4 text-slate-600">{user.bio}</p>
        )}
        <div className="mt-4 text-rose-500">★ Fame rating: {user.fame_rating ?? 0}</div>
        {user.city && (
          <p className="mt-2 text-slate-600">City: {user.city}</p>
        )}
        {user.latitude != null && user.longitude != null && (
          <p className="mt-2 text-xs text-slate-500">
            Location: {user.latitude.toFixed(2)}, {user.longitude.toFixed(2)}
          </p>
        )}
        {Array.isArray(user.tags) && user.tags.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2">
            {user.tags.map((tag) => (
              <span key={tag} className="text-xs px-2 py-1 bg-slate-100 rounded text-slate-600">
                #{tag}
              </span>
            ))}
          </div>
        )}
        {!hasPrimaryPhoto && (
          <p className="mt-4 text-amber-600 text-sm">
            Add a profile picture to like other users.
          </p>
        )}
        <div className="mt-6 flex gap-3 flex-wrap">
          {iLiked ? (
            <button
              onClick={onUnlike}
              disabled={liking || blocked}
              className="px-4 py-2 border border-slate-300 text-slate-700 rounded hover:bg-slate-50 disabled:opacity-60"
            >
              {liking ? 'Unlike...' : 'Unlike'}
            </button>
          ) : (
            <button
              onClick={onLike}
              disabled={liking || blocked || !hasPrimaryPhoto}
              className="px-4 py-2 bg-rose-500 text-white rounded hover:bg-rose-600 disabled:opacity-60"
            >
              {liking ? 'Liking...' : 'Like'}
            </button>
          )}
          {isMatch && !blocked && (
            <Link
              to={`/chat/${id}`}
              className="px-4 py-2 border border-slate-300 text-slate-700 rounded hover:bg-slate-50"
            >
              Open chat
            </Link>
          )}
          <button
            onClick={onToggleBlock}
            disabled={blocking}
            className={`px-4 py-2 rounded border ${
              blocked
                ? 'border-emerald-300 text-emerald-700 hover:bg-emerald-50'
                : 'border-rose-300 text-rose-700 hover:bg-rose-50'
            } disabled:opacity-60`}
          >
            {blocking ? 'Updating...' : blocked ? 'Unblock' : 'Block'}
          </button>
        </div>
        {info && <p className="mt-3 text-emerald-600 text-sm">{info}</p>}
        {error && <p className="mt-3 text-rose-600 text-sm">{error}</p>}
      </div>
    </div>
  )
}
