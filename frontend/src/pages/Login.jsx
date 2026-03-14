import { useState, useEffect } from 'react'
import { Link, useNavigate, useLocation, useSearchParams } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { auth } from '../api/client'

export default function Login() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = location.state?.from?.pathname || '/'

  useEffect(() => {
    const u = searchParams.get('username')
    const verified = searchParams.get('verified')
    const err = searchParams.get('error')
    const already = searchParams.get('already')
    if (u) setUsername(decodeURIComponent(u))
    if (verified === '1') setInfo('Email verified! Enter your password to sign in.')
    if (err === 'token_required') setError('Verification link is invalid or expired.')
    else if (err === 'verify_failed') setError('Verification failed. Please try again.')
    else if (err === 'internal') setError('Something went wrong. Please try again.')
    if (already === '1') setInfo('Your email is already verified. Sign in below.')
    if (u || verified || err || already) {
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, setSearchParams])

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const { token, user } = await auth.login({ username, password })
      login(token, user)
      navigate(from, { replace: true })
    } catch (err) {
      setError(err.message || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-md mx-auto w-full px-2 sm:px-0">
      <div className="bg-white rounded-2xl shadow-lg p-8 border border-slate-100">
        <h1 className="text-2xl font-bold text-slate-800 mb-6">Welcome back</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {info && (
            <div className="bg-emerald-50 text-emerald-700 px-4 py-3 rounded-lg text-sm">
              {info}
            </div>
          )}
          {error && (
            <div className="bg-rose-50 text-rose-700 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Username</label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none transition"
              placeholder="your_username"
              required
              autoComplete="username"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-2 rounded-lg border border-slate-200 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none transition"
              placeholder="••••••••"
              required
              minLength={8}
              maxLength={72}
              autoComplete="current-password"
            />
            <p className="text-xs text-slate-500 mt-1">
              8–72 characters. Avoid common passwords (e.g. password123, qwerty).
            </p>
            <div className="mt-2 text-right">
              <Link to="/forgot-password" className="text-sm text-rose-600 hover:underline">
                Forgot password?
              </Link>
            </div>
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 bg-rose-500 text-white font-medium rounded-lg hover:bg-rose-600 disabled:opacity-50 transition"
          >
            {loading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
        <p className="mt-6 text-center text-slate-600 text-sm">
          Don't have an account?{' '}
          <Link to="/register" className="text-rose-600 font-medium hover:underline">
            Sign up
          </Link>
        </p>
      </div>
    </div>
  )
}
