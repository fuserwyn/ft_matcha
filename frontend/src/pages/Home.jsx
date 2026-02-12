import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function Home() {
  const { user } = useAuth()

  return (
    <div className="text-center py-16">
      <h1 className="text-4xl font-bold text-slate-800 mb-4">Matcha</h1>
      <p className="text-slate-600 text-lg mb-8 max-w-md mx-auto">
        Dating app — find your match. Register to create a profile and start meeting people.
      </p>
      {user ? (
        <div className="space-y-4">
          <p className="text-slate-600">Welcome, {user.first_name}!</p>
          <Link
            to="/profile"
            className="inline-block px-6 py-3 bg-rose-500 text-white font-medium rounded-lg hover:bg-rose-600 transition"
          >
            Edit profile
          </Link>
        </div>
      ) : (
        <div className="flex gap-4 justify-center">
          <Link
            to="/login"
            className="px-6 py-3 border border-slate-300 text-slate-700 font-medium rounded-lg hover:bg-slate-50 transition"
          >
            Sign in
          </Link>
          <Link
            to="/register"
            className="px-6 py-3 bg-rose-500 text-white font-medium rounded-lg hover:bg-rose-600 transition"
          >
            Sign up
          </Link>
        </div>
      )}
    </div>
  )
}
