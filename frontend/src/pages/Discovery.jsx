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
  })

  const load = async () => {
    setLoading(true)
    try {
      const params = {}
      if (filters.gender) params.gender = filters.gender
      if (filters.interest) params.interest = filters.interest
      if (filters.min_age) params.min_age = filters.min_age
      if (filters.max_age) params.max_age = filters.max_age
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
  }, [filters.gender, filters.interest, filters.min_age, filters.max_age])

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
            <label className="block text-xs text-slate-500 mb-1">Interested in</label>
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
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
