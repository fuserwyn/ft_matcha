import { useState, useEffect, useCallback, useRef } from 'react'
import { Link } from 'react-router-dom'
import { presence, users, photos } from '../api/client'

export default function ProfileModal({ userId, onClose }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [isMatch, setIsMatch] = useState(false)
  const [likedMe, setLikedMe] = useState(false)
  const [iLiked, setILiked] = useState(false)
  const [presenceState, setPresenceState] = useState(null)
  const [liking, setLiking] = useState(false)
  const [blocking, setBlocking] = useState(false)
  const [blocked, setBlocked] = useState(false)
  const [reporting, setReporting] = useState(false)
  const [reportModal, setReportModal] = useState(false)
  const [selectedReportReason, setSelectedReportReason] = useState('')
  const [hasPrimaryPhoto, setHasPrimaryPhoto] = useState(false)
  const [lightboxIndex, setLightboxIndex] = useState(null)
  const touchStartX = useRef(null)

  const allPhotos = user?.photos || []
  const isLightboxOpen = lightboxIndex !== null

  const openLightbox = useCallback((idx) => setLightboxIndex(idx), [])
  const closeLightbox = useCallback(() => setLightboxIndex(null), [])
  const prevPhoto = useCallback(() =>
    setLightboxIndex((i) => (i - 1 + allPhotos.length) % allPhotos.length),
  [allPhotos.length])
  const nextPhoto = useCallback(() =>
    setLightboxIndex((i) => (i + 1) % allPhotos.length),
  [allPhotos.length])

  useEffect(() => {
    const onKey = (e) => {
      if (isLightboxOpen) {
        if (e.key === 'Escape') closeLightbox()
        if (e.key === 'ArrowLeft') prevPhoto()
        if (e.key === 'ArrowRight') nextPhoto()
      } else {
        if (e.key === 'Escape') onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isLightboxOpen, closeLightbox, prevPhoto, nextPhoto, onClose])

  useEffect(() => {
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = prev }
  }, [])

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const [u, p, myPhotos] = await Promise.all([
          users.getById(userId),
          presence.get(userId),
          photos.listMe().catch(() => []),
        ])
        if (!active) return
        setUser(u)
        setIsMatch(Boolean(u.is_match))
        setLikedMe(Boolean(u.liked_me))
        setILiked(Boolean(u.i_liked))
        setPresenceState(p)
        setHasPrimaryPhoto(Array.isArray(myPhotos) && myPhotos.some((ph) => ph.is_primary))
      } catch {
        if (active) setError('User not found')
      } finally {
        if (active) setLoading(false)
      }
    })()
    return () => { active = false }
  }, [userId])

  const age = (birthDate) => {
    if (!birthDate) return null
    const diff = Date.now() - new Date(birthDate).getTime()
    return Math.floor(diff / (365.25 * 24 * 60 * 60 * 1000))
  }

  const onLike = async () => {
    if (blocked || !hasPrimaryPhoto) return
    setLiking(true); setInfo(''); setError('')
    try {
      const res = await users.like(userId)
      if (res?.is_match) { setIsMatch(true); setILiked(true); setInfo("It's a match!") }
      else { setILiked(true); setInfo('Liked') }
    } catch (err) { setError(err.message || 'Failed') }
    finally { setLiking(false) }
  }

  const onUnlike = async () => {
    if (blocked) return
    setLiking(true); setInfo(''); setError('')
    try { await users.unlike(userId); setILiked(false); setIsMatch(false); setInfo('Like removed') }
    catch (err) { setError(err.message || 'Failed') }
    finally { setLiking(false) }
  }

  const onReport = async (reason, shouldBlock) => {
    setReporting(true); setInfo(''); setError('')
    try {
      await users.report(userId, { reason, comment: null, block_user: shouldBlock && !blocked })
      if (shouldBlock && !blocked) {
        setBlocked(true)
      }
      setReportModal(false)
      setSelectedReportReason('')
      setInfo(shouldBlock ? 'Report submitted and user blocked.' : 'Report submitted. Thank you.')
    }
    catch (err) { setError(err.message || 'Failed') }
    finally { setReporting(false) }
  }

  const closeReportModal = () => {
    if (reporting) return
    setReportModal(false)
    setSelectedReportReason('')
  }

  const onToggleBlock = async () => {
    setBlocking(true); setInfo(''); setError('')
    try {
      if (blocked) { await users.unblock(userId); setBlocked(false); setInfo('User unblocked') }
      else { await users.block(userId); setBlocked(true); setInfo('User blocked') }
    } catch (err) { setError(err.message || 'Failed') }
    finally { setBlocking(false) }
  }

  const primaryIdx = allPhotos.findIndex((p) => p.is_primary)
  const primaryPhotoIdx = primaryIdx >= 0 ? primaryIdx : 0
  const primary = allPhotos[primaryPhotoIdx]
  const others = allPhotos.filter((_, i) => i !== primaryPhotoIdx)

  const handleTouchStart = (e) => { touchStartX.current = e.touches[0].clientX }
  const handleTouchEnd = (e) => {
    if (touchStartX.current === null) return
    const delta = e.changedTouches[0].clientX - touchStartX.current
    if (delta > 50) prevPhoto()
    else if (delta < -50) nextPhoto()
    touchStartX.current = null
  }

  return (
    <>
      <div
        className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6"
        style={{ background: 'rgba(15,15,20,0.65)', backdropFilter: 'blur(8px)' }}
        onClick={onClose}
      >
        <div
          className="relative bg-white rounded-3xl shadow-2xl w-full max-w-2xl max-h-[92vh] overflow-y-auto"
          style={{ animation: 'modalIn 0.22s cubic-bezier(0.34,1.56,0.64,1) both' }}
          onClick={(e) => e.stopPropagation()}
        >
          <button
            onClick={onClose}
            aria-label="Close"
            className="absolute top-3 right-3 z-20 w-9 h-9 flex items-center justify-center rounded-full bg-white shadow-md text-slate-400 hover:text-slate-700 text-2xl leading-none transition"
          >
            ×
          </button>

          {loading && (
            <div className="flex justify-center py-24">
              <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
            </div>
          )}

          {!loading && (error || !user) && (
            <div className="text-center py-24 text-rose-600">{error || 'User not found'}</div>
          )}

          {!loading && user && (
            <div className="md:flex">
              <div className="md:w-64 lg:w-72 shrink-0 border-b md:border-b-0 md:border-r border-slate-100">
                {primary ? (
                  <div
                    className="cursor-pointer group overflow-hidden rounded-t-3xl md:rounded-tr-none md:rounded-l-3xl"
                    onClick={() => openLightbox(primaryPhotoIdx)}
                  >
                    <img
                      src={primary.url}
                      alt={user.first_name}
                      className="w-full aspect-[3/4] object-cover object-top group-hover:scale-105 transition-transform duration-500"
                    />
                  </div>
                ) : (
                  <div className="aspect-[3/4] bg-gradient-to-br from-slate-100 to-slate-200 flex items-center justify-center rounded-t-3xl md:rounded-tr-none md:rounded-l-3xl">
                    <span className="text-8xl font-bold text-slate-300">
                      {(user.first_name?.[0] || user.username?.[0] || '?').toUpperCase()}
                    </span>
                  </div>
                )}

                {others.length > 0 && (
                  <div className="p-2.5 grid grid-cols-3 gap-1.5">
                    {others.map((p) => (
                      <div
                        key={p.id}
                        className="overflow-hidden rounded-xl cursor-pointer group"
                        onClick={() => openLightbox(allPhotos.indexOf(p))}
                      >
                        <img
                          src={p.url} alt=""
                          className="w-full aspect-square object-cover group-hover:scale-110 group-hover:opacity-90 transition-transform duration-300"
                          referrerPolicy="no-referrer"
                        />
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex-1 p-5 sm:p-6 flex flex-col min-w-0">
                <div>
                  <h2 className="text-2xl font-bold text-slate-800 leading-tight">
                    {user.first_name} {user.last_name}
                  </h2>
                  <p className="text-slate-400 text-sm mt-0.5">@{user.username}</p>

                  {presenceState && (
                    <p className="text-sm mt-2">
                      {presenceState.is_online
                        ? <span className="text-emerald-600 font-medium">● Online now</span>
                        : <span className="text-slate-400">Last seen: {presenceState.last_seen ? new Date(presenceState.last_seen).toLocaleString() : 'unknown'}</span>
                      }
                    </p>
                  )}

                  <div className="mt-3 flex flex-wrap gap-2">
                    {user.gender && <span className="px-3 py-1 bg-slate-100 rounded-full text-sm text-slate-600">{user.gender}</span>}
                    {user.birth_date && <span className="px-3 py-1 bg-slate-100 rounded-full text-sm text-slate-600">{age(user.birth_date)} yo</span>}
                    {Array.isArray(user.sexual_preference) && user.sexual_preference.length > 0 && (
                      <span className="px-3 py-1 bg-slate-100 rounded-full text-sm text-slate-600">
                        Likes {user.sexual_preference.join(', ')}
                      </span>
                    )}
                    {user.relationship_goal && (
                      <span className="px-3 py-1 bg-rose-50 rounded-full text-sm text-rose-600">
                        {user.relationship_goal.replace(/-/g, ' ')}
                      </span>
                    )}
                  </div>

                  {user.city && <p className="mt-2.5 text-slate-500 text-sm">📍 {user.city}</p>}

                  <div className="mt-2.5 flex flex-wrap gap-2 text-xs">
                    {isMatch && <span className="px-2.5 py-1 rounded-full bg-emerald-50 text-emerald-700 font-medium">✓ Match</span>}
                    {likedMe && <span className="px-2.5 py-1 rounded-full bg-blue-50 text-blue-700 font-medium">Liked you</span>}
                    {iLiked && <span className="px-2.5 py-1 rounded-full bg-rose-50 text-rose-700 font-medium">You liked</span>}
                  </div>

                  {user.bio && <p className="mt-4 text-slate-600 text-sm leading-relaxed">{user.bio}</p>}

                  <div className="mt-3 flex items-center gap-1 text-amber-500 text-sm font-medium">
                    <span>★</span><span>{user.fame_rating ?? 0} fame</span>
                  </div>

                  {Array.isArray(user.tags) && user.tags.length > 0 && (
                    <div className="mt-3 flex flex-wrap gap-1.5">
                      {user.tags.map((tag) => (
                        <span key={tag} className="text-xs px-2.5 py-1 bg-rose-50 text-rose-600 rounded-full">#{tag}</span>
                      ))}
                    </div>
                  )}
                </div>

                <div className="mt-auto pt-5">
                  {!hasPrimaryPhoto && (
                    <p className="mb-3 text-amber-600 text-xs">Add a profile picture to like others.</p>
                  )}
                  <div className="flex gap-2 flex-wrap">
                    {iLiked ? (
                      <button onClick={onUnlike} disabled={liking || blocked}
                        className="px-4 py-2 border border-slate-300 text-slate-700 rounded-full hover:bg-slate-50 disabled:opacity-60 font-medium text-sm">
                        {liking ? '...' : 'Unlike'}
                      </button>
                    ) : (
                      <button onClick={onLike} disabled={liking || blocked || !hasPrimaryPhoto}
                        className="px-4 py-2 bg-rose-500 text-white rounded-full hover:bg-rose-600 disabled:opacity-60 font-medium text-sm">
                        {liking ? '...' : '♡ Like'}
                      </button>
                    )}
                    {isMatch && !blocked && (
                      <Link to={`/chat/${userId}`} className="px-4 py-2 bg-emerald-500 text-white rounded-full hover:bg-emerald-600 font-medium text-sm">
                        💬 Chat
                      </Link>
                    )}
                    <button onClick={onToggleBlock} disabled={blocking}
                      className={`px-4 py-2 rounded-full border font-medium text-sm disabled:opacity-60 ${blocked ? 'border-emerald-300 text-emerald-700 hover:bg-emerald-50' : 'border-slate-300 text-slate-600 hover:bg-slate-50'}`}>
                      {blocking ? '...' : blocked ? 'Unblock' : 'Block'}
                    </button>
                    <button onClick={() => setReportModal(true)} disabled={reporting}
                      className="px-4 py-2 rounded-full border border-amber-300 text-amber-700 hover:bg-amber-50 disabled:opacity-60 font-medium text-sm">
                      Report
                    </button>
                  </div>
                  {info && <p className="mt-2.5 text-emerald-600 text-sm">{info}</p>}
                  {error && <p className="mt-2.5 text-rose-600 text-sm">{error}</p>}

                  <div className="mt-4 pt-4 border-t border-slate-100">
                    <Link to={`/users/${userId}`} className="text-sm text-slate-400 hover:text-rose-500 transition">
                      View full profile →
                    </Link>
                  </div>
                </div>
              </div>

            </div>
          )}
        </div>
      </div>

      {isLightboxOpen && (
        <div
          className="fixed inset-0 bg-black/95 flex items-center justify-center z-[60]"
          onClick={closeLightbox}
          onTouchStart={handleTouchStart}
          onTouchEnd={handleTouchEnd}
        >
          <img
            src={allPhotos[lightboxIndex]?.url}
            alt="Full view"
            className="max-w-full max-h-full object-contain select-none"
            referrerPolicy="no-referrer"
            onClick={(e) => e.stopPropagation()}
          />
          <button onClick={closeLightbox}
            className="absolute top-4 right-4 text-white/80 hover:text-white text-4xl leading-none" aria-label="Close">
            ×
          </button>
          {allPhotos.length > 1 && (
            <>
              <button onClick={(e) => { e.stopPropagation(); prevPhoto() }}
                className="absolute left-4 top-1/2 -translate-y-1/2 text-white/70 hover:text-white text-5xl leading-none px-2" aria-label="Previous">
                ‹
              </button>
              <button onClick={(e) => { e.stopPropagation(); nextPhoto() }}
                className="absolute right-4 top-1/2 -translate-y-1/2 text-white/70 hover:text-white text-5xl leading-none px-2" aria-label="Next">
                ›
              </button>
              <div className="absolute bottom-6 left-1/2 -translate-x-1/2 flex gap-2">
                {allPhotos.map((_, i) => (
                  <button key={i}
                    onClick={(e) => { e.stopPropagation(); setLightboxIndex(i) }}
                    className={`w-2 h-2 rounded-full transition ${i === lightboxIndex ? 'bg-white' : 'bg-white/40 hover:bg-white/60'}`}
                  />
                ))}
              </div>
            </>
          )}
        </div>
      )}

      {reportModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[70] p-4" onClick={closeReportModal}>
          <div className="bg-white rounded-xl p-6 max-w-sm w-full shadow-xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-semibold text-slate-800 mb-3">Report profile</h3>
            {!selectedReportReason ? (
              <>
                <p className="text-sm text-slate-600 mb-4">Choose report reason:</p>
                <div className="flex flex-col gap-2">
                  {[
                    { value: 'fake_account', label: 'Fake account' },
                    { value: 'spam', label: 'Spam' },
                    { value: 'harassment', label: 'Harassment' },
                    { value: 'inappropriate', label: 'Inappropriate content' },
                    { value: 'scam', label: 'Scam' },
                    { value: 'other', label: 'Other' },
                  ].map((opt) => (
                    <button key={opt.value} onClick={() => setSelectedReportReason(opt.value)} disabled={reporting}
                      className="text-left px-4 py-2 rounded border border-slate-200 hover:bg-slate-50 text-slate-700 disabled:opacity-60">
                      {opt.label}
                    </button>
                  ))}
                </div>
                <button onClick={closeReportModal} className="mt-4 w-full py-2 text-slate-500 hover:text-slate-700 text-sm">
                  Cancel
                </button>
              </>
            ) : (
              <>
                <p className="text-sm text-slate-700 mb-4">Do you also want to add this user to block list?</p>
                <div className="flex flex-col gap-2">
                  <button onClick={() => onReport(selectedReportReason, true)} disabled={reporting}
                    className="px-4 py-2 rounded border border-rose-300 bg-rose-50 text-rose-700 hover:bg-rose-100 disabled:opacity-60">
                    {reporting ? 'Sending...' : 'Yes, report and block'}
                  </button>
                  <button onClick={() => onReport(selectedReportReason, false)} disabled={reporting}
                    className="px-4 py-2 rounded border border-slate-300 text-slate-700 hover:bg-slate-50 disabled:opacity-60">
                    {reporting ? 'Sending...' : 'No, only report'}
                  </button>
                </div>
                <button
                  onClick={() => setSelectedReportReason('')}
                  disabled={reporting}
                  className="mt-4 w-full py-2 text-slate-500 hover:text-slate-700 text-sm disabled:opacity-60"
                >
                  Back
                </button>
              </>
            )}
          </div>
        </div>
      )}

      <style>{`
        @keyframes modalIn {
          from { opacity: 0; transform: scale(0.93) translateY(12px); }
          to   { opacity: 1; transform: scale(1) translateY(0); }
        }
      `}</style>
    </>
  )
}
