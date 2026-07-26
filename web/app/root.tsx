import {
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
  isRouteErrorResponse,
} from "react-router"

import type { Route } from "./+types/root"
import "./app.css"
import { Toaster } from "sonner"
import { SettingsProvider, useSiteSettings } from "./components/settings-context"
import { ThemeProvider } from "./components/theme-provider"

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <SettingsProvider>
        <html lang="zh-CN">
          <head>
          <meta charSet="utf-8" />
          <meta name="viewport" content="width=device-width, initial-scale=1" />
          <Title />
          <MetaTags />
          <Meta />
          <Links />
        </head>
        <body>
          {children}
          <Toaster
            position="top-right"
            richColors
            closeButton
            toastOptions={{
              duration: 3000,
            }}
          />
          <ScrollRestoration />
          <Scripts />
        </body>
      </html>
    </SettingsProvider>
    </ThemeProvider>
  )
}

function Title() {
  const { settings } = useSiteSettings()
  return <title>{settings.site_title}</title>
}

function MetaTags() {
  const { settings } = useSiteSettings()

  return (
    <>
      <meta name="description" content={settings.site_description || "A high-performance blog system"} />
      <meta name="keywords" content={settings.site_keywords || "blog,go,react"} />
      <meta name="author" content={settings.site_author || "Axce"} />
      <meta name="copyright" content={settings.site_copyright || "All rights reserved"} />
    </>
  )
}

export default function App() {
  return <Outlet />
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let message = "Oops!"
  let details = "An unexpected error occurred."
  let stack: string | undefined

  if (isRouteErrorResponse(error)) {
    message = error.status === 404 ? "404" : "Error"
    details =
      error.status === 404
        ? "The requested page could not be found."
        : error.statusText || details
  } else if (import.meta.env.DEV && error && error instanceof Error) {
    details = error.message
    stack = error.stack
  }

  return (
    <main className="container mx-auto p-4 pt-16">
      <h1>{message}</h1>
      <p>{details}</p>
      {stack && (
        <pre className="w-full overflow-x-auto p-4">
          <code>{stack}</code>
        </pre>
      )}
    </main>
  )
}
