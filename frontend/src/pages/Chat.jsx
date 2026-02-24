import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { chat, presence, users, wsChatUrl } from '../api/client'
import { useAuth } from '../context/AuthContext'

function formatDate(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export default function Chat() {
  const { id: otherUserId } = useParams()
  const { user } = useAuth()

  const [profile, setProfile] = useState(null)
  const [presenceState, setPresenceState] = useState({ is_online: false, last_seen: null })
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [connected, setConnected] = useState(false)

  const wsRef = useRef(null)
  const listEndRef = useRef(null)

  const wsUrl = useMemo(() => wsChatUrl(), [])

  useEffect(() => {
    listEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    let active = true

    ;(async () => {
      try {
        const [u, msgs, p] = await Promise.all([
          users.getById(otherUserId),
          chat.listMessages(otherUserId),
          presence.get(otherUserId),
        ])
        if (!active) return
        setProfile(u)
        setMessages(msgs)
        setPresenceState(p)
        setError('')
        await chat.markRead(otherUserId)
      } catch (err) {
        if (active) setError(err.message || 'Failed to load chat')
      } finally {
        if (active) setLoading(false)
      }
    })()

    return () => {
      active = false
    }
  }, [otherUserId])

  useEffect(() => {
    if (!wsUrl) return undefined

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)
    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)
    ws.onmessage = async (event) => {
      try {
        const payload = JSON.parse(event.data)
        if (payload.type === 'message' && payload.data) {
          const m = payload.data
          const isCurrentConversation =
            (m.sender_id === otherUserId && m.receiver_id === user?.id) ||
            (m.sender_id === user?.id && m.receiver_id === otherUserId)

          if (isCurrentConversation) {
            setMessages((prev) => {
              if (prev.some((x) => x.id === m.id)) return prev
              return [...prev, m]
            })
            if (m.sender_id === otherUserId) {
              await chat.markRead(otherUserId)
            }
          }
        } else if (payload.type === 'error') {
          setError(payload.error || 'WebSocket error')
        }
      } catch {
        // ignore malformed events
      }
    }

    return () => {
      ws.close()
    }
  }, [wsUrl, otherUserId, user?.id])

  const send = async (e) => {
    e.preventDefault()
    const content = input.trim()
    if (!content) return

    setError('')
    try {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            to_user_id: otherUserId,
            content,
          }),
        )
      } else {
        await chat.sendMessage(otherUserId, content)
        const refreshed = await chat.listMessages(otherUserId)
        setMessages(refreshed)
      }
      setInput('')
    } catch (err) {
      setError(err.message || 'Failed to send message')
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
      </div>
    )
  }

  if (error && !profile) {
    return (
      <div className="text-center py-12">
        <p className="text-rose-600 mb-3">{error}</p>
        <Link to="/matches" className="text-rose-600 hover:underline">
          Back to matches
        </Link>
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">
            Chat with {profile?.first_name} {profile?.last_name}
          </h1>
          <p className="text-sm text-slate-500">
            {presenceState.is_online
              ? 'Online now'
              : `Last seen: ${presenceState.last_seen ? new Date(presenceState.last_seen).toLocaleString() : 'unknown'}`}
          </p>
        </div>
        <span
          className={`text-xs px-2 py-1 rounded ${connected ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-600'}`}
        >
          {connected ? 'WS connected' : 'WS offline'}
        </span>
      </div>

      <div className="bg-white border border-slate-200 rounded-lg p-4 h-[420px] overflow-y-auto">
        {messages.length === 0 ? (
          <p className="text-slate-500 text-sm">No messages yet.</p>
        ) : (
          <div className="space-y-3">
            {messages.map((m) => {
              const mine = m.sender_id === user?.id
              return (
                <div key={m.id} className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-[75%] rounded-lg px-3 py-2 ${mine ? 'bg-rose-500 text-white' : 'bg-slate-100 text-slate-800'}`}>
                    <p className="text-sm whitespace-pre-wrap">{m.content}</p>
                    <p className={`text-[10px] mt-1 ${mine ? 'text-rose-100' : 'text-slate-500'}`}>
                      {formatDate(m.created_at)}
                      {mine ? ` • ${m.is_read ? 'read' : 'sent'}` : ''}
                    </p>
                  </div>
                </div>
              )
            })}
            <div ref={listEndRef} />
          </div>
        )}
      </div>

      <form onSubmit={send} className="mt-4 flex gap-2">
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          maxLength={2000}
          placeholder="Type your message..."
          className="flex-1 px-4 py-2 rounded border border-slate-300 focus:ring-2 focus:ring-rose-500 focus:border-transparent outline-none"
        />
        <button type="submit" className="px-4 py-2 bg-rose-500 text-white rounded hover:bg-rose-600">
          Send
        </button>
      </form>
      {error && <p className="text-rose-600 text-sm mt-3">{error}</p>}
    </div>
  )
}
