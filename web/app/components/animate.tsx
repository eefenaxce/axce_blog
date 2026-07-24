import { motion, type Variants } from "framer-motion"
import type { ReactNode } from "react"

const fadeIn: Variants = {
  hidden: { opacity: 0, y: 12 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.3, ease: "easeOut" } },
}

const slideUp: Variants = {
  hidden: { opacity: 0, y: 24 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.35, ease: "easeOut" } },
}

const scaleIn: Variants = {
  hidden: { opacity: 0, scale: 0.95 },
  visible: { opacity: 1, scale: 1, transition: { duration: 0.25, ease: "easeOut" } },
}

const stagger: Variants = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.06 } },
}

const listItem: Variants = {
  hidden: { opacity: 0, x: -12 },
  visible: { opacity: 1, x: 0, transition: { duration: 0.25, ease: "easeOut" } },
}

export function FadeIn({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <motion.div variants={fadeIn} initial="hidden" animate="visible" className={className}>
      {children}
    </motion.div>
  )
}

export function SlideUp({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <motion.div variants={slideUp} initial="hidden" animate="visible" className={className}>
      {children}
    </motion.div>
  )
}

export function ScaleIn({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <motion.div variants={scaleIn} initial="hidden" animate="visible" className={className}>
      {children}
    </motion.div>
  )
}

export function Stagger({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <motion.div variants={stagger} initial="hidden" animate="visible" className={className}>
      {children}
    </motion.div>
  )
}

export function StaggerTbody({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <motion.tbody variants={stagger} initial="hidden" animate="visible" className={className}>
      {children}
    </motion.tbody>
  )
}

export function SlideIn({ children, className, delay = 0 }: { children: ReactNode; className?: string; delay?: number }) {
  return (
    <motion.div
      initial={{ opacity: 0, x: -16 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.3, delay, ease: "easeOut" }}
      className={className}
    >
      {children}
    </motion.div>
  )
}

export const itemVariants = listItem

// Page transition wrapper
const pageVariants: Variants = {
  initial: { opacity: 0, y: 8 },
  enter: { opacity: 1, y: 0, transition: { duration: 0.25, ease: "easeOut" } },
  exit: { opacity: 0, y: -8, transition: { duration: 0.15, ease: "easeIn" } },
}

export function AnimatedPage({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <motion.div
      variants={pageVariants}
      initial="initial"
      animate="enter"
      exit="exit"
      className={className}
    >
      {children}
    </motion.div>
  )
}
