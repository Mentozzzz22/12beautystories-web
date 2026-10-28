import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import type { Banner } from '../api/types'

const AUTOPLAY_MS = 6000

export function BannerCarousel({ banners }: { banners: Banner[] }) {
  const [index, setIndex] = useState(0)
  const [paused, setPaused] = useState(false)
  const touchX = useRef<number | null>(null)
  const count = banners.length

  useEffect(() => {
    if (count < 2 || paused) return
    const id = window.setInterval(() => setIndex((i) => (i + 1) % count), AUTOPLAY_MS)
    return () => window.clearInterval(id)
  }, [count, paused])

  if (count === 0) return null

  const go = (delta: number) => setIndex((i) => (i + delta + count) % count)

  return (
    <section
      aria-roledescription="carousel"
      aria-label="Акции и новинки"
      className="relative overflow-hidden rounded-3xl bg-sand"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onTouchStart={(e) => {
        touchX.current = e.touches[0]?.clientX ?? null
      }}
      onTouchEnd={(e) => {
        const start = touchX.current
        const end = e.changedTouches[0]?.clientX
        touchX.current = null
        if (start === null || end === undefined || Math.abs(end - start) < 40) return
        go(end < start ? 1 : -1)
      }}
    >
      <div
        className="flex transition-transform duration-500 ease-out"
        style={{ transform: `translateX(-${index * 100}%)` }}
      >
        {banners.map((b, i) => (
          <BannerSlide key={b.id} banner={b} active={i === index} />
        ))}
      </div>

      {count > 1 && (
        <div className="absolute inset-x-0 bottom-4 flex justify-center gap-2">
          {banners.map((b, i) => (
            <button
              key={b.id}
              type="button"
              aria-label={`Слайд ${i + 1}`}
              aria-current={i === index}
              onClick={() => setIndex(i)}
              className={`h-1.5 rounded-full transition-all ${
                i === index ? 'w-6 bg-white' : 'w-1.5 bg-white/60'
              }`}
            />
          ))}
        </div>
      )}
    </section>
  )
}

function BannerSlide({ banner, active }: { banner: Banner; active: boolean }) {
  const content = (
    <>
      <picture>
        {banner.mobileImageUrl && (
          <source media="(max-width: 640px)" srcSet={banner.mobileImageUrl} />
        )}
        <img
          src={banner.imageUrl}
          alt={banner.title}
          className="aspect-[4/5] w-full object-cover sm:aspect-[16/6]"
        />
      </picture>
      {(banner.title || banner.subtitle) && (
        <div className="absolute inset-0 flex flex-col justify-end bg-gradient-to-t from-ink/60 via-ink/10 to-transparent p-6 pb-12 text-white sm:p-10 sm:pb-14">
          {banner.title && <h2 className="text-3xl sm:text-5xl">{banner.title}</h2>}
          {banner.subtitle && <p className="mt-2 text-sm sm:text-base">{banner.subtitle}</p>}
        </div>
      )}
    </>
  )

  const className = 'relative block w-full shrink-0'
  const tabIndex = active ? 0 : -1
  if (!banner.linkUrl) return <div className={className}>{content}</div>
  if (/^https?:\/\//.test(banner.linkUrl)) {
    return (
      <a href={banner.linkUrl} className={className} tabIndex={tabIndex}>
        {content}
      </a>
    )
  }
  return (
    <Link to={banner.linkUrl} className={className} tabIndex={tabIndex}>
      {content}
    </Link>
  )
}
