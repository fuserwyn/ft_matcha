import { useState, useEffect, useRef } from 'react'
import { users, profile } from '../api/client'
import CityInput from '../components/CityInput'
import ProfileModal from '../components/ProfileModal'

const INTERESTS = ['male', 'female', 'non-binary', 'other']
const RELATIONSHIP_GOALS = [
  { value: 'long-term', label: 'Long-term' },
  { value: 'long-term-open', label: 'Long-term open' },
  { value: 'short-term-open', label: 'Short-term open' },
  { value: 'short-term', label: 'Short-term' },
  { value: 'friends', label: 'Friends' },
  { value: 'not-sure', label: 'Not sure' },
]

const GOAL_LABELS = {
  'long-term': 'Long-Term',
  'long-term-open': 'Long-Term, open to short',
  'short-term-open': 'Short-Term, open to long',
  'short-term': 'Short-Term',
  'friends': 'Friends',
  'not-sure': 'Not sure',
}

// Module-level cache — survives React unmount/remount within the same session
let cache = null

const defaultFilters = {
  interests: [],
  relationship_goals: [],
  min_age: '',
  max_age: '',
  min_fame: '',
  max_fame: '',
  city: '',
  tags: '',
  max_distance_km: '',
  sort_by: '',
  sort_order: '',
}

export default function Discovery() {
  const [list, setList] = useState(() => cache?.list || [])
  const [loading, setLoading] = useState(!cache)
  const [selectedUserId, setSelectedUserId] = useState(null)
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [tagSuggestions, setTagSuggestions] = useState([])
  const [tagSuggestionsOpen, setTagSuggestionsOpen] = useState(false)
  const [aggregations, setAggregations] = useState({ gender: {}, interest: {}, relationship_goal: {} })
  const [filters, setFilters] = useState(() => cache?.filters || defaultFilters)

  // pendingScroll: set when restoring from cache, consumed once list renders
  const pendingScroll = useRef(cache?.scrollY ?? null)
  // Track if filters changed by user (vs. initial mount)
  const initialMount = useRef(true)
  // Suppress one filter-change effect cycle after profile pre-population
  const suppressNextFilterEffect = useRef(false)

  // Scroll to saved position once the list is rendered
  useEffect(() => {
    if (pendingScroll.current !== null && list.length > 0) {
      const y = pendingScroll.current
      pendingScroll.current = null
      // Double rAF ensures the DOM has fully painted
      requestAnimationFrame(() => requestAnimationFrame(() => window.scrollTo(0, y)))
    }
  }, [list])

  // Save to cache on unmount
  useEffect(() => {
    return () => {
      cache = {
        list,
        filters,
        scrollY: window.scrollY,
      }
    }
  })

  // Initial mount: always pre-populate interests/goals from profile; use cache for everything else
  useEffect(() => {
    users.filterAggregations()
      .then((r) => setAggregations({ gender: r.gender || {}, interest: r.interest || {}, relationship_goal: r.relationship_goal || {} }))
      .catch(() => {})

    profile.get()
      .then((p) => {
        const base = cache?.filters || defaultFilters
        const initFilters = {
          ...base,
          interests: Array.isArray(p.sexual_preference) && p.sexual_preference.length > 0
            ? p.sexual_preference
            : base.interests,
          relationship_goals: p.relationship_goal ? [p.relationship_goal] : base.relationship_goals,
        }
        suppressNextFilterEffect.current = true
        setFilters(initFilters)
        load(initFilters)
      })
      .catch(() => { if (!cache) load(defaultFilters) })
  }, [])

  const load = async (f) => {
    setLoading(true)
    try {
      const params = {}
      if (f.interests?.length > 0) params.interest = f.interests.join(',')
      if (f.relationship_goals?.length > 0) params.relationship_goal = f.relationship_goals.join(',')
      if (f.min_age) params.min_age = f.min_age
      if (f.max_age) params.max_age = f.max_age
      if (f.min_fame) params.min_fame = f.min_fame
      if (f.max_fame) params.max_fame = f.max_fame
      if (f.city) params.city = f.city
      if (f.tags) params.tags = f.tags
      if (f.max_distance_km) params.max_distance_km = f.max_distance_km
      if (f.sort_by) params.sort_by = f.sort_by
      if (f.sort_order) params.sort_order = f.sort_order
      params.limit = 500
      const data = await users.search(params)
      setList(data)
    } catch {
      setList([])
    } finally {
      setLoading(false)
    }
  }

  // Reload when filters change (skip initial mount and profile pre-population)
  useEffect(() => {
    if (initialMount.current) {
      initialMount.current = false
      return
    }
    if (suppressNextFilterEffect.current) {
      suppressNextFilterEffect.current = false
      return
    }
    pendingScroll.current = null // user changed filters, don't restore old scroll
    load(filters)
  }, [
    filters.interests,
    filters.relationship_goals,
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

  // Tag suggestions
  useEffect(() => {
    const lastComma = filters.tags.lastIndexOf(',')
    const prefix = (lastComma >= 0 ? filters.tags.slice(lastComma + 1) : filters.tags).trim().toLowerCase()
    if (prefix.length < 2) {
      setTagSuggestions([])
      setTagSuggestionsOpen(false)
      return
    }
    const t = setTimeout(() => {
      profile.tagSuggestions(prefix)
        .then((r) => { setTagSuggestions(r.tags || []); setTagSuggestionsOpen(true) })
        .catch(() => setTagSuggestions([]))
    }, 200)
    return () => clearTimeout(t)
  }, [filters.tags])

  const handleFilterChange = (e) => {
    const { name, value } = e.target
    setFilters((f) => ({ ...f, [name]: value }))
  }

  const toggleInterest = (i) => {
    setFilters((f) => ({
      ...f,
      interests: f.interests?.includes(i) ? f.interests.filter((x) => x !== i) : [...(f.interests || []), i],
    }))
  }
  const toggleRelationshipGoal = (g) => {
    setFilters((f) => ({
      ...f,
      relationship_goals: f.relationship_goals?.includes(g) ? f.relationship_goals.filter((x) => x !== g) : [...(f.relationship_goals || []), g],
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

              {/* Interested in */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Interested in</p>
                <div className="flex flex-wrap gap-2">
                  {INTERESTS.map((i) => (
                    <label key={i} className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-sm cursor-pointer transition ${
                      filters.interests?.includes(i)
                        ? 'border-rose-400 bg-rose-50 text-rose-700'
                        : 'border-slate-200 text-slate-600 hover:border-slate-300'
                    }`}>
                      <input type="checkbox" checked={filters.interests?.includes(i) || false}
                        onChange={() => toggleInterest(i)} className="sr-only" />
                      {i}
                    </label>
                  ))}
                </div>
              </div>

              {/* Looking for (relationship goal) */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Looking for</p>
                <div className="flex flex-wrap gap-2">
                  {RELATIONSHIP_GOALS.map((rg) => (
                    <label key={rg.value} className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-sm cursor-pointer transition ${
                      filters.relationship_goals?.includes(rg.value)
                        ? 'border-rose-400 bg-rose-50 text-rose-700'
                        : 'border-slate-200 text-slate-600 hover:border-slate-300'
                    }`}>
                      <input type="checkbox" checked={filters.relationship_goals?.includes(rg.value) || false}
                        onChange={() => toggleRelationshipGoal(rg.value)} className="sr-only" />
                      {rg.label}
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
                    <input type="number" name="min_age" value={filters.min_age} onChange={handleFilterChange}
                      min="18" max="99" placeholder="18"
                      className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300" />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">To</label>
                    <input type="number" name="max_age" value={filters.max_age} onChange={handleFilterChange}
                      min="18" max="99" placeholder="99"
                      className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300" />
                  </div>
                </div>
              </div>

              {/* Fame range */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Fame rating</p>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">Min</label>
                    <input type="number" name="min_fame" value={filters.min_fame} onChange={handleFilterChange}
                      min="0" placeholder="0"
                      className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300" />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">Max</label>
                    <input type="number" name="max_fame" value={filters.max_fame} onChange={handleFilterChange}
                      min="0" placeholder="100"
                      className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300" />
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
                  <input type="text" name="tags" value={filters.tags} onChange={handleFilterChange}
                    onBlur={() => setTimeout(() => setTagSuggestionsOpen(false), 150)}
                    onFocus={() => tagSuggestions.length > 0 && setTagSuggestionsOpen(true)}
                    placeholder="music, travel..." autoComplete="off"
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300" />
                  {tagSuggestionsOpen && tagSuggestions.length > 0 && (
                    <div className="absolute z-10 mt-1 w-full bg-white border border-slate-200 rounded-lg shadow-lg max-h-40 overflow-auto">
                      {tagSuggestions.map((t) => (
                        <button key={t} type="button"
                          className="block w-full text-left px-3 py-2 text-sm hover:bg-rose-50"
                          onClick={() => applyTagSuggestion(t)}>
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
                <input type="number" name="max_distance_km" value={filters.max_distance_km}
                  onChange={handleFilterChange} min="1" placeholder="50"
                  className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300" />
              </div>

              {/* Sort */}
              <div>
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Sort by</p>
                <select name="sort_by" value={filters.sort_by} onChange={handleFilterChange}
                  className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:border-rose-300">
                  <option value="">Relevance</option>
                  <option value="last_online">Recently Online</option>
                  <option value="location">Closest to Me</option>
                  <option value="fame">Fame Rating</option>
                  <option value="age">Age</option>
                  <option value="tags">Common Tags</option>
                </select>
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
                  <button key={u.id} type="button"
                    onClick={() => setSelectedUserId(u.id)}
                    className="group relative block w-full rounded-2xl overflow-hidden aspect-[3/4] bg-slate-100 hover:shadow-xl transition-shadow active:scale-[0.98] cursor-pointer">
                    {u.primary_photo_url ? (
                      <img src={u.primary_photo_url} alt={displayName}
                        className="absolute inset-0 w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
                        referrerPolicy="no-referrer" />
                    ) : (
                      <div className="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-slate-100 to-slate-200">
                        <span className="text-7xl font-bold text-slate-300">{initial}</span>
                      </div>
                    )}
                    <div className="absolute inset-0 bg-gradient-to-t from-black/75 via-black/10 to-transparent" />
                    <div className="absolute top-3 left-3 right-3 flex justify-between items-start gap-2">
                      {u.relationship_goal && GOAL_LABELS[u.relationship_goal] && (
                        <span className="px-2 py-0.5 rounded-full bg-rose-500/80 backdrop-blur-sm text-white text-[10px] font-medium">
                          {GOAL_LABELS[u.relationship_goal]}
                        </span>
                      )}
                      {u.fame_rating > 0 && (
                        <span className="ml-auto px-2 py-0.5 rounded-full bg-black/40 backdrop-blur-sm text-amber-300 text-xs font-semibold">
                          ★ {u.fame_rating}
                        </span>
                      )}
                    </div>
                    <div className="absolute bottom-0 inset-x-0 p-4 text-white">
                      <div className="font-bold text-lg leading-tight truncate drop-shadow">
                        {displayName || u.username}{u.birth_date ? `, ${age(u.birth_date)}` : ''}
                      </div>
                      {u.city && <div className="text-xs text-white/80 mt-0.5 truncate">📍 {u.city}</div>}
                      {u.gender && (
                        <div className="text-[10px] text-white/60 mt-0.5">{u.gender}</div>
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
                  </button>
                )
              })}
            </div>
          )}
        </div>

      </div>

      {selectedUserId && (
        <ProfileModal userId={selectedUserId} onClose={() => setSelectedUserId(null)} />
      )}
    </div>
  )
}
