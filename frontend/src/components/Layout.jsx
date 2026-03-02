import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useNotifications } from '../context/NotificationsContext'

export function Layout({ children }) {
  const { user, logout } = useAuth()
  const { unreadCount } = useNotifications()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/')
  }

  return (
    <div className="min-h-screen">
      <nav className="bg-white border-b border-slate-200 shadow-sm">
        <div className="max-w-4xl mx-auto px-4 h-14 flex items-center justify-between">
          <Link to="/" className="text-xl font-semibold text-rose-600">
            Matcha
          </Link>
          <div className="flex items-center gap-4">
            {user ? (
              <>
                <Link to="/discover" className="text-slate-600 hover:text-rose-600 transition">
                  Discover
                </Link>
                <Link to="/matches" className="text-slate-600 hover:text-rose-600 transition">
                  Matches
                </Link>
                <Link to="/views" className="text-slate-600 hover:text-rose-600 transition">
                  Views
                </Link>
                <Link to="/notifications" className="text-slate-600 hover:text-rose-600 transition relative">
                  Notifications
                  {unreadCount > 0 && (
                    <span className="ml-2 inline-flex items-center justify-center min-w-[1.25rem] h-5 px-1 rounded-full bg-rose-500 text-white text-xs font-semibold">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  )}
                </Link>
                <Link to="/profile" className="text-slate-600 hover:text-rose-600 transition">
                  Profile
                </Link>
                <span className="text-slate-500 text-sm">{user.username}</span>
                <button
                  onClick={handleLogout}
                  className="text-slate-600 hover:text-rose-600 transition text-sm"
                >
                  Logout
                </button>
              </>
            ) : (
              <>
                <Link to="/login" className="text-slate-600 hover:text-rose-600 transition">
                  Login
                </Link>
                <Link
                  to="/register"
                  className="bg-rose-500 text-white px-4 py-2 rounded-lg hover:bg-rose-600 transition"
                >
                  Sign up
                </Link>
              </>
            )}
          </div>
        </div>
      </nav>
      <main className="max-w-4xl mx-auto px-4 py-8">{children}</main>
    </div>
  )
}
