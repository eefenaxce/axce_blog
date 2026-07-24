import { useRef, useState, useCallback, useEffect, type ReactNode } from "react"
import { cn } from "~/lib/utils"

interface ScrollAreaProps {
  className?: string
  children: ReactNode
  maxHeight?: string
}

export function ScrollArea({ className, children, maxHeight }: ScrollAreaProps) {
  const viewportRef = useRef<HTMLDivElement>(null)

  // Vertical
  const [hasOverflowY, setHasOverflowY] = useState(false)
  const [thumbHeight, setThumbHeight] = useState(0)
  const [thumbTop, setThumbTop] = useState(0)

  // Horizontal
  const [hasOverflowX, setHasOverflowX] = useState(false)
  const [thumbWidth, setThumbWidth] = useState(0)
  const [thumbLeft, setThumbLeft] = useState(0)

  const [isDraggingY, setIsDraggingY] = useState(false)
  const [isDraggingX, setIsDraggingX] = useState(false)
  const [isHovering, setIsHovering] = useState(false)
  const dragStartRef = useRef({ x: 0, y: 0, scrollTop: 0, scrollLeft: 0 })

  const updateThumb = useCallback(() => {
    const viewport = viewportRef.current
    if (!viewport) return

    // Vertical
    const { scrollTop, scrollHeight, clientHeight, scrollLeft, scrollWidth, clientWidth } = viewport
    if (scrollHeight > clientHeight) {
      setHasOverflowY(true)
      const h = Math.max((clientHeight / scrollHeight) * clientHeight, 32)
      const t = (scrollTop / (scrollHeight - clientHeight)) * (clientHeight - h)
      setThumbHeight(h)
      setThumbTop(t)
    } else {
      setHasOverflowY(false)
    }

    // Horizontal
    if (scrollWidth > clientWidth) {
      setHasOverflowX(true)
      const w = Math.max((clientWidth / scrollWidth) * clientWidth, 32)
      const l = (scrollLeft / (scrollWidth - clientWidth)) * (clientWidth - w)
      setThumbWidth(w)
      setThumbLeft(l)
    } else {
      setHasOverflowX(false)
    }
  }, [])

  const handleScroll = useCallback(() => {
    updateThumb()
  }, [updateThumb])

  const handleMouseEnter = () => setIsHovering(true)
  const handleMouseLeave = () => {
    if (!isDraggingY && !isDraggingX) setIsHovering(false)
  }

  // Vertical thumb drag
  const handleThumbYMousedown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDraggingY(true)
    const v = viewportRef.current!
    dragStartRef.current = { ...dragStartRef.current, y: e.clientY, scrollTop: v.scrollTop }
  }, [])

  // Horizontal thumb drag
  const handleThumbXMousedown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDraggingX(true)
    const v = viewportRef.current!
    dragStartRef.current = { ...dragStartRef.current, x: e.clientX, scrollLeft: v.scrollLeft }
  }, [])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      const viewport = viewportRef.current
      if (!viewport) return

      if (isDraggingY) {
        const { y: startY, scrollTop: startScrollTop } = dragStartRef.current
        const deltaY = e.clientY - startY
        const maxScrollTop = viewport.scrollHeight - viewport.clientHeight
        const trackHeight = viewport.clientHeight - thumbHeight
        const ratio = trackHeight > 0 ? deltaY / trackHeight : 0
        viewport.scrollTop = Math.max(0, Math.min(startScrollTop + ratio * maxScrollTop, maxScrollTop))
      }

      if (isDraggingX) {
        const { x: startX, scrollLeft: startScrollLeft } = dragStartRef.current
        const deltaX = e.clientX - startX
        const maxScrollLeft = viewport.scrollWidth - viewport.clientWidth
        const trackWidth = viewport.clientWidth - thumbWidth
        const ratio = trackWidth > 0 ? deltaX / trackWidth : 0
        viewport.scrollLeft = Math.max(0, Math.min(startScrollLeft + ratio * maxScrollLeft, maxScrollLeft))
      }
    }

    const handleMouseUp = () => {
      setIsDraggingY(false)
      setIsDraggingX(false)
    }

    if (isDraggingY || isDraggingX) {
      document.addEventListener("mousemove", handleMouseMove)
      document.addEventListener("mouseup", handleMouseUp)
      document.body.style.userSelect = "none"
    }
    return () => {
      document.removeEventListener("mousemove", handleMouseMove)
      document.removeEventListener("mouseup", handleMouseUp)
      document.body.style.userSelect = ""
    }
  }, [isDraggingY, isDraggingX, thumbHeight, thumbWidth])

  useEffect(() => {
    updateThumb()
  }, [updateThumb])

  useEffect(() => {
    const viewport = viewportRef.current
    if (!viewport) return
    const observer = new ResizeObserver(() => updateThumb())
    observer.observe(viewport)
    return () => observer.disconnect()
  }, [updateThumb])

  const showVThumb = hasOverflowY
  const showHThumb = hasOverflowX

  return (
    <div
      className={cn("relative overflow-hidden", className)}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {/* Viewport */}
      <div
        ref={viewportRef}
        onScroll={handleScroll}
        className="h-full w-full overflow-auto scrollbar-none"
        style={maxHeight ? { maxHeight } : undefined}
      >
        {children}
      </div>

      {/* Vertical scrollbar */}
      {hasOverflowY && (
        <div
          className={cn(
            "absolute right-0 top-1 bottom-1 w-2.5 z-10 pointer-events-none transition-opacity duration-150",
            showVThumb ? "opacity-100" : "opacity-0"
          )}
        >
          <div
            onMouseDown={handleThumbYMousedown}
            className="absolute right-0.5 rounded-full bg-muted-foreground/25 hover:bg-muted-foreground/50 active:bg-muted-foreground/70 transition-colors pointer-events-auto cursor-grab active:cursor-grabbing"
            style={{ height: `${thumbHeight}px`, top: `${thumbTop}px`, width: "6px" }}
          />
        </div>
      )}

      {/* Horizontal scrollbar */}
      {hasOverflowX && (
        <div
          className={cn(
            "absolute bottom-0 left-1 right-1 h-2.5 z-10 pointer-events-none transition-opacity duration-150",
            showHThumb ? "opacity-100" : "opacity-0"
          )}
        >
          <div
            onMouseDown={handleThumbXMousedown}
            className="absolute bottom-0.5 rounded-full bg-muted-foreground/25 hover:bg-muted-foreground/50 active:bg-muted-foreground/70 transition-colors pointer-events-auto cursor-grab active:cursor-grabbing"
            style={{ width: `${thumbWidth}px`, left: `${thumbLeft}px`, height: "5px" }}
          />
        </div>
      )}
    </div>
  )
}
