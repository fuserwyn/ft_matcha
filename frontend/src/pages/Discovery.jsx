import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { users, profile } from '../api/client'

const GENDERS = ['male', 'female', 'non-binary', 'other']
const INTERESTS = ['male', 'female', 'both', 'other']

export default function Discovery() {
  const [list, setList] = useState([])
  const [loading, setLoading] = useState(true)
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [citySuggestions, setCitySuggestions] = useState([])
  const [tagSuggestions, setTagSuggestions] = useState([])
  const [tagSuggestionsOpen, setTagSuggestionsOpen] = useState(false)
  const [aggregations, setAggregations] = useState({ gender: {}, interest: {} })
  const [filters, setFilters] = useState({
    genders: [],
    interests: [],
    min_age: '',
    max_age: '',
    min_fame: '',
    max_fame: '',
    city: '',
    tags: '',
    max_distance_km: '',
    sort_by: '',
    sort_order: '',
  })

  useEffect(() => {
    users.filterAggregations().then((r) => setAggregations({ gender: r.gender || {}, interest: r.interest || {} })).catch(() => {})
  }, [])

  const load = async () => {
    setLoading(true)
    try {
      const params = {}
      if (filters.genders?.length > 0) params.gender = filters.genders.join(',')
      if (filters.interests?.length > 0) params.interest = filters.interests.join(',')
      if (filters.min_age) params.min_age = filters.min_age
      if (filters.max_age) params.max_age = filters.max_age
      if (filters.min_fame) params.min_fame = filters.min_fame
      if (filters.max_fame) params.max_fame = filters.max_fame
      if (filters.city) params.city = filters.city
      if (filters.tags) params.tags = filters.tags
      if (filters.max_distance_km) params.max_distance_km = filters.max_distance_km
      if (filters.sort_by) params.sort_by = filters.sort_by
      if (filters.sort_order) params.sort_order = filters.sort_order
      params.limit = 500
      const data = await users.search(params)
      setList(data)
    } catch (err) {
      setList([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const q = filters.city.trim()
    if (q.length < 2) {
      setCitySuggestions([])
      return
    }
    const t = setTimeout(() => {
      profile.citySuggestions(q).then((r) => setCitySuggestions(r.cities || [])).catch(() => setCitySuggestions([]))
    }, 200)
    return () => clearTimeout(t)
  }, [filters.city])

  useEffect(() => {
    const lastComma = filters.tags.lastIndexOf(',')
    const prefix = (lastComma >= 0 ? filters.tags.slice(lastComma + 1) : filters.tags).trim().toLowerCase()
    if (prefix.length < 2) {
      setTagSuggestions([])
      setTagSuggestionsOpen(false)
      return
    }
    const t = setTimeout(() => {
      profile.tagSuggestions(prefix).then((r) => {
        setTagSuggestions(r.tags || [])
        setTagSuggestionsOpen(true)
      }).catch(() => setTagSuggestions([]))
    }, 200)
    return () => clearTimeout(t)
  }, [filters.tags])

  useEffect(() => {
    load()
  }, [
    filters.genders,
    filters.interests,
    filters.min_age,
    filters.max_age,
    filters.min_fame,
    filters.max_fame,
    filters.city,
    filters.tags,
    filters.max_distance_km,
    filters.sort_by,
    filters.sort_order,
  ])

  const handleFilterChange = (e) => {
    const { name, value } = e.target
    setFilters((f) => ({ ...f, [name]: value }))
  }

  const toggleGender = (g) => {
    setFilters((f) => ({
      ...f,
      genders: f.genders?.includes(g) ? f.genders.filter((x) => x !== g) : [...(f.genders || []), g],
    }))
  }
  const toggleInterest = (i) => {
    setFilters((f) => ({
      ...f,
      interests: f.interests?.includes(i) ? f.interests.filter((x) => x !== i) : [...(f.interests || []), i],
    }))
  }

  const applyTagSuggestion = (tag) => {
    const lastComma = filters.tags.lastIndexOf(',')
    const before = lastComma >= 0 ? filters.tags.slice(0, lastComma + 1) : ''
    setFilters((f) => ({ ...f, tags: (before + (before ? ' ' : '') + tag).trim() }))
    setTagSuggestionsOpen(false)
  }

  const age = (birthDate) => {
    if (!birthDate) return null
    const diff = Date.now() - new Date(birthDate).getTime()
    return Math.floor(diff / (365.25 * 24 * 60 * 60 * 1000))
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl sm:text-2xl font-bold text-slate-800">Discover</h1>
        <button
          type="button"
          onClick={() => setFiltersOpen(!filtersOpen)}
          className="lg:hidden px-4 py-2 text-sm font-medium text-rose-600 border border-rose-200 rounded-lg hover:bg-rose-50"
        >
          {filtersOpen ? 'Hide filters' : 'Filters'}
        </button>
      </div>

      <div className={`mb-6 p-4 bg-white rounded-lg border border-slate-200 ${filtersOpen ? 'block' : 'hidden lg:block'}`}>
        <p className="text-sm font-medium text-slate-700 mb-3">Filters</p>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
          <div>
            <label className="block text-xs text-slate-500 mb-1">Gender</label>
            <div className="flex flex-wrap gap-2">
              {GENDERS.map((g) => (
                <label key={g} className="flex items-center gap-1.5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={filters.genders?.includes(g) || false}
                    onChange={() => toggleGender(g)}
                    className="rounded border-slate-300 text-rose-500 focus:ring-rose-400"
                  />
                  <span className="text-sm">
                    {g}
                    {aggregations.gender[g] != null && (
                      <span className="text-slate-400 ml-0.5">({aggregations.gender[g]})</span>
                    )}
                  </span>
                </label>
              ))}
            </div>
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Looking for</label>
            <div className="flex flex-wrap gap-2">
              {INTERESTS.map((i) => (
                <label key={i} className="flex items-center gap-1.5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={filters.interests?.includes(i) || false}
                    onChange={() => toggleInterest(i)}
                    className="rounded border-slate-300 text-rose-500 focus:ring-rose-400"
                  />
                  <span className="text-sm">
                    {i}
                    {aggregations.interest[i] != null && (
                      <span className="text-slate-400 ml-0.5">({aggregations.interest[i]})</span>
                    )}
                  </span>
                </label>
              ))}
            </div>
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Min age</label>
            <input
              type="number"
              name="min_age"
              value={filters.min_age}
              onChange={handleFilterChange}
              min="18"
              max="99"
              placeholder="18"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Max age</label>
            <input
              type="number"
              name="max_age"
              value={filters.max_age}
              onChange={handleFilterChange}
              min="18"
              max="99"
              placeholder="99"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Min fame</label>
            <input
              type="number"
              name="min_fame"
              value={filters.min_fame}
              onChange={handleFilterChange}
              min="0"
              placeholder="0"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Max fame</label>
            <input
              type="number"
              name="max_fame"
              value={filters.max_fame}
              onChange={handleFilterChange}
              min="0"
              placeholder="100"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">City</label>
            <input
              type="text"
              name="city"
              value={filters.city}
              onChange={handleFilterChange}
              placeholder="Par, Amster, Paris..."
              list="city-suggestions"
              autoComplete="off"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
            <datalist id="city-suggestions">
              {citySuggestions.map((c) => (
                <option key={c} value={c} />
              ))}
            </datalist>
          </div>
          <div className="relative">
            <label className="block text-xs text-slate-500 mb-1">Tags (comma, partial: mus → music)</label>
            <input
              type="text"
              name="tags"
              value={filters.tags}
              onChange={handleFilterChange}
              onBlur={() => setTimeout(() => setTagSuggestionsOpen(false), 150)}
              onFocus={() => tagSuggestions.length > 0 && setTagSuggestionsOpen(true)}
              placeholder="music, travel, mus..."
              autoComplete="off"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
            {tagSuggestionsOpen && tagSuggestions.length > 0 && (
              <div className="absolute z-10 mt-1 w-full bg-white border border-slate-200 rounded-lg shadow-lg max-h-40 overflow-auto">
                {tagSuggestions.map((t) => (
                  <button
                    key={t}
                    type="button"
                    className="block w-full text-left px-3 py-2 text-sm hover:bg-rose-50"
                    onClick={() => applyTagSuggestion(t)}
                  >
                    {t}
                  </button>
                ))}
              </div>
            )}
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Max distance (km)</label>
            <input
              type="number"
              name="max_distance_km"
              value={filters.max_distance_km}
              onChange={handleFilterChange}
              min="1"
              placeholder="50"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Sort by</label>
            <select
              name="sort_by"
              value={filters.sort_by}
              onChange={handleFilterChange}
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            >
              <option value="">Relevance</option>
              <option value="age">Age</option>
              <option value="location">Location</option>
              <option value="fame">Fame</option>
              <option value="tags">Tags</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Sort order</label>
            <select
              name="sort_order"
              value={filters.sort_order}
              onChange={handleFilterChange}
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            >
              <option value="">Default</option>
              <option value="asc">ASC</option>
              <option value="desc">DESC</option>
            </select>
          </div>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
        </div>
      ) : list.length === 0 ? (
        <p className="text-slate-500 text-center py-12">No users found</p>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {list.map((u) => {
            const displayName = (u.first_name && u.last_name && u.first_name === u.last_name)
              ? u.first_name
              : [u.first_name, u.last_name].filter(Boolean).join(' ')
            return (
            <Link
              key={u.id}
              to={`/users/${u.id}`}
              className="block p-4 bg-white rounded-xl border border-slate-200 hover:border-rose-300 hover:shadow-md transition active:scale-[0.99]"
            >
              {u.primary_photo_url ? (
                <img
                  src={u.primary_photo_url}
                  alt={displayName}
                  className="w-full h-40 sm:h-36 object-cover rounded-lg mb-3"
                  referrerPolicy="no-referrer"
                />
              ) : (
                <div className="w-full h-40 sm:h-36 bg-slate-100 rounded-lg mb-3 flex items-center justify-center text-slate-400 text-sm">
                  No photo
                </div>
              )}
              <div className="font-semibold text-slate-800 truncate">
                {displayName || u.username}
              </div>
              <div className="text-sm text-slate-500">@{u.username}</div>
              {u.gender && (
                <span className="text-xs text-slate-500">{u.gender}</span>
              )}
              {u.birth_date && (
                <span className="text-xs text-slate-500 ml-2">
                  {age(u.birth_date)} y.o.
                </span>
              )}
              {u.bio && (
                <p className="mt-2 text-sm text-slate-600 line-clamp-2 break-words">{u.bio}</p>
              )}
              {u.fame_rating > 0 && (
                <div className="mt-2 text-xs text-rose-500">★ {u.fame_rating}</div>
              )}
              {u.city && <div className="mt-1 text-xs text-slate-500">City: {u.city}</div>}
              {Array.isArray(u.tags) && u.tags.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
                  {u.tags.slice(0, 4).map((tag) => (
                    <span key={tag} className="text-[11px] px-2 py-0.5 bg-slate-100 rounded text-slate-600">
                      #{tag}
                    </span>
                  ))}
                </div>
              )}
            </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}
