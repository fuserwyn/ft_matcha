import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { notifications, wsChatUrl } from '../api/client'
import { useAuth } from './AuthContext'

const NotificationsContext = createContext(null)

export function NotificationsProvider({ children }) {
  const { user } = useAuth()
  const [unreadCount, setUnreadCount] = useState(0)

  const refreshUnread = useCallback(async () => {
    if (!user) {
      setUnreadCount(0)
      return
    }
    try {
      const data = await notifications.list({ unread_only: true, limit: 100 })
      setUnreadCount(Array.isArray(data) ? data.length : 0)
    } catch {
      // keep previous count on transient errors
    }
  }, [user])

  useEffect(() => {
    if (!user) {
      setUnreadCount(0)
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
          // ignore malformed payloads
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
      refreshUnread,
      setUnreadCount,
    }),
    [unreadCount, refreshUnread],
  )

  return <NotificationsContext.Provider value={value}>{children}</NotificationsContext.Provider>
}

export function useNotifications() {
  const ctx = useContext(NotificationsContext)
  if (!ctx) throw new Error('useNotifications must be inside NotificationsProvider')
  return ctx
}
