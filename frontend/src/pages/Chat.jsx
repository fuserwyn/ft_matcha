import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { chat, presence, users, wsChatUrl } from '../api/client'
import { useAuth } from '../context/AuthContext'

function formatDate(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function friendlyCallError(err) {
  const msg = String(err?.message || '').toLowerCase()
  const name = String(err?.name || '')
  if (msg.includes('load failed')) {
    return 'Cannot start camera stream on this device. Check camera/mic permissions and retry.'
  }
  if (name === 'NotAllowedError' || msg.includes('permission') || msg.includes('not allowed')) {
    return 'Camera/microphone permission denied. Allow access in browser settings.'
  }
  if (name === 'NotFoundError' || msg.includes('requested device not found')) {
    return 'No camera or microphone found on this device.'
  }
  if (name === 'NotReadableError' || msg.includes('could not start video source')) {
    return 'Camera is busy in another app. Close other apps using camera and retry.'
  }
  if (msg.includes('https') || msg.includes('secure context')) {
    return 'Video calls on mobile require HTTPS (or localhost).'
  }
  return err?.message || 'Failed to start call'
}

function createCallId() {
  const maybeRandomUUID = globalThis?.crypto?.randomUUID
  if (typeof maybeRandomUUID === 'function') return maybeRandomUUID.call(globalThis.crypto)
  return `call-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export default function Chat() {
  const { id: otherUserId } = useParams()
  const { user } = useAuth()
  const location = useLocation()

  const [profile, setProfile] = useState(null)
  const [presenceState, setPresenceState] = useState({ is_online: false, last_seen: null })
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [connected, setConnected] = useState(false)
  const [callState, setCallState] = useState('idle')
  const [incomingOffer, setIncomingOffer] = useState(null)
  const [callMode, setCallMode] = useState('video')
  const [isRecordingVoice, setIsRecordingVoice] = useState(false)

  const wsRef = useRef(null)
  const listEndRef = useRef(null)
  const pcRef = useRef(null)
  const localStreamRef = useRef(null)
  const remoteStreamRef = useRef(null)
  const localVideoRef = useRef(null)
  const remoteVideoRef = useRef(null)
  const activeCallIdRef = useRef('')
  const callStateRef = useRef('idle')
  const activeCallModeRef = useRef('video')
  const mediaRecorderRef = useRef(null)
  const mediaChunksRef = useRef([])

  useEffect(() => {
    listEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    callStateRef.current = callState
  }, [callState])

  // If navigated here from GlobalCallBanner (user accepted call from another page),
  // pre-populate the incoming offer so the in-chat banner appears immediately.
  useEffect(() => {
    const pending = location.state?.pendingCall
    if (pending) {
      setIncomingOffer(pending)
      setCallState('incoming')
      // Clear the state so a page refresh doesn't re-trigger it
      window.history.replaceState({}, '', window.location.href)
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const sendWsEvent = (payload) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket is offline')
    }
    wsRef.current.send(JSON.stringify(payload))
  }

  const stopVideoCall = () => {
    if (pcRef.current) {
      pcRef.current.onicecandidate = null
      pcRef.current.ontrack = null
      pcRef.current.onconnectionstatechange = null
      pcRef.current.close()
      pcRef.current = null
    }
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach((t) => t.stop())
      localStreamRef.current = null
    }
    if (remoteStreamRef.current) {
      remoteStreamRef.current.getTracks().forEach((t) => t.stop())
      remoteStreamRef.current = null
    }
    if (localVideoRef.current) localVideoRef.current.srcObject = null
    if (remoteVideoRef.current) remoteVideoRef.current.srcObject = null
    activeCallIdRef.current = ''
    activeCallModeRef.current = 'video'
    setIncomingOffer(null)
    setCallMode('video')
    setCallState('idle')
  }

  const stopVoiceRecorderTracks = () => {
    const rec = mediaRecorderRef.current
    if (!rec || !rec.stream) return
    rec.stream.getTracks().forEach((t) => t.stop())
  }

  useEffect(() => {
    return () => {
      stopVideoCall()
      stopVoiceRecorderTracks()
    }
  }, [])

  const setupPeerConnection = (callId) => {
    const pc = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
    })
    pc.onicecandidate = (event) => {
      if (!event.candidate || !activeCallIdRef.current) return
      try {
        sendWsEvent({
          type: 'call_ice',
          to_user_id: otherUserId,
          call_id: activeCallIdRef.current,
          candidate: event.candidate,
        })
      } catch {
        // ignore transient websocket errors
      }
    }
    pc.ontrack = (event) => {
      const [stream] = event.streams
      if (stream && remoteVideoRef.current) {
        remoteStreamRef.current = stream
        remoteVideoRef.current.srcObject = stream
      }
    }
    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'connected') {
        setCallState('in_call')
      }
      if (['failed', 'disconnected', 'closed'].includes(pc.connectionState)) {
        stopVideoCall()
      }
    }
    pcRef.current = pc
    activeCallIdRef.current = callId
    return pc
  }

  const getLocalMedia = async (mode) => {
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      throw new Error('This browser does not support camera/microphone access')
    }
    if (!window.isSecureContext && window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
      throw new Error('Video calls on mobile require HTTPS (or localhost)')
    }
    const constraints = mode === 'audio' ? { video: false, audio: true } : { video: true, audio: true }
    const stream = await navigator.mediaDevices.getUserMedia(constraints)
    localStreamRef.current = stream
    if (localVideoRef.current) localVideoRef.current.srcObject = stream
    return stream
  }

  const startCall = async (mode) => {
    setError('')
    if (!connected) {
      setError('WebSocket offline. Reconnect and try again.')
      return
    }
    if (callState !== 'idle') return
    try {
      setCallState('calling')
      setCallMode(mode)
      activeCallModeRef.current = mode
      const callId = createCallId()
      const stream = await getLocalMedia(mode)
      const pc = setupPeerConnection(callId)
      stream.getTracks().forEach((track) => pc.addTrack(track, stream))
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)
      sendWsEvent({
        type: 'call_invite',
        to_user_id: otherUserId,
        call_id: callId,
        mode,
        sdp: offer.sdp,
      })
    } catch (err) {
      stopVideoCall()
      setError(friendlyCallError(err))
    }
  }

  const startVideoCall = async () => startCall('video')
  const startVoiceCall = async () => startCall('audio')

  const startVoiceRecording = async () => {
    setError('')
    try {
      if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
        throw new Error('This browser does not support audio recording')
      }
      if (!window.isSecureContext && window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
        throw new Error('Voice recording on mobile requires HTTPS (or localhost)')
      }
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false })
      const recorder = new MediaRecorder(stream)
      mediaChunksRef.current = []
      recorder.ondataavailable = (event) => {
        if (event.data && event.data.size > 0) {
          mediaChunksRef.current.push(event.data)
        }
      }
      recorder.onstop = async () => {
        const blob = new Blob(mediaChunksRef.current, { type: recorder.mimeType || 'audio/webm' })
        stopVoiceRecorderTracks()
        mediaRecorderRef.current = null
        mediaChunksRef.current = []
        setIsRecordingVoice(false)
        if (blob.size === 0) return
        try {
          const ext = blob.type.includes('mp4') ? 'm4a' : blob.type.includes('mpeg') ? 'mp3' : blob.type.includes('ogg') ? 'ogg' : blob.type.includes('wav') ? 'wav' : 'webm'
          const file = new File([blob], `voice-${Date.now()}.${ext}`, { type: blob.type || 'audio/webm' })
          const created = await chat.sendVoiceMessage(otherUserId, file)
          setMessages((prev) => (prev.some((x) => x.id === created.id) ? prev : [...prev, created]))
        } catch (err) {
          setError(err.message || 'Failed to send voice message')
        }
      }
      mediaRecorderRef.current = recorder
      recorder.start()
      setIsRecordingVoice(true)
    } catch (err) {
      setError(friendlyCallError(err))
      setIsRecordingVoice(false)
      stopVoiceRecorderTracks()
    }
  }

  const stopVoiceRecording = () => {
    const rec = mediaRecorderRef.current
    if (!rec || rec.state !== 'recording') return
    rec.stop()
  }

  const acceptVideoCall = async () => {
    if (!incomingOffer) return
    setError('')
    try {
      setCallState('connecting')
      const mode = incomingOffer.mode || 'video'
      setCallMode(mode)
      activeCallModeRef.current = mode
      const stream = await getLocalMedia(mode)
      const pc = setupPeerConnection(incomingOffer.call_id)
      stream.getTracks().forEach((track) => pc.addTrack(track, stream))
      await pc.setRemoteDescription({ type: 'offer', sdp: incomingOffer.sdp })
      const answer = await pc.createAnswer()
      await pc.setLocalDescription(answer)
      sendWsEvent({
        type: 'call_accept',
        to_user_id: otherUserId,
        call_id: incomingOffer.call_id,
        mode,
        sdp: answer.sdp,
      })
      setIncomingOffer(null)
    } catch (err) {
      stopVideoCall()
      setError(friendlyCallError(err))
    }
  }

  const rejectVideoCall = () => {
    if (!incomingOffer) return
    try {
      sendWsEvent({
        type: 'call_reject',
        to_user_id: otherUserId,
        call_id: incomingOffer.call_id,
      })
    } catch {
      // ignore websocket errors
    }
    setIncomingOffer(null)
    setCallState('idle')
  }

  const endVideoCall = () => {
    try {
      if (activeCallIdRef.current) {
        sendWsEvent({
          type: 'call_end',
          to_user_id: otherUserId,
          call_id: activeCallIdRef.current,
        })
      }
    } catch {
      // ignore websocket errors
    }
    stopVideoCall()
  }

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
    const wsUrl = wsChatUrl()
    if (!wsUrl) return undefined

    let active = true
    let ws = null
    let reconnectTimer = null

    const connect = () => {
      ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => setConnected(true)
      ws.onclose = () => {
        setConnected(false)
        if (active) {
          reconnectTimer = setTimeout(connect, 2000)
        }
      }
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
          } else if (payload.type === 'message_read' && payload.data) {
            const r = payload.data
            const readByOther = r.reader_id === otherUserId && r.sender_id === user?.id
            if (readByOther) {
              setMessages((prev) =>
                prev.map((m) => {
                  const shouldMark = m.sender_id === user?.id && m.receiver_id === otherUserId
                  return shouldMark ? { ...m, is_read: true, read_at: r.read_at || m.read_at } : m
                }),
              )
            }
          } else if (payload.type === 'error') {
            setError(payload.error || 'WebSocket error')
          } else if (payload.type === 'call_invite' && payload.data) {
            const d = payload.data
            if (d.from_user_id !== otherUserId) return
            if (callStateRef.current !== 'idle') {
              sendWsEvent({
                type: 'call_reject',
                to_user_id: otherUserId,
                call_id: d.call_id,
              })
              return
            }
            setIncomingOffer({ call_id: d.call_id, sdp: d.sdp, mode: d.mode || 'video' })
            setCallState('incoming')
          } else if (payload.type === 'call_accept' && payload.data) {
            const d = payload.data
            if (d.from_user_id !== otherUserId || !pcRef.current || !d.sdp) return
            if (activeCallIdRef.current && d.call_id !== activeCallIdRef.current) return
            setCallMode(d.mode || activeCallModeRef.current || 'video')
            activeCallModeRef.current = d.mode || activeCallModeRef.current || 'video'
            await pcRef.current.setRemoteDescription({ type: 'answer', sdp: d.sdp })
            setCallState('connecting')
          } else if (payload.type === 'call_ice' && payload.data) {
            const d = payload.data
            if (d.from_user_id !== otherUserId || !pcRef.current || !d.candidate) return
            if (activeCallIdRef.current && d.call_id !== activeCallIdRef.current) return
            try {
              await pcRef.current.addIceCandidate(d.candidate)
            } catch {
              // ignore race where remote description not set yet
            }
          } else if (payload.type === 'call_reject' && payload.data) {
            const d = payload.data
            if (d.from_user_id !== otherUserId) return
            setError('Call rejected')
            stopVideoCall()
          } else if (payload.type === 'call_end' && payload.data) {
            const d = payload.data
            if (d.from_user_id !== otherUserId) return
            stopVideoCall()
          }
        } catch {
          // ignore malformed events
        }
      }
    }

    connect()
    return () => {
      active = false
      if (reconnectTimer) clearTimeout(reconnectTimer)
      if (ws) ws.close()
    }
  }, [otherUserId, user?.id])

  useEffect(() => {
    const id = setInterval(async () => {
      try {
        const p = await presence.get(otherUserId)
        setPresenceState(p)
      } catch {
        // ignore poll errors
      }
    }, 15000)
    return () => clearInterval(id)
  }, [otherUserId])

  useEffect(() => {
    const id = setInterval(async () => {
      if (connected) return
      try {
        const msgs = await chat.listMessages(otherUserId)
        setMessages(msgs)
      } catch {
        // ignore poll errors
      }
    }, 5000)
    return () => clearInterval(id)
  }, [connected, otherUserId])

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

  // ─── helpers ────────────────────────────────────────────────────────────────

  const avatarUrl = profile?.primary_photo_url
  const displayName = profile ? `${profile.first_name} ${profile.last_name}` : '…'
  const initials = profile
    ? `${profile.first_name?.[0] ?? ''}${profile.last_name?.[0] ?? ''}`.toUpperCase()
    : '?'

  const inActiveCall = ['calling', 'connecting', 'in_call'].includes(callState)

  // ─── loading / fatal error ───────────────────────────────────────────────────

  if (loading) {
    return (
      <div className="flex justify-center py-16">
        <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
      </div>
    )
  }

  if (error && !profile) {
    return (
      <div className="text-center py-16">
        <p className="text-rose-600 mb-4">{error}</p>
        <Link to="/matches" className="text-rose-600 hover:underline font-medium">
          ← Back to matches
        </Link>
      </div>
    )
  }

  // ─── main chat UI ────────────────────────────────────────────────────────────

  return (
    <div className="max-w-2xl mx-auto flex flex-col -mt-6 sm:-mt-8 -mb-6 sm:-mb-8" style={{ height: 'calc(100vh - 3.5rem)' }}>

      {/* ── header ── */}
      <div className="bg-white border-b border-slate-200 px-4 py-3 flex items-center gap-3 shrink-0 shadow-sm sticky top-14 z-10">
        <Link
          to="/matches"
          className="text-slate-400 hover:text-slate-700 transition p-1 -ml-1 rounded-full hover:bg-slate-100"
          title="Back to matches"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
          </svg>
        </Link>

        {/* avatar */}
        {avatarUrl ? (
          <img
            src={avatarUrl}
            alt={displayName}
            className="w-10 h-10 rounded-full object-cover ring-2 ring-white shadow-sm shrink-0"
          />
        ) : (
          <div className="w-10 h-10 rounded-full bg-gradient-to-br from-rose-400 to-pink-500 flex items-center justify-center text-white font-semibold text-sm shrink-0">
            {initials}
          </div>
        )}

        {/* name + presence */}
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-slate-800 truncate leading-tight">{displayName}</p>
          <div className="flex items-center gap-1.5 mt-0.5">
            <span className={`w-2 h-2 rounded-full shrink-0 ${presenceState.is_online ? 'bg-emerald-500' : 'bg-slate-300'}`} />
            <span className="text-xs text-slate-500 truncate">
              {presenceState.is_online
                ? 'Online now'
                : presenceState.last_seen
                ? `Last seen ${new Date(presenceState.last_seen).toLocaleString()}`
                : 'Offline'}
            </span>
          </div>
        </div>

        {/* call action buttons */}
        <div className="flex items-center gap-2 shrink-0">
          <button
            type="button"
            onClick={startVoiceCall}
            disabled={!connected || callState !== 'idle'}
            title="Voice call"
            className="w-9 h-9 rounded-full flex items-center justify-center border border-slate-200 text-slate-600 hover:border-rose-300 hover:text-rose-500 hover:bg-rose-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
            </svg>
          </button>
          <button
            type="button"
            onClick={startVideoCall}
            disabled={!connected || callState !== 'idle'}
            title="Video call"
            className="w-9 h-9 rounded-full flex items-center justify-center border border-slate-200 text-slate-600 hover:border-rose-300 hover:text-rose-500 hover:bg-rose-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          </button>
        </div>
      </div>

      {/* ── incoming call banner ── */}
      {callState === 'incoming' && (
        <div className="shrink-0 mx-3 mt-3 bg-white border border-slate-200 rounded-2xl shadow-lg p-4 flex items-center gap-4">
          <div className="w-12 h-12 rounded-full bg-gradient-to-br from-indigo-400 to-violet-500 flex items-center justify-center text-white shrink-0 animate-pulse">
            {incomingOffer?.mode === 'audio' ? (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
            )}
          </div>
          <div className="flex-1 min-w-0">
            <p className="font-semibold text-slate-800 text-sm">Incoming {incomingOffer?.mode === 'audio' ? 'voice' : 'video'} call</p>
            <p className="text-xs text-slate-500">{displayName}</p>
          </div>
          <div className="flex gap-2 shrink-0">
            <button
              type="button"
              onClick={rejectVideoCall}
              className="w-10 h-10 rounded-full bg-rose-100 text-rose-600 hover:bg-rose-200 transition flex items-center justify-center"
              title="Decline"
            >
              <svg className="w-4 h-4 rotate-135" fill="currentColor" viewBox="0 0 24 24">
                <path d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
              </svg>
            </button>
            <button
              type="button"
              onClick={acceptVideoCall}
              className="w-10 h-10 rounded-full bg-emerald-500 text-white hover:bg-emerald-600 transition flex items-center justify-center"
              title="Accept"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </button>
          </div>
        </div>
      )}

      {/* ── video panel ──
           Outer div: padding-bottom trick gives 16:9 ratio in ALL browsers.
           padding-bottom % is relative to WIDTH, so height = 56.25% of width = 16:9.
           When idle: padding-bottom:0 + height:0 → collapses to nothing.
           Video refs always stay in DOM so WebRTC works. */}
      <div
        className={`shrink-0 ${inActiveCall ? 'mx-3 mt-3' : ''}`}
        style={{
          position: 'relative',
          height: 0,
          paddingBottom: inActiveCall ? '56.25%' : '0',
          overflow: 'hidden',
        }}
      >
        {/* Inner panel — absolutely fills outer, clips rounded corners */}
        <div
          className={`absolute inset-0 bg-black ${inActiveCall ? 'rounded-2xl overflow-hidden' : 'overflow-hidden'}`}
        >
          {/* remote video — object-cover fills edge-to-edge, no bars */}
          <video
            ref={remoteVideoRef}
            autoPlay
            playsInline
            style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
            className={callMode === 'audio' ? 'hidden' : ''}
          />

          {/* audio-only placeholder */}
          {callMode === 'audio' && inActiveCall && (
            <div className="absolute inset-0 flex flex-col items-center justify-center gap-3">
              {avatarUrl ? (
                <img src={avatarUrl} alt={displayName} className="w-24 h-24 rounded-full object-cover ring-4 ring-white/20" />
              ) : (
                <div className="w-24 h-24 rounded-full bg-gradient-to-br from-rose-400 to-pink-500 flex items-center justify-center text-white text-3xl font-bold">
                  {initials}
                </div>
              )}
              <p className="text-white font-semibold">{displayName}</p>
              <p className="text-white/60 text-sm">
                {callState === 'calling' ? 'Calling…' : callState === 'connecting' ? 'Connecting…' : 'Voice call'}
              </p>
            </div>
          )}

          {/* local video PiP — padding-bottom trick for 3:4 ratio, cross-browser */}
          <div
            style={{
              position: 'absolute',
              bottom: '12px',
              right: '12px',
              width: '22%',
              height: 0,
              paddingBottom: '29.33%',  /* 22% * (4/3) = 29.33% of panel width */
              borderRadius: '16px',
              overflow: 'hidden',
              boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
              border: '2px solid rgba(255,255,255,0.2)',
              background: '#000',
            }}
          >
            <video
              ref={localVideoRef}
              autoPlay
              playsInline
              muted
              style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
              className={callMode === 'audio' ? 'hidden' : ''}
            />
            {callMode === 'audio' && inActiveCall && (
              <div className="absolute inset-0 flex items-center justify-center text-white/40 text-xs">You</div>
            )}
          </div>

          {/* end-call button — bottom-center, z-10 keeps it above the connecting overlay */}
          {inActiveCall && (
            <div className="absolute bottom-4 left-1/2 -translate-x-1/2 z-10">
              <button
                type="button"
                onClick={endVideoCall}
                className="w-14 h-14 rounded-full bg-rose-500 hover:bg-rose-600 text-white flex items-center justify-center shadow-xl transition"
                title="End call"
              >
                <svg className="w-6 h-6 rotate-135" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
                </svg>
              </button>
            </div>
          )}

          {/* connecting overlay */}
          {(callState === 'calling' || callState === 'connecting') && (
            <div className="absolute inset-0 bg-black/50 flex items-center justify-center">
              <div className="text-center">
                <div className="w-8 h-8 border-2 border-white border-t-transparent rounded-full animate-spin mx-auto mb-2" />
                <p className="text-white text-sm">{callState === 'calling' ? 'Calling…' : 'Connecting…'}</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ── error banner ── */}
      {error && (
        <div className="shrink-0 mx-3 mt-3 flex items-center gap-2 bg-rose-50 border border-rose-200 text-rose-700 text-sm rounded-xl px-4 py-2.5">
          <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span className="flex-1">{error}</span>
          <button type="button" onClick={() => setError('')} className="text-rose-400 hover:text-rose-600 transition">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      )}

      {/* ── messages ── */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-3 min-h-0">
        {messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full gap-3 text-slate-400">
            <div className="w-14 h-14 rounded-full bg-slate-100 flex items-center justify-center">
              <svg className="w-7 h-7" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
              </svg>
            </div>
            <p className="text-sm">Say hello to {profile?.first_name}!</p>
          </div>
        ) : (
          <>
            {messages.map((m) => {
              const mine = m.sender_id === user?.id
              const isVoice = m.message_type === 'voice' && m.media_url
              return (
                <div key={m.id} className={`flex items-end gap-2 ${mine ? 'flex-row-reverse' : 'flex-row'}`}>
                  {/* avatar for received messages */}
                  {!mine && (
                    avatarUrl ? (
                      <img src={avatarUrl} alt="" className="w-7 h-7 rounded-full object-cover shrink-0 mb-0.5" />
                    ) : (
                      <div className="w-7 h-7 rounded-full bg-gradient-to-br from-rose-400 to-pink-500 flex items-center justify-center text-white text-xs font-semibold shrink-0 mb-0.5">
                        {initials}
                      </div>
                    )
                  )}
                  <div className={`max-w-[72%] ${mine ? 'items-end' : 'items-start'} flex flex-col gap-0.5`}>
                    <div className={`rounded-2xl px-3.5 py-2.5 ${
                      mine
                        ? 'bg-rose-500 text-white rounded-br-sm'
                        : 'bg-white border border-slate-200 text-slate-800 rounded-bl-sm shadow-sm'
                    }`}>
                      {isVoice ? (
                        <div className="flex items-center gap-2 py-0.5">
                          <svg className={`w-4 h-4 shrink-0 ${mine ? 'text-rose-200' : 'text-rose-500'}`} fill="currentColor" viewBox="0 0 24 24">
                            <path d="M12 1a3 3 0 00-3 3v8a3 3 0 006 0V4a3 3 0 00-3-3z"/>
                            <path d="M19 10v2a7 7 0 01-14 0v-2H3v2a9 9 0 008 8.94V23h2v-2.06A9 9 0 0021 12v-2h-2z"/>
                          </svg>
                          <audio controls preload="none" src={m.media_url} className="h-8 max-w-[180px]" style={{ filter: mine ? 'invert(1) brightness(1.8)' : 'none' }} />
                        </div>
                      ) : (
                        <p className="text-sm whitespace-pre-wrap leading-relaxed">{m.content}</p>
                      )}
                    </div>
                    <div className={`flex items-center gap-1 px-1 ${mine ? 'flex-row-reverse' : ''}`}>
                      <span className="text-[10px] text-slate-400">{formatDate(m.created_at)}</span>
                      {mine && (
                        <span className={`text-[10px] ${m.is_read ? 'text-rose-400' : 'text-slate-400'}`}>
                          {m.is_read ? '✓✓' : '✓'}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              )
            })}
            <div ref={listEndRef} />
          </>
        )}
      </div>

      {/* ── input bar ── */}
      <div className="shrink-0 bg-white border-t border-slate-200 px-3 py-3">
        {isRecordingVoice ? (
          /* recording state */
          <div className="flex items-center gap-3 bg-rose-50 border border-rose-200 rounded-2xl px-4 py-3">
            <span className="flex items-center gap-2 flex-1 text-rose-700 text-sm font-medium">
              <span className="w-2.5 h-2.5 rounded-full bg-rose-500 animate-pulse" />
              Recording…
            </span>
            <button
              type="button"
              onClick={stopVoiceRecording}
              className="flex items-center gap-1.5 px-4 py-2 bg-rose-500 text-white text-sm font-medium rounded-xl hover:bg-rose-600 transition"
            >
              <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                <rect x="6" y="6" width="12" height="12" rx="1" />
              </svg>
              Send
            </button>
          </div>
        ) : (
          /* normal input row */
          <form onSubmit={send} className="flex items-end gap-2">
            {/* mic button */}
            <button
              type="button"
              onClick={startVoiceRecording}
              title="Record voice message"
              className="w-10 h-10 rounded-full flex items-center justify-center border border-slate-200 text-slate-500 hover:border-rose-300 hover:text-rose-500 hover:bg-rose-50 transition shrink-0"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 1a3 3 0 00-3 3v8a3 3 0 006 0V4a3 3 0 00-3-3z"/>
                <path d="M19 10v2a7 7 0 01-14 0v-2H3v2a9 9 0 008 8.94V23h2v-2.06A9 9 0 0021 12v-2h-2z"/>
              </svg>
            </button>
            {/* text input */}
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              maxLength={2000}
              placeholder={`Message ${profile?.first_name ?? ''}…`}
              className="flex-1 bg-slate-50 border border-slate-200 rounded-2xl px-4 py-2.5 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-rose-400 focus:border-transparent transition resize-none"
            />
            {/* send button */}
            <button
              type="submit"
              disabled={!input.trim()}
              className="w-10 h-10 rounded-full flex items-center justify-center bg-rose-500 text-white hover:bg-rose-600 disabled:opacity-40 disabled:cursor-not-allowed transition shrink-0"
              title="Send"
            >
              <svg className="w-4 h-4 -translate-x-px" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            </button>
          </form>
        )}
      </div>

    </div>
  )
}
