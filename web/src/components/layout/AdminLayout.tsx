import { useState } from "react"
import { Link, useLocation, useNavigate } from "react-router-dom"
import { FileText, Globe, Home, LayoutDashboard, LogOut, Menu, PanelLeft, Shield } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet"
import { Separator } from "@/components/ui/separator"
import { useAuth } from "@/hooks/useAuth"
import { logout } from "@/api/admin"

const navItems = [
  { path: "/admin/stats", label: "首页", icon: Home },
  { path: "/admin/domains", label: "域名映射", icon: Globe },
  { path: "/admin/logs", label: "访问日志", icon: LayoutDashboard },
  { path: "/admin/blocks", label: "拦截日志", icon: FileText },
]

export function AdminLayout({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const { logout: localLogout } = useAuth()
  const [open, setOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(false)

  const handleLogout = async () => {
    try {
      await logout()
    } finally {
      localLogout()
      navigate("/login")
    }
  }

  const NavLinks = ({ mobile = false }: { mobile?: boolean }) => (
    <>
      <div className={`flex items-center gap-2 ${mobile || !collapsed ? "px-4 py-5" : "justify-center px-2 py-5"}`}>
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Shield className="h-5 w-5" />
        </div>
        {(mobile || !collapsed) && <span className="text-lg font-bold tracking-tight text-foreground">SafeGate</span>}
      </div>
      <Separator className="bg-border/60" />
      <nav className={`flex ${mobile ? "flex-col gap-2 px-3 py-4" : `flex-col gap-1 px-2 py-4 ${collapsed ? "items-center" : ""}`}`}>
        {navItems.map((item) => {
          const Icon = item.icon
          const active = location.pathname === item.path
          return (
            <Link
              key={item.path}
              to={item.path}
              title={collapsed && !mobile ? item.label : undefined}
              onClick={() => setOpen(false)}
              className={`flex items-center gap-3 rounded-md py-2.5 text-sm font-medium transition-colors ${
                mobile || !collapsed ? "px-3" : "justify-center px-2"
              } ${
                active
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              }`}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {(mobile || !collapsed) && <span>{item.label}</span>}
            </Link>
          )
        })}
      </nav>
      <div className="mt-auto px-3 py-4">
        <Button
          variant="ghost"
          className={`justify-start gap-2 text-muted-foreground hover:text-foreground ${collapsed ? "w-full px-2" : "w-full"}`}
          onClick={handleLogout}
        >
          <LogOut className="h-4 w-4 shrink-0" />
          {(collapsed) ? null : <span>退出登录</span>}
        </Button>
      </div>
    </>
  )

  return (
    <div className="flex min-h-screen bg-background">
      {/* Desktop sidebar */}
      <aside
        className={`hidden flex-col border-r border-border bg-card shadow-sm transition-all duration-300 lg:flex ${
          collapsed ? "w-16" : "w-64"
        }`}
      >
        <NavLinks />
        <div className="px-2 pb-3">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-center text-muted-foreground hover:text-foreground"
            onClick={() => setCollapsed((v) => !v)}
          >
            <PanelLeft className={`h-4 w-4 transition-transform ${collapsed ? "rotate-180" : ""}`} />
          </Button>
        </div>
      </aside>

      {/* Mobile header */}
      <div className="flex flex-1 flex-col">
        <header className="flex items-center justify-between border-b border-border bg-card p-4 lg:hidden">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Shield className="h-4 w-4" />
            </div>
            <span className="text-lg font-bold tracking-tight">SafeGate</span>
          </div>
          <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger asChild>
              <Button variant="ghost" size="icon">
                <Menu className="h-5 w-5" />
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="flex w-64 flex-col border-r border-border p-0">
              <NavLinks mobile />
            </SheetContent>
          </Sheet>
        </header>

        <main className="flex-1 overflow-auto p-4 lg:p-8">{children}</main>
      </div>
    </div>
  )
}
