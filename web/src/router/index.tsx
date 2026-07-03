import { createBrowserRouter, Navigate, Outlet } from "react-router-dom"

import { AdminLayout } from "@/components/layout/AdminLayout"
import { LoginPage } from "@/pages/Login"
import { DomainsPage } from "@/pages/Domains"
import { RulesPage } from "@/pages/Rules"
import { LogsPage } from "@/pages/Logs"
import { StatsPage } from "@/pages/Stats"
import { BlockedLogsPage } from "@/pages/BlockedLogs"

function ProtectedRoute() {
  const token = localStorage.getItem("token")
  return token ? (
    <AdminLayout>
      <Outlet />
    </AdminLayout>
  ) : (
    <Navigate to="/login" replace />
  )
}

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/admin",
    element: <ProtectedRoute />,
    children: [
      { path: "domains", element: <DomainsPage /> },
      { path: "rules", element: <RulesPage /> },
      { path: "logs", element: <LogsPage /> },
      { path: "stats", element: <StatsPage /> },
      { path: "blocks", element: <BlockedLogsPage /> },
      { index: true, element: <Navigate to="stats" replace /> },
    ],
  },
  {
    path: "/",
    element: <Navigate to="/admin/stats" replace />,
  },
])
