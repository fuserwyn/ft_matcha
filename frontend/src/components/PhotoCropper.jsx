import { useState, useCallback } from 'react'
import Cropper from 'react-easy-crop'

const ASPECT = 3 / 4

function getCroppedBlob(imageSrc, pixelCrop) {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.addEventListener('load', () => {
      const canvas = document.createElement('canvas')
      canvas.width = pixelCrop.width
      canvas.height = pixelCrop.height
      const ctx = canvas.getContext('2d')
      ctx.drawImage(
        image,
        pixelCrop.x,
        pixelCrop.y,
        pixelCrop.width,
        pixelCrop.height,
        0,
        0,
        pixelCrop.width,
        pixelCrop.height,
      )
      canvas.toBlob((blob) => {
        if (blob) resolve(blob)
        else reject(new Error('Canvas toBlob failed'))
      }, 'image/jpeg', 0.92)
    })
    image.addEventListener('error', reject)
    image.src = imageSrc
  })
}

export default function PhotoCropper({ imageSrc, onConfirm, onCancel }) {
  const [crop, setCrop] = useState({ x: 0, y: 0 })
  const [zoom, setZoom] = useState(1)
  const [croppedAreaPixels, setCroppedAreaPixels] = useState(null)
  const [processing, setProcessing] = useState(false)

  const onCropComplete = useCallback((_, pixels) => {
    setCroppedAreaPixels(pixels)
  }, [])

  const handleConfirm = async () => {
    if (!croppedAreaPixels) return
    setProcessing(true)
    try {
      const blob = await getCroppedBlob(imageSrc, croppedAreaPixels)
      onConfirm(blob)
    } catch {
    } finally {
      setProcessing(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/90 z-50 flex flex-col">
      <div className="flex items-center justify-between px-5 py-4 shrink-0">
        <button onClick={onCancel} className="text-white/70 hover:text-white text-sm font-medium">
          Cancel
        </button>
        <p className="text-white font-semibold text-sm">Move and Scale</p>
        <button
          onClick={handleConfirm}
          disabled={processing}
          className="text-rose-400 hover:text-rose-300 font-semibold text-sm disabled:opacity-50"
        >
          {processing ? 'Processing...' : 'Choose'}
        </button>
      </div>

      <div className="relative flex-1">
        <Cropper
          image={imageSrc}
          crop={crop}
          zoom={zoom}
          aspect={ASPECT}
          onCropChange={setCrop}
          onZoomChange={setZoom}
          onCropComplete={onCropComplete}
          cropShape="rect"
          showGrid={false}
          style={{
            containerStyle: { background: '#000' },
            cropAreaStyle: {
              border: '2px solid rgba(255,255,255,0.8)',
              borderRadius: '12px',
            },
          }}
        />
      </div>

      <div className="shrink-0 px-8 pb-6 pt-4 flex items-center gap-4">
        <span className="text-white/50 text-lg">⊖</span>
        <input
          type="range"
          min={1}
          max={3}
          step={0.01}
          value={zoom}
          onChange={(e) => setZoom(Number(e.target.value))}
          className="flex-1 accent-rose-500 h-1 rounded"
        />
        <span className="text-white/50 text-lg">⊕</span>
      </div>
    </div>
  )
}
