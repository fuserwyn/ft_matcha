import { useState, useEffect, useCallback, useRef } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { presence, users, photos } from '../api/client'

export default function UserProfile() {
  const { id } = useParams()
  const navigate = useNavigate()
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
    if (!isLightboxOpen) return
    const onKey = (e) => {
      if (e.key === 'Escape') closeLightbox()
      if (e.key === 'ArrowLeft') prevPhoto()
      if (e.key === 'ArrowRight') nextPhoto()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isLightboxOpen, closeLightbox, prevPhoto, nextPhoto])

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const [u, p, myPhotos] = await Promise.all([
          users.getById(id),
          presence.get(id),
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
  }, [id])

  const age = (birthDate) => {
    if (!birthDate) return null
    const diff = Date.now() - new Date(birthDate).getTime()
    return Math.floor(diff / (365.25 * 24 * 60 * 60 * 1000))
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin w-10 h-10 border-2 border-rose-400 border-t-transparent rounded-full" />
      </div>
    )
  }

  if (error || !user) {
    return (
      <div className="text-center py-12">
        <p className="text-rose-600 mb-4">{error || 'User not found'}</p>
        <button onClick={() => navigate('/discover')} className="text-rose-600 hover:underline">
          Back to Discover
        </button>
      </div>
    )
  }

  const onLike = async () => {
    if (blocked || !hasPrimaryPhoto) return
    setLiking(true); setInfo(''); setError('')
    try {
      const res = await users.like(id)
      if (res?.is_match) { setIsMatch(true); setILiked(true); setInfo("It's a match! You can open chat now.") }
      else { setILiked(true); setInfo('Liked') }
    } catch (err) { setError(err.message || 'Failed to like user') }
    finally { setLiking(false) }
  }

  const onUnlike = async () => {
    if (blocked) return
    setLiking(true); setInfo(''); setError('')
    try { await users.unlike(id); setILiked(false); setIsMatch(false); setInfo('Like removed') }
    catch (err) { setError(err.message || 'Failed to unlike user') }
    finally { setLiking(false) }
  }

  const onReport = async (reason, comment) => {
    setReporting(true); setInfo(''); setError('')
    try { await users.report(id, { reason, comment: comment || null }); setReportModal(false); setInfo('Report submitted. Thank you.') }
    catch (err) { setError(err.message || 'Failed to report') }
    finally { setReporting(false) }
  }

  const onToggleBlock = async () => {
    setBlocking(true); setInfo(''); setError('')
    try {
      if (blocked) { await users.unblock(id); setBlocked(false); setInfo('User unblocked') }
      else { await users.block(id); setBlocked(true); setInfo('User blocked') }
    } catch (err) { setError(err.message || 'Failed to update block state') }
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
    <div className="max-w-4xl w-full min-w-0">
      <button onClick={() => navigate(-1)} className="text-slate-600 hover:text-rose-600 mb-4 text-sm">
        ← Back
      </button>

      {/* ── Lightbox ── */}
      {isLightboxOpen && (
        <div
          className="fixed inset-0 bg-black/95 flex items-center justify-center z-50"
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

          {/* Close */}
          <button onClick={closeLightbox}
            className="absolute top-4 right-4 text-white/80 hover:text-white text-4xl leading-none" aria-label="Close">
            ×
          </button>

          {/* Prev */}
          {allPhotos.length > 1 && (
            <button
              onClick={(e) => { e.stopPropagation(); prevPhoto() }}
              className="absolute left-4 top-1/2 -translate-y-1/2 text-white/70 hover:text-white text-5xl leading-none px-2"
              aria-label="Previous"
            >
              ‹
            </button>
          )}

          {/* Next */}
          {allPhotos.length > 1 && (
            <button
              onClick={(e) => { e.stopPropagation(); nextPhoto() }}
              className="absolute right-16 top-1/2 -translate-y-1/2 text-white/70 hover:text-white text-5xl leading-none px-2"
              aria-label="Next"
            >
              ›
          </button>
          )}

          {/* Dot indicators */}
          {allPhotos.length > 1 && (
            <div className="absolute bottom-6 left-1/2 -translate-x-1/2 flex gap-2">
              {allPhotos.map((_, i) => (
                <button
                  key={i}
                  onClick={(e) => { e.stopPropagation(); setLightboxIndex(i) }}
                  className={`w-2 h-2 rounded-full transition ${i === lightboxIndex ? 'bg-white' : 'bg-white/40 hover:bg-white/60'}`}
                />
              ))}
            </div>
          )}
        </div>
      )}

      <div className="bg-white rounded-2xl shadow-lg border border-slate-100 overflow-hidden">
        <div className="lg:flex">

          {/* ── Left: photos ── */}
          <div className="lg:w-80 xl:w-96 shrink-0 border-b lg:border-b-0 lg:border-r border-slate-100">
            {/* Primary photo — full width */}
            {primary ? (
              <div
                className="cursor-pointer group overflow-hidden"
                onClick={() => openLightbox(primaryPhotoIdx)}
              >
                <img
                  src={primary.url}
                  alt={`${user.first_name} ${user.last_name}`}
                  className="w-full aspect-[4/5] object-cover group-hover:scale-105 transition-transform duration-500"
                  referrerPolicy="no-referrer"
                />
              </div>
            ) : (
              <div className="aspect-[4/5] bg-gradient-to-br from-slate-100 to-slate-200 flex items-center justify-center">
                <span className="text-8xl font-bold text-slate-300">
                  {(user.first_name?.[0] || user.username?.[0] || '?').toUpperCase()}
                </span>
              </div>
            )}

            {/* Other photos — 2 columns (50% width each) */}
            {others.length > 0 && (
              <div className="p-3 pt-3 grid grid-cols-2 gap-2">
                {others.map((p, i) => (
                  <div
                    key={p.id}
                    className="overflow-hidden rounded-xl cursor-pointer group"
                    onClick={() => openLightbox(allPhotos.indexOf(p))}
                  >
                    <img
                      src={p.url}
                      alt=""
                      className="w-full aspect-square object-cover group-hover:scale-105 transition-transform duration-500 group-hover:opacity-90"
                      referrerPolicy="no-referrer"
                    />
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* ── Right: info ── */}
          <div className="flex-1 p-5 sm:p-7 flex flex-col">
            <div>
              <h1 className="text-2xl font-bold text-slate-800">{user.first_name} {user.last_name}</h1>
              <p className="text-slate-400 text-sm">@{user.username}</p>

              {presenceState && (
                <p className="text-sm mt-1.5">
                  {presenceState.is_online
                    ? <span className="text-emerald-600 font-medium">● Online now</span>
                    : <span className="text-slate-400">Last seen: {presenceState.last_seen ? new Date(presenceState.last_seen).toLocaleString() : 'unknown'}</span>
                  }
                </p>
              )}

              <div className="mt-4 flex flex-wrap gap-2">
                {user.gender && <span className="px-3 py-1 bg-slate-100 rounded-full text-sm text-slate-600">{user.gender}</span>}
                {user.birth_date && <span className="px-3 py-1 bg-slate-100 rounded-full text-sm text-slate-600">{age(user.birth_date)} years old</span>}
                {user.sexual_preference && <span className="px-3 py-1 bg-slate-100 rounded-full text-sm text-slate-600">Interested in {user.sexual_preference}</span>}
              </div>

              {user.city && <p className="mt-3 text-slate-500 text-sm">📍 {user.city}</p>}

              <div className="mt-3 flex flex-wrap gap-2 text-xs">
                {isMatch && <span className="px-2.5 py-1 rounded-full bg-emerald-50 text-emerald-700 font-medium">✓ Match</span>}
                {likedMe && <span className="px-2.5 py-1 rounded-full bg-blue-50 text-blue-700 font-medium">Liked you</span>}
                {iLiked && <span className="px-2.5 py-1 rounded-full bg-rose-50 text-rose-700 font-medium">You liked</span>}
              </div>

              {user.bio && <p className="mt-4 text-slate-600 leading-relaxed">{user.bio}</p>}

              <div className="mt-4 flex items-center gap-1 text-amber-500 text-sm font-medium">
                <span>★</span><span>{user.fame_rating ?? 0} fame</span>
              </div>

              {Array.isArray(user.tags) && user.tags.length > 0 && (
                <div className="mt-4 flex flex-wrap gap-2">
                  {user.tags.map((tag) => (
                    <span key={tag} className="text-xs px-2.5 py-1 bg-rose-50 text-rose-600 rounded-full">#{tag}</span>
                  ))}
                </div>
              )}
            </div>

            <div className="mt-auto pt-6">
              {!hasPrimaryPhoto && (
                <p className="mb-3 text-amber-600 text-sm">Add a profile picture to like other users.</p>
              )}
              <div className="flex gap-3 flex-wrap">
                {iLiked ? (
                  <button onClick={onUnlike} disabled={liking || blocked}
                    className="px-5 py-2.5 border border-slate-300 text-slate-700 rounded-full hover:bg-slate-50 disabled:opacity-60 font-medium text-sm">
                    {liking ? 'Removing...' : 'Unlike'}
                  </button>
                ) : (
                  <button onClick={onLike} disabled={liking || blocked || !hasPrimaryPhoto}
                    className="px-5 py-2.5 bg-rose-500 text-white rounded-full hover:bg-rose-600 disabled:opacity-60 font-medium text-sm">
                    {liking ? 'Liking...' : '♡ Like'}
                  </button>
                )}
                {isMatch && !blocked && (
                  <Link to={`/chat/${id}`} className="px-5 py-2.5 bg-emerald-500 text-white rounded-full hover:bg-emerald-600 font-medium text-sm">
                    💬 Chat
                  </Link>
                )}
                <button onClick={onToggleBlock} disabled={blocking}
                  className={`px-5 py-2.5 rounded-full border font-medium text-sm ${blocked ? 'border-emerald-300 text-emerald-700 hover:bg-emerald-50' : 'border-slate-300 text-slate-600 hover:bg-slate-50'} disabled:opacity-60`}>
                  {blocking ? '...' : blocked ? 'Unblock' : 'Block'}
                </button>
                <button onClick={() => setReportModal(true)} disabled={reporting}
                  className="px-5 py-2.5 rounded-full border border-amber-300 text-amber-700 hover:bg-amber-50 disabled:opacity-60 font-medium text-sm">
                  Report
                </button>
              </div>
              {info && <p className="mt-3 text-emerald-600 text-sm">{info}</p>}
              {error && <p className="mt-3 text-rose-600 text-sm">{error}</p>}
            </div>
          </div>
        </div>
      </div>

      {reportModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={() => setReportModal(false)}>
          <div className="bg-white rounded-xl p-6 max-w-sm w-full shadow-xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-semibold text-slate-800 mb-3">Report profile</h3>
            <p className="text-sm text-slate-600 mb-4">Report this profile as:</p>
            <div className="flex flex-col gap-2">
              {[
                { value: 'fake_account', label: 'Fake account' },
                { value: 'spam', label: 'Spam' },
                { value: 'harassment', label: 'Harassment' },
                { value: 'inappropriate', label: 'Inappropriate content' },
                { value: 'scam', label: 'Scam' },
                { value: 'other', label: 'Other' },
              ].map((opt) => (
                <button key={opt.value} onClick={() => onReport(opt.value)} disabled={reporting}
                  className="text-left px-4 py-2 rounded border border-slate-200 hover:bg-slate-50 text-slate-700 disabled:opacity-60">
                  {opt.label}
                </button>
              ))}
            </div>
            <button onClick={() => setReportModal(false)} className="mt-4 w-full py-2 text-slate-600 hover:text-slate-800">
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
