import { useState, useEffect, useRef } from 'react'
import { auth, photos, profile } from '../api/client'
import { useAuth } from '../context/AuthContext'
import CityInput from '../components/CityInput'
import PhotoCropper from '../components/PhotoCropper'

const GENDERS = ['male', 'female', 'non-binary', 'other']
const PREFERENCES = ['male', 'female', 'non-binary', 'other']
const RELATIONSHIP_GOALS = [
  { value: 'long-term', label: 'Long-term' },
  { value: 'long-term-open', label: 'Long-term, open to short' },
  { value: 'short-term-open', label: 'Short-term, open to long' },
  { value: 'short-term', label: 'Short-term' },
  { value: 'friends', label: 'Friends' },
  { value: 'not-sure', label: 'Not sure yet' },
]

export default function Profile() {
  const { updateUser } = useAuth()
  const cityInputRef = useRef(null)
  const [account, setAccount] = useState({
    username: '',
    email: '',
    first_name: '',
    last_name: '',
  })
  const [data, setData] = useState({
    bio: '',
    gender: '',
    sexual_preference: [],
    relationship_goal: '',
    birth_date: '',
    city: '',
    latitude: '',
    longitude: '',
  })
  const [tagsInput, setTagsInput] = useState('')
  const [tagSuggestions, setTagSuggestions] = useState([])
  const [loading, setLoading] = useState(true)
  const [savingAccount, setSavingAccount] = useState(false)
  const [savedAccount, setSavedAccount] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)
  const [savingTags, setSavingTags] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [photoList, setPhotoList] = useState([])
  const [cropSrc, setCropSrc] = useState(null)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [tagsMessage, setTagsMessage] = useState('')
  const [tagsError, setTagsError] = useState('')
  const [passwordForm, setPasswordForm] = useState({
    current_password: '',
    new_password: '',
    confirm_password: '',
  })

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
          sexual_preference: Array.isArray(p.sexual_preference) ? p.sexual_preference : [],
          relationship_goal: p.relationship_goal || '',
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
      setSavedAccount(true)
      setTimeout(() => setSavedAccount(false), 2000)
    } catch (err) {
      setError(err.message || 'Account update failed')
    } finally {
      setSavingAccount(false)
    }
  }

  const handlePasswordInputChange = (e) => {
    const { name, value } = e.target
    setPasswordForm((prev) => ({ ...prev, [name]: value }))
  }

  const handlePasswordSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setMessage('')
    if (!passwordForm.current_password || !passwordForm.new_password || !passwordForm.confirm_password) {
      setError('Please fill all password fields')
      return
    }
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      setError('New password and confirmation do not match')
      return
    }
    setSavingPassword(true)
    try {
      await auth.changePassword(passwordForm)
      setPasswordForm({ current_password: '', new_password: '', confirm_password: '' })
      setMessage('Password updated')
    } catch (err) {
      setError(err.message || 'Password update failed')
    } finally {
      setSavingPassword(false)
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
    const effectiveSexualPreference =
      data.sexual_preference.length > 0 ? data.sexual_preference : ['male', 'female']
    const payload = {}
    if (data.bio) payload.bio = data.bio
    if (data.gender) payload.gender = data.gender
    payload.sexual_preference = effectiveSexualPreference
    if (data.relationship_goal) payload.relationship_goal = data.relationship_goal
    if (data.birth_date) payload.birth_date = data.birth_date
    if (data.city) payload.city = data.city
    if (data.latitude !== '') payload.latitude = parseFloat(data.latitude)
    if (data.longitude !== '') payload.longitude = parseFloat(data.longitude)
    try {
      await profile.update(payload)
      setData((d) => ({ ...d, sexual_preference: effectiveSexualPreference }))
      setMessage('Profile updated')
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      setError(err.message || 'Update failed')
    } finally {
      setSaving(false)
    }
  }

  const handleSaveTags = async () => {
    setTagsError('')
    setTagsMessage('')
    setSavingTags(true)
    try {
      const tags = tagsInput
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean)
      await profile.updateTags(tags)
      setTagsMessage('Tags updated')
      setTimeout(() => setTagsMessage(''), 3000)
    } catch (err) {
      setTagsError(err.message || 'Tags update failed')
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
    setMessage('Getting your location...')
    navigator.geolocation.getCurrentPosition(
      async (pos) => {
        const { latitude, longitude } = pos.coords
        setData((d) => ({ ...d, latitude: String(latitude), longitude: String(longitude) }))
        try {
          const res = await fetch(
            `https://nominatim.openstreetmap.org/reverse?lat=${latitude}&lon=${longitude}&format=json`,
            { headers: { 'Accept-Language': 'en' } },
          )
          const place = await res.json()
          const city =
            place.address?.city ||
            place.address?.town ||
            place.address?.village ||
            place.address?.county ||
            ''
          if (city) {
            cityInputRef.current?.suppressNext()
            setData((d) => ({ ...d, latitude: String(latitude), longitude: String(longitude), city }))
            setMessage(`Location set to ${city}`)
          } else {
            setMessage('GPS coordinates updated')
          }
        } catch {
          setMessage('GPS coordinates updated (city lookup failed)')
        }
      },
      async (err) => {
        setMessage('')
        if (err.code === 1 && location.protocol !== 'https:') {
          try {
            const res = await fetch('https://ipapi.co/json/')
            const data = await res.json()
            if (data.latitude && data.longitude) {
              const city = data.city || ''
              if (city) cityInputRef.current?.suppressNext()
              setData((d) => ({
                ...d,
                latitude: String(data.latitude),
                longitude: String(data.longitude),
                ...(city ? { city } : {}),
              }))
              setMessage(`Location set${city ? ` to ${city}` : ''} (via IP — GPS requires HTTPS)`)
            } else {
              setError('Location unavailable. GPS requires HTTPS; set your city manually.')
            }
          } catch {
            setError('Location unavailable. GPS requires HTTPS; set your city manually.')
          }
        } else {
          setError('Unable to retrieve your location. Please set your city manually.')
        }
      },
      { enableHighAccuracy: true, timeout: 10000, maximumAge: 30000 },
    )
  }

  const refreshPhotos = async () => {
    const list = await photos.listMe()
    setPhotoList(list)
  }

  const handleUpload = (e) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => setCropSrc(reader.result)
    reader.readAsDataURL(file)
  }

  const handleCropConfirm = async (blob) => {
    setCropSrc(null)
    setError('')
    setMessage('')
    setUploading(true)
    try {
      const file = new File([blob], 'photo.jpg', { type: 'image/jpeg' })
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
    <div>
      {cropSrc && (
        <PhotoCropper
          imageSrc={cropSrc}
          onConfirm={handleCropConfirm}
          onCancel={() => setCropSrc(null)}
        />
      )}

      <h1 className="text-2xl font-bold text-slate-800 mb-6">Your profile</h1>

      {error && <div className="mb-4 bg-rose-50 text-rose-700 px-4 py-3 rounded-lg text-sm">{error}</div>}
      {message && <div className="mb-4 bg-emerald-50 text-emerald-700 px-4 py-3 rounded-lg text-sm">{message}</div>}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="space-y-6">
          <form onSubmit={handleAccountSubmit} className="p-5 bg-white rounded-xl border border-slate-200 space-y-4">
            <p className="text-sm font-semibold text-slate-700">Account</p>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Username</label>
                <input type="text" name="username" value={account.username} onChange={handleAccountChange}
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none" />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Email</label>
                <input type="email" name="email" value={account.email} onChange={handleAccountChange}
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none" />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">First name</label>
                <input type="text" name="first_name" value={account.first_name} onChange={handleAccountChange}
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none" />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Last name</label>
                <input type="text" name="last_name" value={account.last_name} onChange={handleAccountChange}
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none" />
              </div>
            </div>
            <button type="submit" disabled={savingAccount}
              className={`py-2 px-4 text-white font-medium rounded-lg transition disabled:opacity-50 ${savedAccount ? 'bg-emerald-600' : savingAccount ? 'bg-emerald-500 opacity-80' : 'bg-slate-800 hover:bg-slate-900'}`}>
              {savingAccount ? 'Saving...' : savedAccount ? 'Saved!' : 'Save account'}
            </button>
          </form>

          <form onSubmit={handlePasswordSubmit} className="p-5 bg-white rounded-xl border border-slate-200 space-y-4">
            <p className="text-sm font-semibold text-slate-700">Change password</p>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Current password</label>
                <input
                  type="password"
                  name="current_password"
                  value={passwordForm.current_password}
                  onChange={handlePasswordInputChange}
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">New password</label>
                <input
                  type="password"
                  name="new_password"
                  value={passwordForm.new_password}
                  onChange={handlePasswordInputChange}
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Confirm password</label>
                <input
                  type="password"
                  name="confirm_password"
                  value={passwordForm.confirm_password}
                  onChange={handlePasswordInputChange}
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
                />
              </div>
            </div>
            <button
              type="submit"
              disabled={savingPassword}
              className="py-2 px-4 bg-slate-800 text-white font-medium rounded-lg hover:bg-slate-900 disabled:opacity-50 transition"
            >
              {savingPassword ? 'Updating...' : 'Change password'}
            </button>
          </form>

          <div className="p-5 bg-white rounded-xl border border-slate-200 space-y-3">
            <p className="text-sm font-semibold text-slate-700">Interests / Tags</p>
            <input type="text" value={tagsInput} onChange={(e) => setTagsInput(e.target.value)}
              placeholder="music, travel, books"
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none" />
            {tagSuggestions.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {tagSuggestions.map((tag) => (
                  <button key={tag} type="button"
                    onClick={() => {
                      const existing = tagsInput.split(',').map((x) => x.trim().toLowerCase()).filter(Boolean)
                      if (existing.includes(tag.toLowerCase())) return
                      setTagsInput((prev) => (prev.trim() ? `${prev}, ${tag}` : tag))
                    }}
                    className="text-xs px-2 py-1 bg-slate-100 rounded hover:bg-slate-200 text-slate-700">
                    #{tag}
                  </button>
                ))}
              </div>
            )}
            <button type="button" onClick={handleSaveTags} disabled={savingTags}
              className="py-2 px-4 bg-rose-500 text-white font-medium rounded-lg hover:bg-rose-600 disabled:opacity-50 transition">
              {savingTags ? 'Saving tags...' : 'Save tags'}
            </button>
            {tagsError && <div className="bg-rose-50 text-rose-700 px-4 py-2 rounded-lg text-sm">{tagsError}</div>}
            {tagsMessage && <div className="bg-emerald-50 text-emerald-700 px-4 py-2 rounded-lg text-sm">{tagsMessage}</div>}
          </div>

          <div className="p-5 bg-white rounded-xl border border-slate-200">
            <div className="flex items-center justify-between mb-3">
              <p className="text-sm font-semibold text-slate-700">Photos ({photoList.length}/5)</p>
              <label className="px-3 py-1.5 text-sm bg-rose-500 text-white rounded cursor-pointer hover:bg-rose-600">
                {uploading ? 'Uploading...' : 'Upload'}
                <input type="file" accept="image/*" className="hidden" onChange={handleUpload} disabled={uploading || photoList.length >= 5} />
              </label>
            </div>
            {photoList.length === 0 ? (
              <p className="text-xs text-slate-500">Add at least one photo to improve your profile.</p>
            ) : (
              <div className="grid grid-cols-3 gap-3">
                {photoList.map((p) => (
                  <div key={p.id} className="border border-slate-200 rounded-lg p-2">
                    <div className="w-full h-28 bg-slate-100 rounded overflow-hidden">
                      <img src={p.url} alt="User upload" className="w-full h-28 object-cover" referrerPolicy="no-referrer"
                        onError={(e) => { e.target.style.display = 'none'; e.target.nextElementSibling?.classList.remove('hidden') }} />
                      <div className="hidden w-full h-28 flex items-center justify-center text-slate-400 text-xs">Photo</div>
                    </div>
                    <div className="mt-2 flex gap-1 flex-wrap">
                      {!p.is_primary && (
                        <button type="button" onClick={() => handleSetPrimary(p.id)}
                          className="text-xs px-2 py-1 rounded border border-slate-300 hover:bg-slate-50">
                          Set primary
                        </button>
                      )}
                      {p.is_primary && <span className="text-xs px-2 py-1 rounded bg-emerald-50 text-emerald-700">Primary</span>}
                      <button type="button" onClick={() => handleDeletePhoto(p.id)}
                        className="text-xs px-2 py-1 rounded border border-rose-200 text-rose-700 hover:bg-rose-50">
                        Delete
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

        </div>

        <div>
          <form onSubmit={handleSubmit} className="p-5 bg-white rounded-xl border border-slate-200 space-y-4">
            <p className="text-sm font-semibold text-slate-700">Profile details</p>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">Bio</label>
              <textarea name="bio" value={data.bio} onChange={handleChange} rows={4} maxLength={500}
                className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
                placeholder="Tell us about yourself..." />
              <p className="text-xs text-slate-500 mt-1">{data.bio.length}/500</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">Gender</label>
              <div className="flex flex-wrap gap-2">
                {GENDERS.map((g) => (
                  <button key={g} type="button"
                    onClick={() => setData((d) => ({ ...d, gender: d.gender === g ? '' : g }))}
                    className={`px-3 py-1.5 rounded-full border text-sm transition ${
                      data.gender === g
                        ? 'border-rose-400 bg-rose-50 text-rose-700'
                        : 'border-slate-200 text-slate-600 hover:border-slate-300'
                    }`}>
                    {g}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">Interested in</label>
              <div className="flex flex-wrap gap-2">
                {PREFERENCES.map((p) => (
                  <button key={p} type="button"
                    onClick={() => setData((d) => ({
                      ...d,
                      sexual_preference: d.sexual_preference.includes(p)
                        ? d.sexual_preference.filter((x) => x !== p)
                        : [...d.sexual_preference, p],
                    }))}
                    className={`px-3 py-1.5 rounded-full border text-sm transition ${
                      data.sexual_preference.includes(p)
                        ? 'border-rose-400 bg-rose-50 text-rose-700'
                        : 'border-slate-200 text-slate-600 hover:border-slate-300'
                    }`}>
                    {p}
                  </button>
                ))}
              </div>
              <p className="text-xs text-slate-400 mt-1">Select all that apply</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">Looking for</label>
              <div className="flex flex-wrap gap-2">
                {RELATIONSHIP_GOALS.map((rg) => (
                  <button key={rg.value} type="button"
                    onClick={() => setData((d) => ({ ...d, relationship_goal: d.relationship_goal === rg.value ? '' : rg.value }))}
                    className={`px-3 py-1.5 rounded-full border text-sm transition ${
                      data.relationship_goal === rg.value
                        ? 'border-rose-400 bg-rose-50 text-rose-700'
                        : 'border-slate-200 text-slate-600 hover:border-slate-300'
                    }`}>
                    {rg.label}
                  </button>
                ))}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Birth date</label>
                <input type="date" name="birth_date" value={data.birth_date} onChange={handleChange}
                  max={new Date().toISOString().split('T')[0]}
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none" />
                <p className="text-xs text-slate-500 mt-1">18+</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Location</label>
                <CityInput
                  ref={cityInputRef}
                  value={data.city}
                  onChange={(val) => setData((d) => ({ ...d, city: val }))}
                  placeholder="Paris, France"
                  className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
                />
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <button type="button" onClick={handleUseMyLocation}
                className="px-3 py-2 rounded border border-slate-300 text-sm text-slate-700 hover:bg-slate-50">
                Use my current GPS location
              </button>
              {data.latitude && data.longitude && !isNaN(parseFloat(data.latitude)) && !isNaN(parseFloat(data.longitude)) && (
                <span className="text-xs text-emerald-600 bg-emerald-50 px-2 py-1 rounded">
                  Location set ({parseFloat(data.latitude).toFixed(2)}, {parseFloat(data.longitude).toFixed(2)})
                </span>
              )}
            </div>
            <button type="submit" disabled={saving}
              className={`w-full py-3 text-white font-medium rounded-lg transition disabled:opacity-50 ${saved ? 'bg-emerald-500' : saving ? 'bg-rose-500 opacity-70' : 'bg-rose-500 hover:bg-rose-600'}`}>
              {saving ? 'Saving...' : saved ? 'Saved!' : 'Save profile'}
            </button>
          </form>
        </div>

      </div>
    </div>
  )
}
