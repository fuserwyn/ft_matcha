import { useState, useEffect } from 'react'
import { auth, photos, profile } from '../api/client'
import { useAuth } from '../context/AuthContext'

const GENDERS = ['male', 'female', 'non-binary', 'other']
const PREFERENCES = ['male', 'female', 'both', 'other']

export default function Profile() {
  const { updateUser } = useAuth()
  const [account, setAccount] = useState({
    username: '',
    email: '',
    first_name: '',
    last_name: '',
  })
  const [data, setData] = useState({
    bio: '',
    gender: '',
    sexual_preference: '',
    birth_date: '',
    city: '',
    latitude: '',
    longitude: '',
  })
  const [tagsInput, setTagsInput] = useState('')
  const [tagSuggestions, setTagSuggestions] = useState([])
  const [loading, setLoading] = useState(true)
  const [savingAccount, setSavingAccount] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savingTags, setSavingTags] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [photoList, setPhotoList] = useState([])
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([auth.me(), profile.get(), profile.getTags(), profile.tagSuggestions()])
      .then(([me, p, tagsRes, suggRes]) => {
        setAccount({
          username: me.username || '',
          email: me.email || '',
          first_name: me.first_name || '',
          last_name: me.last_name || '',
        })
        setData({
          bio: p.bio || '',
          gender: p.gender || '',
          sexual_preference: p.sexual_preference || '',
          birth_date: p.birth_date || '',
          city: p.city || '',
          latitude: p.latitude ?? '',
          longitude: p.longitude ?? '',
        })
        setTagsInput((tagsRes?.tags || []).join(', '))
        setTagSuggestions(suggRes?.tags || [])
        setPhotoList(p.photos || [])
      })
      .catch(() => setError('Failed to load profile'))
      .finally(() => setLoading(false))
  }, [])

  const handleChange = (e) => {
    const { name, value } = e.target
    setData((d) => ({ ...d, [name]: value }))
  }

  const handleAccountChange = (e) => {
    const { name, value } = e.target
    setAccount((d) => ({ ...d, [name]: value }))
  }

  const handleAccountSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setMessage('')
    setSavingAccount(true)
    try {
      await auth.updateMe(account)
      updateUser({ username: account.username })
      setMessage('Account updated')
    } catch (err) {
      setError(err.message || 'Account update failed')
    } finally {
      setSavingAccount(false)
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setMessage('')
    if (data.birth_date) {
      const d = new Date(data.birth_date)
      if (d > new Date()) {
        setError('Birth date must be in the past')
        return
      }
      const age = Math.floor((new Date() - d) / (365.25 * 24 * 60 * 60 * 1000))
      if (age < 18) {
        setError('You must be at least 18 years old')
        return
      }
    }
    setSaving(true)
    const payload = {}
    if (data.bio) payload.bio = data.bio
    if (data.gender) payload.gender = data.gender
    if (data.sexual_preference) payload.sexual_preference = data.sexual_preference
    if (data.birth_date) payload.birth_date = data.birth_date
    if (data.city) payload.city = data.city
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

  const handleSaveTags = async () => {
    setError('')
    setMessage('')
    setSavingTags(true)
    try {
      const tags = tagsInput
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean)
      await profile.updateTags(tags)
      setMessage('Tags updated')
    } catch (err) {
      setError(err.message || 'Tags update failed')
    } finally {
      setSavingTags(false)
    }
  }

  const handleUseMyLocation = () => {
    if (!navigator.geolocation) {
      setError('Geolocation is not supported by your browser')
      return
    }
    setError('')
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setData((d) => ({
          ...d,
          latitude: String(pos.coords.latitude),
          longitude: String(pos.coords.longitude),
        }))
        setMessage('Coordinates updated from GPS')
      },
      () => {
        setError('Unable to retrieve your location')
      },
      { enableHighAccuracy: true, timeout: 10000, maximumAge: 30000 },
    )
  }

  const refreshPhotos = async () => {
    const list = await photos.listMe()
    setPhotoList(list)
  }

  const handleUpload = async (e) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setError('')
    setMessage('')
    setUploading(true)
    try {
      await photos.upload(file)
      await refreshPhotos()
      setMessage('Photo uploaded')
    } catch (err) {
      setError(err.message || 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  const handleDeletePhoto = async (id) => {
    setError('')
    setMessage('')
    try {
      await photos.remove(id)
      await refreshPhotos()
      setMessage('Photo deleted')
    } catch (err) {
      setError(err.message || 'Delete failed')
    }
  }

  const handleSetPrimary = async (id) => {
    setError('')
    setMessage('')
    try {
      await photos.setPrimary(id)
      await refreshPhotos()
      setMessage('Primary photo updated')
    } catch (err) {
      setError(err.message || 'Update failed')
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
      <form onSubmit={handleAccountSubmit} className="space-y-4 mb-6 p-4 bg-white rounded-lg border border-slate-200">
        <p className="text-sm font-medium text-slate-700">Account</p>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Username</label>
            <input
              type="text"
              name="username"
              value={account.username}
              onChange={handleAccountChange}
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Email</label>
            <input
              type="email"
              name="email"
              value={account.email}
              onChange={handleAccountChange}
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">First name</label>
            <input
              type="text"
              name="first_name"
              value={account.first_name}
              onChange={handleAccountChange}
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Last name</label>
            <input
              type="text"
              name="last_name"
              value={account.last_name}
              onChange={handleAccountChange}
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
            />
          </div>
        </div>
        <button
          type="submit"
          disabled={savingAccount}
          className="py-2 px-4 bg-slate-800 text-white font-medium rounded-lg hover:bg-slate-900 disabled:opacity-50 transition"
        >
          {savingAccount ? 'Saving account...' : 'Save account'}
        </button>
      </form>
      <div className="space-y-3 mb-6 p-4 bg-white rounded-lg border border-slate-200">
        <p className="text-sm font-medium text-slate-700">Interests / Tags</p>
        <input
          type="text"
          value={tagsInput}
          onChange={(e) => setTagsInput(e.target.value)}
          placeholder="music, travel, books"
          className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
        />
        {tagSuggestions.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {tagSuggestions.map((tag) => (
              <button
                key={tag}
                type="button"
                onClick={() => {
                  const existing = tagsInput
                    .split(',')
                    .map((x) => x.trim().toLowerCase())
                    .filter(Boolean)
                  if (existing.includes(tag.toLowerCase())) return
                  setTagsInput((prev) => (prev.trim() ? `${prev}, ${tag}` : tag))
                }}
                className="text-xs px-2 py-1 bg-slate-100 rounded hover:bg-slate-200 text-slate-700"
              >
                #{tag}
              </button>
            ))}
          </div>
        )}
        <div>
          <button
            type="button"
            onClick={handleSaveTags}
            disabled={savingTags}
            className="py-2 px-4 bg-rose-500 text-white font-medium rounded-lg hover:bg-rose-600 disabled:opacity-50 transition"
          >
            {savingTags ? 'Saving tags...' : 'Save tags'}
          </button>
        </div>
      </div>
      <div className="mb-6 p-4 bg-white rounded-lg border border-slate-200">
        <div className="flex items-center justify-between mb-3">
          <p className="text-sm font-medium text-slate-700">Photos ({photoList.length}/5)</p>
          <label className="px-3 py-1.5 text-sm bg-rose-500 text-white rounded cursor-pointer hover:bg-rose-600">
            {uploading ? 'Uploading...' : 'Upload'}
            <input
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleUpload}
              disabled={uploading || photoList.length >= 5}
            />
          </label>
        </div>
        {photoList.length === 0 ? (
          <p className="text-xs text-slate-500">Add at least one photo to improve your profile.</p>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {photoList.map((p) => (
              <div key={p.id} className="border border-slate-200 rounded-lg p-2">
                <img src={p.url} alt="User upload" className="w-full h-28 object-cover rounded" />
                <div className="mt-2 flex gap-2">
                  {!p.is_primary && (
                    <button
                      type="button"
                      onClick={() => handleSetPrimary(p.id)}
                      className="text-xs px-2 py-1 rounded border border-slate-300 hover:bg-slate-50"
                    >
                      Set primary
                    </button>
                  )}
                  {p.is_primary && (
                    <span className="text-xs px-2 py-1 rounded bg-emerald-50 text-emerald-700">
                      Primary
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => handleDeletePhoto(p.id)}
                    className="text-xs px-2 py-1 rounded border border-rose-200 text-rose-700 hover:bg-rose-50"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
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
            max={new Date().toISOString().split('T')[0]}
            className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
          />
          <p className="text-xs text-slate-500 mt-1">YYYY-MM-DD, 18+</p>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="col-span-2">
            <label className="block text-sm font-medium text-slate-700 mb-1">City</label>
            <input
              type="text"
              name="city"
              value={data.city}
              onChange={handleChange}
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
              placeholder="Paris"
            />
          </div>
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
          <div className="col-span-2 flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={handleUseMyLocation}
              className="px-3 py-2 rounded border border-slate-300 text-sm text-slate-700 hover:bg-slate-50"
            >
              Use my current GPS location
            </button>
            {data.latitude && data.longitude && !isNaN(parseFloat(data.latitude)) && !isNaN(parseFloat(data.longitude)) && (
              <span className="text-xs text-emerald-600 bg-emerald-50 px-2 py-1 rounded">
                ✓ Location set ({parseFloat(data.latitude).toFixed(2)}, {parseFloat(data.longitude).toFixed(2)})
              </span>
            )}
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
