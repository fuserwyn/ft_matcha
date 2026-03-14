import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { users, wsChatUrl } from '../api/client'

export default function GlobalCallBanner() {
  const { user } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const locationRef = useRef(location.pathname)
  const wsRef = useRef(null)
  const [incoming, setIncoming] = useState(null)

  useEffect(() => {
    locationRef.current = location.pathname
  }, [location.pathname])

  useEffect(() => {
    if (incoming && location.pathname === `/chat/${incoming.fromUserId}`) {
      setIncoming(null)
    }
  }, [location.pathname, incoming])

  useEffect(() => {
    if (!user) return undefined
    const wsUrl = wsChatUrl()
    if (!wsUrl) return undefined

    let active = true
    let ws = null
    let reconnectTimer = null

    const connect = () => {
      ws = new WebSocket(wsUrl)
      wsRef.current = ws
      ws.onclose = () => {
        if (active) reconnectTimer = setTimeout(connect, 3000)
      }
      ws.onmessage = async (event) => {
        try {
          const payload = JSON.parse(event.data)

          if (payload.type === 'call_invite' && payload.data) {
            const d = payload.data
            if (d.from_user_id === user?.id) return
            if (locationRef.current === `/chat/${d.from_user_id}`) return

            let fromName = 'Someone'
            let fromPhoto = null
            try {
              const profile = await users.getById(d.from_user_id)
              fromName = `${profile.first_name} ${profile.last_name}`
              fromPhoto = profile.primary_photo_url
            } catch {}

            if (!active) return
            setIncoming({
              call_id: d.call_id,
              sdp: d.sdp,
              mode: d.mode || 'video',
              fromUserId: d.from_user_id,
              fromName,
              fromPhoto,
            })
          } else if (
            (payload.type === 'call_end' || payload.type === 'call_reject') &&
            payload.data
          ) {
            setIncoming((prev) =>
              prev?.call_id === payload.data.call_id ? null : prev
            )
          }
        } catch {
        }
      }
    }

    connect()
    return () => {
      active = false
      clearTimeout(reconnectTimer)
      ws?.close()
    }
  }, [user])

  if (!incoming) return null

  const accept = () => {
    const { call_id, sdp, mode, fromUserId } = incoming
    setIncoming(null)
    navigate(`/chat/${fromUserId}`, {
      state: { pendingCall: { call_id, sdp, mode } },
    })
  }

  const decline = () => {
    try {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: 'call_reject',
            to_user_id: incoming.fromUserId,
            call_id: incoming.call_id,
          })
        )
      }
    } catch {}
    setIncoming(null)
  }

  const initials = incoming.fromName
    .split(' ')
    .map((n) => n[0] ?? '')
    .join('')
    .toUpperCase()
    .slice(0, 2)

  return (
    <div className="fixed top-16 inset-x-0 z-50 flex justify-center px-4 pointer-events-none">
      <style>{`@keyframes bannerIn { from { opacity:0; transform:translateY(-12px) } to { opacity:1; transform:translateY(0) } }`}</style>
      <div
        className="bg-white border border-slate-200 rounded-2xl shadow-2xl p-4 flex items-center gap-4 w-full max-w-sm pointer-events-auto"
        style={{ animation: 'bannerIn 0.25s cubic-bezier(0.34,1.56,0.64,1) both' }}
      >
        <div className="relative shrink-0">
          {incoming.fromPhoto ? (
            <img
              src={incoming.fromPhoto}
              alt={incoming.fromName}
              className="w-12 h-12 rounded-full object-cover"
            />
          ) : (
            <div className="w-12 h-12 rounded-full bg-gradient-to-br from-rose-400 to-pink-500 flex items-center justify-center text-white font-semibold">
              {initials}
            </div>
          )}
          <span className="absolute inset-0 rounded-full border-2 border-rose-400 animate-ping opacity-60" />
        </div>

        <div className="flex-1 min-w-0">
          <p className="font-semibold text-slate-800 text-sm truncate">{incoming.fromName}</p>
          <p className="text-xs text-slate-500">
            Incoming {incoming.mode === 'audio' ? 'voice' : 'video'} call…
          </p>
        </div>

        <button
          type="button"
          onClick={decline}
          className="w-10 h-10 rounded-full bg-rose-100 text-rose-600 hover:bg-rose-200 transition flex items-center justify-center shrink-0"
          title="Decline"
        >
          <svg className="w-4 h-4 rotate-135" fill="currentColor" viewBox="0 0 24 24">
            <path d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
          </svg>
        </button>

        <button
          type="button"
          onClick={accept}
          className="w-10 h-10 rounded-full bg-emerald-500 text-white hover:bg-emerald-600 transition flex items-center justify-center shrink-0"
          title="Accept"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
          </svg>
        </button>
      </div>
    </div>
  )
}
