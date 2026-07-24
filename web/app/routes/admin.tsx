import { AdminLayout } from "~/components/admin/layout"
import { AdminUserProvider } from "~/components/admin/user-context"

export default function Admin() {
  return (
    <AdminUserProvider>
      <AdminLayout />
    </AdminUserProvider>
  )
}
