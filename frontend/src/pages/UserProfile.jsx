import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { users } from '../api/client'

export default function UserProfile() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    users
      .getById(id)
      .then(setUser)
      .catch(() => setError('User not found'))
      .finally(() => setLoading(false))
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

  return (
    <div className="max-w-lg">
      <button
        onClick={() => navigate(-1)}
        className="text-slate-600 hover:text-rose-600 mb-4 text-sm"
      >
        ← Back
      </button>
      <div className="bg-white rounded-2xl shadow-lg p-8 border border-slate-100">
        <h1 className="text-2xl font-bold text-slate-800">
          {user.first_name} {user.last_name}
        </h1>
        <p className="text-slate-500">@{user.username}</p>
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
      </div>
    </div>
  )
}
