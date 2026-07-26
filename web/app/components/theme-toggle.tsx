import { Moon, Sun } from "lucide-react"
import { Button } from "./ui/button"
import { useTheme } from "./theme-provider"

interface ThemeToggleProps {
  variant?: "ghost" | "outline" | "default"
  size?: "icon" | "sm" | "default" | "lg"
  className?: string
}

export function ThemeToggle({
  variant = "ghost",
  size = "icon",
  className = "",
}: ThemeToggleProps) {
  const { resolvedTheme, toggleTheme } = useTheme()

  return (
    <Button
      variant={variant}
      size={size}
      className={className}
      onClick={toggleTheme}
      title={resolvedTheme === "dark" ? "切换浅色模式" : "切换深色模式"}
    >
      {resolvedTheme === "dark" ? (
        <Sun className="h-4 w-4" />
      ) : (
        <Moon className="h-4 w-4" />
      )}
    </Button>
  )
}
