import { useState, useEffect } from 'react'
import { profile } from '../api/client'

const GENDERS = ['male', 'female', 'non-binary', 'other']
const PREFERENCES = ['male', 'female', 'both', 'other']

export default function Profile() {
  const [data, setData] = useState({
    bio: '',
    gender: '',
    sexual_preference: '',
    birth_date: '',
    latitude: '',
    longitude: '',
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    profile
      .get()
      .then((p) => {
        setData({
          bio: p.bio || '',
          gender: p.gender || '',
          sexual_preference: p.sexual_preference || '',
          birth_date: p.birth_date || '',
          latitude: p.latitude ?? '',
          longitude: p.longitude ?? '',
        })
      })
      .catch(() => setError('Failed to load profile'))
      .finally(() => setLoading(false))
  }, [])

  const handleChange = (e) => {
    const { name, value } = e.target
    setData((d) => ({ ...d, [name]: value }))
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setMessage('')
    setSaving(true)
    const payload = {}
    if (data.bio) payload.bio = data.bio
    if (data.gender) payload.gender = data.gender
    if (data.sexual_preference) payload.sexual_preference = data.sexual_preference
    if (data.birth_date) payload.birth_date = data.birth_date
    if (data.latitude !== '') payload.latitude = parseFloat(data.latitude)
    if (data.longitude !== '') payload.longitude = parseFloat(data.longitude)
    try {
      await profile.update(payload)
      setMessage('Profile updated')
    } catch (err) {
      setError(err.message || 'Update failed')
    } finally {
      setSaving(false)
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
    <div className="max-w-lg">
      <h1 className="text-2xl font-bold text-slate-800 mb-6">Your profile</h1>
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="bg-rose-50 text-rose-700 px-4 py-3 rounded-lg text-sm">{error}</div>
        )}
        {message && (
          <div className="bg-emerald-50 text-emerald-700 px-4 py-3 rounded-lg text-sm">
            {message}
          </div>
        )}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">Bio</label>
          <textarea
            name="bio"
            value={data.bio}
            onChange={handleChange}
            rows={3}
            maxLength={500}
            className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
            placeholder="Tell us about yourself..."
          />
          <p className="text-xs text-slate-500 mt-1">{data.bio.length}/500</p>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">Gender</label>
          <select
            name="gender"
            value={data.gender}
            onChange={handleChange}
            className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
          >
            <option value="">Select</option>
            {GENDERS.map((g) => (
              <option key={g} value={g}>
                {g}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">Interested in</label>
          <select
            name="sexual_preference"
            value={data.sexual_preference}
            onChange={handleChange}
            className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
          >
            <option value="">Select</option>
            {PREFERENCES.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">Birth date</label>
          <input
            type="date"
            name="birth_date"
            value={data.birth_date}
            onChange={handleChange}
            className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
          />
          <p className="text-xs text-slate-500 mt-1">YYYY-MM-DD, 18+</p>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Latitude</label>
            <input
              type="number"
              name="latitude"
              value={data.latitude}
              onChange={handleChange}
              step="any"
              min="-90"
              max="90"
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
              placeholder="-90 to 90"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Longitude</label>
            <input
              type="number"
              name="longitude"
              value={data.longitude}
              onChange={handleChange}
              step="any"
              min="-180"
              max="180"
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
              placeholder="-180 to 180"
            />
          </div>
        </div>
        <button
          type="submit"
          disabled={saving}
          className="w-full py-3 bg-rose-500 text-white font-medium rounded-lg hover:bg-rose-600 disabled:opacity-50 transition"
        >
          {saving ? 'Saving...' : 'Save profile'}
        </button>
      </form>
    </div>
  )
}
