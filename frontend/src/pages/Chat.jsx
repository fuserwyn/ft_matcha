import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
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
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={startVideoCall}
          disabled={!connected || callState !== 'idle'}
          className="px-3 py-2 bg-indigo-600 text-white rounded disabled:opacity-50"
        >
          Start video call
        </button>
        <button
          type="button"
          onClick={startVoiceCall}
          disabled={!connected || callState !== 'idle'}
          className="px-3 py-2 bg-violet-600 text-white rounded disabled:opacity-50"
        >
          Start voice call
        </button>
        <button
          type="button"
          onClick={endVideoCall}
          disabled={!['calling', 'connecting', 'in_call'].includes(callState)}
          className="px-3 py-2 bg-slate-700 text-white rounded disabled:opacity-50"
        >
          End call
        </button>
        <span className="text-xs text-slate-600">
          Call status: {callState.replace('_', ' ')} ({callMode})
        </span>
      </div>
      {callState === 'incoming' && (
        <div className="mb-4 p-3 rounded border border-indigo-200 bg-indigo-50 flex items-center gap-2">
          <span className="text-sm text-indigo-900">Incoming {incomingOffer?.mode || 'video'} call</span>
          <button type="button" onClick={acceptVideoCall} className="px-3 py-1 bg-emerald-600 text-white rounded">
            Accept
          </button>
          <button type="button" onClick={rejectVideoCall} className="px-3 py-1 bg-rose-600 text-white rounded">
            Reject
          </button>
        </div>
      )}
      <div className="mb-4 grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="bg-black rounded overflow-hidden h-44">
          <video
            ref={localVideoRef}
            autoPlay
            playsInline
            muted
            className={`w-full h-full object-cover ${callMode === 'audio' ? 'hidden' : ''}`}
          />
          {callMode === 'audio' && <div className="w-full h-full flex items-center justify-center text-slate-200 text-sm">Audio call</div>}
        </div>
        <div className="bg-black rounded overflow-hidden h-44">
          <video
            ref={remoteVideoRef}
            autoPlay
            playsInline
            className={`w-full h-full object-cover ${callMode === 'audio' ? 'hidden' : ''}`}
          />
          {callMode === 'audio' && <div className="w-full h-full flex items-center justify-center text-slate-200 text-sm">Connected by voice</div>}
        </div>
      </div>

      <div className="bg-white border border-slate-200 rounded-lg p-4 h-[420px] overflow-y-auto">
        {messages.length === 0 ? (
          <p className="text-slate-500 text-sm">No messages yet.</p>
        ) : (
          <div className="space-y-3" key={user?.id}>
            {messages.map((m) => {
              const mine = m.sender_id === user?.id
              const isVoice = m.message_type === 'voice' && m.media_url
              return (
                <div key={m.id} className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-[75%] rounded-lg px-3 py-2 ${mine ? 'bg-rose-500 text-white' : 'bg-slate-100 text-slate-800'}`}>
                    {isVoice ? (
                      <audio controls preload="none" src={m.media_url} className="max-w-full" />
                    ) : (
                      <p className="text-sm whitespace-pre-wrap">{m.content}</p>
                    )}
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
      <div className="mt-2 flex gap-2">
        {!isRecordingVoice ? (
          <button
            type="button"
            onClick={startVoiceRecording}
            className="px-3 py-2 bg-violet-600 text-white rounded"
          >
            Record voice
          </button>
        ) : (
          <button
            type="button"
            onClick={stopVoiceRecording}
            className="px-3 py-2 bg-amber-600 text-white rounded"
          >
            Stop and send
          </button>
        )}
      </div>
      {error && <p className="text-rose-600 text-sm mt-3">{error}</p>}
    </div>
  )
}
