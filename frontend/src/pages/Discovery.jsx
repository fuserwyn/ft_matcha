import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { users } from '../api/client'
import CityInput from '../components/CityInput'

const GENDERS = ['male', 'female', 'non-binary', 'other']
const INTERESTS = ['male', 'female', 'both', 'other']

export default function Discovery() {
  const [list, setList] = useState([])
  const [loading, setLoading] = useState(true)
  const [filtersOpen, setFiltersOpen] = useState(false)
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
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-xl sm:text-2xl font-bold text-slate-800">Discover</h1>
        <button
          type="button"
          onClick={() => setFiltersOpen(!filtersOpen)}
          className="lg:hidden px-4 py-2 text-sm font-medium text-rose-600 border border-rose-200 rounded-lg hover:bg-rose-50"
        >
          {filtersOpen ? 'Hide filters' : 'Filters'}
        </button>
      </div>

      <div className="lg:flex lg:gap-6 lg:items-start">

        {/* ── Sidebar filters ── */}
        <aside className={`lg:w-72 xl:w-80 shrink-0 mb-6 lg:mb-0 lg:sticky lg:top-20 ${filtersOpen ? 'block' : 'hidden lg:block'}`}>
          <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
            <div className="px-4 py-3 border-b border-slate-100 bg-slate-50">
              <p className="text-sm font-semibold text-slate-700">Filters</p>
            </div>

            <div className="p-4 space-y-5">

              {/* Gender */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Gender</p>
                <div className="flex flex-wrap gap-2">
                  {GENDERS.map((g) => (
                    <label key={g} className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-sm cursor-pointer transition ${
                      filters.genders?.includes(g)
                        ? 'border-rose-400 bg-rose-50 text-rose-700'
                        : 'border-slate-200 text-slate-600 hover:border-slate-300'
                    }`}>
                      <input
                        type="checkbox"
                        checked={filters.genders?.includes(g) || false}
                        onChange={() => toggleGender(g)}
                        className="sr-only"
                      />
                      {g}
                      {aggregations.gender[g] != null && (
                        <span className="text-xs opacity-60">({aggregations.gender[g]})</span>
                      )}
                    </label>
                  ))}
                </div>
              </div>

              {/* Looking for */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Looking for</p>
                <div className="flex flex-wrap gap-2">
                  {INTERESTS.map((i) => (
                    <label key={i} className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-sm cursor-pointer transition ${
                      filters.interests?.includes(i)
                        ? 'border-rose-400 bg-rose-50 text-rose-700'
                        : 'border-slate-200 text-slate-600 hover:border-slate-300'
                    }`}>
                      <input
                        type="checkbox"
                        checked={filters.interests?.includes(i) || false}
                        onChange={() => toggleInterest(i)}
                        className="sr-only"
                      />
                      {i}
                      {aggregations.interest[i] != null && (
                        <span className="text-xs opacity-60">({aggregations.interest[i]})</span>
                      )}
                    </label>
                  ))}
                </div>
              </div>

              {/* Age range */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Age</p>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">From</label>
                    <input
                      type="number"
                      name="min_age"
                      value={filters.min_age}
                      onChange={handleFilterChange}
                      min="18" max="99"
                      placeholder="18"
                      className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">To</label>
                    <input
                      type="number"
                      name="max_age"
                      value={filters.max_age}
                      onChange={handleFilterChange}
                      min="18" max="99"
                      placeholder="99"
                      className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
                    />
                  </div>
                </div>
              </div>

              {/* Fame range */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Fame rating</p>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">Min</label>
                    <input
                      type="number"
                      name="min_fame"
                      value={filters.min_fame}
                      onChange={handleFilterChange}
                      min="0"
                      placeholder="0"
                      className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">Max</label>
                    <input
                      type="number"
                      name="max_fame"
                      value={filters.max_fame}
                      onChange={handleFilterChange}
                      min="0"
                      placeholder="100"
                      className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
                    />
                  </div>
                </div>
              </div>

              {/* City */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">City</p>
                <CityInput
                  value={filters.city}
                  onChange={(val) => setFilters((f) => ({ ...f, city: val }))}
                  placeholder="Paris, Amsterdam..."
                  className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
                />
              </div>

              {/* Tags */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Tags</p>
                <div className="relative">
                  <input
                    type="text"
                    name="tags"
                    value={filters.tags}
                    onChange={handleFilterChange}
                    onBlur={() => setTimeout(() => setTagSuggestionsOpen(false), 150)}
                    onFocus={() => tagSuggestions.length > 0 && setTagSuggestionsOpen(true)}
                    placeholder="music, travel..."
                    autoComplete="off"
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
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
              </div>

              {/* Distance */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Max distance (km)</p>
                <input
                  type="number"
                  name="max_distance_km"
                  value={filters.max_distance_km}
                  onChange={handleFilterChange}
                  min="1"
                  placeholder="50"
                  className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
                />
              </div>

              {/* Sort */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Sort</p>
                <div className="grid grid-cols-2 gap-2">
                  <select
                    name="sort_by"
                    value={filters.sort_by}
                    onChange={handleFilterChange}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
                  >
                    <option value="">Relevance</option>
                    <option value="age">Age</option>
                    <option value="location">Location</option>
                    <option value="fame">Fame</option>
                    <option value="tags">Tags</option>
                  </select>
                  <select
                    name="sort_order"
                    value={filters.sort_order}
                    onChange={handleFilterChange}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300"
                  >
                    <option value="">Default</option>
                    <option value="asc">ASC</option>
                    <option value="desc">DESC</option>
                  </select>
                </div>
              </div>

            </div>
          </div>
        </aside>

        {/* ── Results grid ── */}
        <div className="flex-1 min-w-0">
          {loading ? (
            <div className="flex justify-center py-12">
              <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
            </div>
          ) : list.length === 0 ? (
            <p className="text-slate-500 text-center py-12">No users found</p>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 sm:gap-4">
              {list.map((u) => {
                const displayName = (u.first_name && u.last_name && u.first_name === u.last_name)
                  ? u.first_name
                  : [u.first_name, u.last_name].filter(Boolean).join(' ')
                const initial = (u.first_name?.[0] || u.username?.[0] || '?').toUpperCase()
                return (
                  <Link
                    key={u.id}
                    to={`/users/${u.id}`}
                    className="group relative block rounded-2xl overflow-hidden aspect-[3/4] bg-slate-100 hover:shadow-xl transition-shadow active:scale-[0.98]"
                  >
                    {u.primary_photo_url ? (
                      <img
                        src={u.primary_photo_url}
                        alt={displayName}
                        className="absolute inset-0 w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
                        referrerPolicy="no-referrer"
                      />
                    ) : (
                      <div className="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-slate-100 to-slate-200">
                        <span className="text-7xl font-bold text-slate-300">{initial}</span>
                      </div>
                    )}
                    {/* gradient overlay */}
                    <div className="absolute inset-0 bg-gradient-to-t from-black/75 via-black/10 to-transparent" />
                    {/* fame badge */}
                    {u.fame_rating > 0 && (
                      <div className="absolute top-3 right-3 px-2 py-0.5 rounded-full bg-black/40 backdrop-blur-sm text-amber-300 text-xs font-semibold">
                        ★ {u.fame_rating}
                      </div>
                    )}
                    {/* info overlay */}
                    <div className="absolute bottom-0 inset-x-0 p-4 text-white">
                      <div className="font-bold text-lg leading-tight truncate drop-shadow">
                        {displayName || u.username}{u.birth_date ? `, ${age(u.birth_date)}` : ''}
                      </div>
                      {u.city && (
                        <div className="text-xs text-white/80 mt-0.5 truncate">📍 {u.city}</div>
                      )}
                      {Array.isArray(u.tags) && u.tags.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {u.tags.slice(0, 3).map((tag) => (
                            <span key={tag} className="text-[10px] px-2 py-0.5 bg-white/20 backdrop-blur-sm rounded-full">
                              #{tag}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                  </Link>
                )
              })}
            </div>
          )}
        </div>

      </div>
    </div>
  )
}
