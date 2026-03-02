import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { users } from '../api/client'

const GENDERS = ['male', 'female', 'non-binary', 'other']
const INTERESTS = ['male', 'female', 'both', 'other']

export default function Discovery() {
  const [list, setList] = useState([])
  const [loading, setLoading] = useState(true)
  const [filters, setFilters] = useState({
    gender: '',
    interest: '',
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

  const load = async () => {
    setLoading(true)
    try {
      const params = {}
      if (filters.interest && filters.interest !== 'both') {
        // "Interested in" should filter target profile gender.
        params.gender = filters.interest
      } else if (filters.gender) {
        params.gender = filters.gender
      }
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
    load()
  }, [
    filters.gender,
    filters.interest,
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

  const age = (birthDate) => {
    if (!birthDate) return null
    const diff = Date.now() - new Date(birthDate).getTime()
    return Math.floor(diff / (365.25 * 24 * 60 * 60 * 1000))
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-slate-800 mb-6">Discover</h1>

      <div className="mb-6 p-4 bg-white rounded-lg border border-slate-200">
        <p className="text-sm font-medium text-slate-700 mb-3">Filters</p>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <label className="block text-xs text-slate-500 mb-1">Gender</label>
            <select
              name="gender"
              value={filters.gender}
              onChange={handleFilterChange}
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            >
              <option value="">Any</option>
              {GENDERS.map((g) => (
                <option key={g} value={g}>{g}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Looking for</label>
            <select
              name="interest"
              value={filters.interest}
              onChange={handleFilterChange}
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            >
              <option value="">Any</option>
              {INTERESTS.map((i) => (
                <option key={i} value={i}>{i}</option>
              ))}
            </select>
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
              placeholder="Paris"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Tags (comma)</label>
            <input
              type="text"
              name="tags"
              value={filters.tags}
              onChange={handleFilterChange}
              placeholder="music,travel"
              className="w-full px-3 py-2 rounded border border-slate-200 text-sm"
            />
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
          {list.map((u) => (
            <Link
              key={u.id}
              to={`/users/${u.id}`}
              className="block p-4 bg-white rounded-lg border border-slate-200 hover:border-rose-300 hover:shadow-md transition"
            >
              {u.primary_photo_url && (
                <img
                  src={u.primary_photo_url}
                  alt={`${u.first_name} ${u.last_name}`}
                  className="w-full h-40 object-cover rounded mb-3"
                />
              )}
              <div className="font-semibold text-slate-800">
                {u.first_name} {u.last_name}
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
                <p className="mt-2 text-sm text-slate-600 line-clamp-2">{u.bio}</p>
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
          ))}
        </div>
      )}
    </div>
  )
}
