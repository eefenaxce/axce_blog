import * as React from "react"
import { cn } from "../../lib/utils"
import { X } from "lucide-react"
import { motion, AnimatePresence } from "framer-motion"
import { ScrollArea } from "./scroll-area"

interface ModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  children: React.ReactNode
  size?: "sm" | "md" | "lg" | "xl" | "2xl"
}

const sizes = {
  sm: "max-w-sm",
  md: "max-w-md",
  lg: "max-w-lg",
  xl: "max-w-xl",
  "2xl": "max-w-6xl",
}

export function Modal({ open, onOpenChange, title, children, size = "md" }: ModalProps) {
  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm"
            onClick={() => onOpenChange(false)}
          />
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className={cn(
              "fixed left-1/2 top-1/2 z-50 flex w-full max-h-[calc(100vh-4rem)] -translate-x-1/2 -translate-y-1/2 flex-col rounded-xl border bg-background shadow-2xl",
              sizes[size],
            )}
          >
            <div className="flex items-center justify-between border-b px-6 py-4">
              <h3 className="text-lg font-semibold">{title}</h3>
              <button
                onClick={() => onOpenChange(false)}
                className="rounded-full p-1 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <ScrollArea className="flex-1" maxHeight="calc(100vh - 8rem)">
              <div className="p-6">{children}</div>
            </ScrollArea>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}