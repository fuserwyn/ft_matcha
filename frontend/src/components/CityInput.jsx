import { useState, useEffect, useRef } from 'react'

export default function CityInput({ value, onChange, placeholder = 'Paris, Amsterdam...', className = '' }) {
  const [suggestions, setSuggestions] = useState([])
  const [open, setOpen] = useState(false)
  const [highlighted, setHighlighted] = useState(-1)
  const containerRef = useRef(null)
  const justSelected = useRef(false)
  const isMount = useRef(true)

  useEffect(() => {
    if (isMount.current) {
      isMount.current = false
      return
    }
    if (justSelected.current) {
      justSelected.current = false
      return
    }
    const q = value.trim()
    if (q.length < 2) {
      setSuggestions([])
      setOpen(false)
      return
    }
    const t = setTimeout(() => {
      fetch(`https://photon.komoot.io/api/?q=${encodeURIComponent(q)}&limit=8&layer=city`)
        .then((r) => r.json())
        .then((data) => {
          const seen = new Set()
          const cities = (data.features || [])
            .map((f) => {
              const { name = '', state = '', country = '' } = f.properties
              const parts = [name, state, country].filter(Boolean)
              const label = parts.join(', ')
              return { name: label, label }
            })
            .filter(({ label }) => {
              if (!label || seen.has(label)) return false
              seen.add(label)
              return true
            })
          setSuggestions(cities)
          setOpen(cities.length > 0)
          setHighlighted(-1)
        })
        .catch(() => setSuggestions([]))
    }, 250)
    return () => clearTimeout(t)
  }, [value])

  useEffect(() => {
    const handler = (e) => {
      if (containerRef.current && !containerRef.current.contains(e.target)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const select = (city) => {
    justSelected.current = true
    onChange(city.name)
    setOpen(false)
    setSuggestions([])
  }

  const handleKeyDown = (e) => {
    if (!open || suggestions.length === 0) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setHighlighted((h) => Math.min(h + 1, suggestions.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setHighlighted((h) => Math.max(h - 1, 0))
    } else if (e.key === 'Enter' && highlighted >= 0) {
      e.preventDefault()
      select(suggestions[highlighted])
    } else if (e.key === 'Escape') {
      setOpen(false)
    }
  }

  return (
    <div ref={containerRef} className="relative">
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onFocus={() => suggestions.length > 0 && setOpen(true)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        autoComplete="off"
        className={className}
      />
      {open && suggestions.length > 0 && (
        <ul className="absolute z-20 mt-1 w-full bg-white border border-slate-200 rounded-lg shadow-lg max-h-52 overflow-auto">
          {suggestions.map((city, idx) => (
            <li key={city.label}>
              <button
                type="button"
                onMouseDown={(e) => { e.preventDefault(); select(city) }}
                className={`block w-full text-left px-3 py-2 text-sm transition ${
                  idx === highlighted ? 'bg-rose-50 text-rose-700' : 'text-slate-700 hover:bg-slate-50'
                }`}
              >
                📍 {city.label}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
