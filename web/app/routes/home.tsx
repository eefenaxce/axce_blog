import { AnimatedPage } from "~/components/animate"
import { useSiteSettings } from "~/components/settings-context"

export default function Home() {
  const { settings, loading } = useSiteSettings()

  return (
    <AnimatedPage>
      <div className="flex min-h-svh flex-col p-6">
        <div className="flex-1">
          <div className="flex max-w-md min-w-0 flex-col gap-4 text-sm leading-loose">
            <div>
              <h1 className="font-medium">Project ready!</h1>
              <p>You may now add components and start building.</p>
            </div>
          </div>
        </div>
        <footer className="mt-auto pt-8 text-center text-sm text-muted-foreground">
          <p>
            {loading ? (
              <span className="inline-block h-4 w-32 rounded bg-muted animate-pulse" />
            ) : (
              settings.site_copyright
            )}
          </p>
        </footer>
      </div>
    </AnimatedPage>
  )
}
