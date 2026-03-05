import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useNotifications } from '../context/NotificationsContext'

export function Layout({ children }) {
  const { user, logout } = useAuth()
  const { unreadCount } = useNotifications()
  const navigate = useNavigate()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  const handleLogout = () => {
    logout()
    navigate('/')
    setMobileMenuOpen(false)
  }

  const closeMobileMenu = () => setMobileMenuOpen(false)

  const navLinks = user ? (
    <>
      <Link to="/discover" className="text-slate-600 hover:text-rose-600 transition py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center" onClick={closeMobileMenu}>
        Discover
      </Link>
      <Link to="/matches" className="text-slate-600 hover:text-rose-600 transition py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center" onClick={closeMobileMenu}>
        Matches
      </Link>
      <Link to="/likes" className="text-slate-600 hover:text-rose-600 transition py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center" onClick={closeMobileMenu}>
        Likes
      </Link>
      <Link to="/views" className="text-slate-600 hover:text-rose-600 transition py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center" onClick={closeMobileMenu}>
        Views
      </Link>
      <Link to="/notifications" className="text-slate-600 hover:text-rose-600 transition relative py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center" onClick={closeMobileMenu}>
        Notifications
        {unreadCount > 0 && (
          <span className="ml-2 inline-flex items-center justify-center min-w-[1.25rem] h-5 px-1 rounded-full bg-rose-500 text-white text-xs font-semibold">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </Link>
      <Link to="/profile" className="text-slate-600 hover:text-rose-600 transition py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center" onClick={closeMobileMenu}>
        Profile
      </Link>
      <span className="text-slate-500 text-sm py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center truncate max-w-[120px] lg:max-w-none">{user.username}</span>
      <button onClick={handleLogout} className="text-slate-600 hover:text-rose-600 transition text-sm text-left py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center">
        Logout
      </button>
    </>
  ) : (
    <>
      <Link to="/login" className="text-slate-600 hover:text-rose-600 transition py-3 lg:py-0 min-h-[44px] lg:min-h-0 flex items-center" onClick={closeMobileMenu}>
        Login
      </Link>
      <Link to="/register" className="bg-rose-500 text-white px-4 py-3 rounded-lg hover:bg-rose-600 transition block text-center lg:inline-block min-h-[44px] lg:min-h-0 flex items-center justify-center" onClick={closeMobileMenu}>
        Sign up
      </Link>
    </>
  )

  return (
    <div className="min-h-screen flex flex-col">
      <nav className="sticky top-0 z-20 bg-white border-b border-slate-200 shadow-sm pl-[env(safe-area-inset-left)] pr-[env(safe-area-inset-right)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between min-w-0">
          <Link to="/" className="text-xl font-semibold text-rose-600">
            Matcha
          </Link>
          <button
            type="button"
            className="lg:hidden p-3 -mr-2 rounded-lg text-slate-600 hover:bg-slate-100 min-h-[44px] min-w-[44px] flex items-center justify-center"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            aria-label="Toggle menu"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              {mobileMenuOpen ? (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              ) : (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              )}
            </svg>
          </button>
          <div className={`absolute top-14 left-0 right-0 bg-white border-b border-slate-200 shadow-lg z-10 lg:shadow-none lg:border-0 lg:static lg:flex lg:flex-row lg:items-center lg:gap-4 ${mobileMenuOpen ? 'flex flex-col p-4 gap-1' : 'hidden lg:flex'}`}>
            {navLinks}
          </div>
        </div>
      </nav>
      <main className="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 py-6 sm:py-8 min-w-0">{children}</main>
      <footer className="bg-white border-t border-slate-200 mt-auto pl-[env(safe-area-inset-left)] pr-[env(safe-area-inset-right)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 py-4 flex flex-col sm:flex-row items-center justify-between gap-2 text-sm text-slate-500">
          <span className="font-medium text-rose-600">Matcha</span>
          <span>Because love, too, can be industrialized.</span>
          <span>© {new Date().getFullYear()}</span>
        </div>
      </footer>
    </div>
  )
}
