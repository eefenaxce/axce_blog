import * as React from "react"
import { cn } from "../../lib/utils"
import { motion, AnimatePresence } from "framer-motion"
import { ChevronDown } from "lucide-react"

interface SelectOption {
  value: string
  label: React.ReactNode
}

interface SelectProps {
  value?: string
  onValueChange?: (value: string) => void
  disabled?: boolean
  placeholder?: string
  options: SelectOption[]
}

export function Select({ 
  value, 
  onValueChange, 
  disabled, 
  placeholder = "请选择",
  options 
}: SelectProps) {
  const [open, setOpen] = React.useState(false)
  const triggerRef = React.useRef<HTMLButtonElement>(null)
  
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (triggerRef.current && !triggerRef.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    
    if (open) {
      document.addEventListener("mousedown", handleClickOutside)
    }
    
    return () => {
      document.removeEventListener("mousedown", handleClickOutside)
    }
  }, [open])
  
  const selectedOption = options.find(opt => opt.value === value)
  
  const handleSelect = (optionValue: string) => {
    onValueChange?.(optionValue)
    setOpen(false)
  }
  
  return (
    <div className="relative inline-flex w-full">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => !disabled && setOpen(!open)}
        disabled={disabled}
        className={cn(
          "flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
          open && "ring-2 ring-ring ring-offset-2"
        )}
      >
        {selectedOption?.label || placeholder}
        <ChevronDown 
          className={cn(
            "h-4 w-4 flex-shrink-0 text-muted-foreground transition-transform duration-200",
            open && "rotate-180"
          )} 
        />
      </button>
      
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -8, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -8, scale: 0.95 }}
            transition={{ duration: 0.15, ease: "easeOut" }}
            className="absolute left-0 right-0 top-full z-50 mt-1 overflow-hidden rounded-md border border-input bg-background shadow-lg"
          >
            <div className="py-1">
              {options.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => handleSelect(option.value)}
                  className={cn(
                    "flex w-full cursor-pointer select-none items-center rounded-sm px-3 py-2 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground",
                    value === option.value && "bg-accent text-accent-foreground"
                  )}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

// 保持旧的 API 兼容
export const SelectTrigger = ({ children }: { children?: React.ReactNode }) => <>{children}</>
export const SelectValue = ({ children }: { children?: React.ReactNode }) => <>{children}</>
export const SelectContent = ({ children }: { children: React.ReactNode }) => <div className="py-1">{children}</div>

interface SelectItemProps {
  value: string
  children?: React.ReactNode
}
export const SelectItem = ({ value, children }: SelectItemProps) => (
  <button
    type="button"
    className="flex w-full cursor-pointer select-none items-center rounded-sm px-3 py-2 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground"
  >
    {children}
  </button>
)