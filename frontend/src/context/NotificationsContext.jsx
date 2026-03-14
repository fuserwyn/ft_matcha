import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { notifications, wsChatUrl } from '../api/client'
import { useAuth } from './AuthContext'

const NotificationsContext = createContext(null)

export function NotificationsProvider({ children }) {
  const { user } = useAuth()
  const [unreadCount, setUnreadCount] = useState(0)
  const [likesCount, setLikesCount] = useState(0)
  const [matchesCount, setMatchesCount] = useState(0)
  const [viewsCount, setViewsCount] = useState(0)

  const refreshUnread = useCallback(async () => {
    if (!user) {
      setUnreadCount(0)
      setLikesCount(0)
      setMatchesCount(0)
      setViewsCount(0)
      return
    }
    try {
      const data = await notifications.list({ unread_only: true, limit: 100 })
      if (!Array.isArray(data)) return
      setUnreadCount(data.length)
      setLikesCount(data.filter((n) => n.type === 'like').length)
      setMatchesCount(data.filter((n) => n.type === 'match').length)
      setViewsCount(data.filter((n) => n.type === 'view').length)
    } catch {
    }
  }, [user])

  useEffect(() => {
    if (!user) {
      setUnreadCount(0)
      setLikesCount(0)
      setMatchesCount(0)
      setViewsCount(0)
      return undefined
    }

    let active = true
    const safeRefresh = async () => {
      if (!active) return
      await refreshUnread()
    }
    safeRefresh()

    const timer = setInterval(safeRefresh, 30000)
    const url = wsChatUrl()
    let ws = null
    if (url) {
      ws = new WebSocket(url)
      ws.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data)
          if (payload?.type === 'notification') {
            safeRefresh()
          }
        } catch {
        }
      }
    }

    return () => {
      active = false
      clearInterval(timer)
      if (ws) ws.close()
    }
  }, [user, refreshUnread])

  const value = useMemo(
    () => ({
      unreadCount,
      likesCount,
      matchesCount,
      viewsCount,
      refreshUnread,
      setUnreadCount,
    }),
    [unreadCount, likesCount, matchesCount, viewsCount, refreshUnread],
  )

  return <NotificationsContext.Provider value={value}>{children}</NotificationsContext.Provider>
}

export function useNotifications() {
  const ctx = useContext(NotificationsContext)
  if (!ctx) throw new Error('useNotifications must be inside NotificationsProvider')
  return ctx
}
