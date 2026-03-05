import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { profile } from '../api/client'

const TABS = [
  { id: 'by-me', label: 'I viewed', fetch: profile.getViewedHistory },
  { id: 'viewed-me', label: 'Viewed me', fetch: profile.getViewedBy },
]

export default function Views() {
  const [activeTab, setActiveTab] = useState('by-me')
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

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
        if (active) setError(err.message || 'Failed to load views')
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [activeTab])

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-slate-800 mb-6">Profile views</h1>

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
          {isByMe ? 'No profiles viewed yet.' : 'No one has viewed your profile yet.'}
        </p>
      ) : (
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
                <p className="text-xs text-white/60 mb-3">
                  {isByMe ? 'Viewed' : 'Viewed you'} · {new Date(u.last_viewed_at).toLocaleString()}
                </p>
                <Link to={`/users/${u.id}`}
                  className="inline-block px-4 py-1.5 rounded-full bg-white/20 backdrop-blur-sm text-xs text-white hover:bg-white/30 transition">
                  View profile
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
