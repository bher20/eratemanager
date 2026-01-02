import { NavLink, useLocation } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { StatusIndicator } from '@/components'
import { useAuth } from '@/context/AuthContext'
import {
  Zap,
  Droplets,
  LayoutDashboard,
  Settings,
  Moon,
  Sun,
  Menu,
  Key,
  LogOut,
  ChevronDown,
  ChevronRight,
  User,
  Shield,
  Lock
} from 'lucide-react'
import { useState, useEffect } from 'react'

interface LayoutProps {
  children: React.ReactNode
}

type NavItem = {
  name: string
  href?: string
  icon?: any
  children?: NavItem[]
}

const navigation: NavItem[] = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard },
  { name: 'Electric Rates', href: '/electric', icon: Zap },
  { name: 'Water Rates', href: '/water', icon: Droplets },
  { 
    name: 'Settings', 
    icon: Settings,
    children: [
      { name: 'General', href: '/settings/general', icon: Settings },
      { name: 'Users', href: '/settings/users', icon: User },
      { 
        name: 'RBAC', 
        icon: Shield,
        children: [
          { name: 'Roles', href: '/settings/rbac/roles', icon: Shield },
          { name: 'Privileges', href: '/settings/rbac/privileges', icon: Lock }
        ]
      }
    ]
  },
]

const SidebarItem = ({ item, depth = 0, setSidebarOpen }: { item: NavItem, depth?: number, setSidebarOpen: (open: boolean) => void }) => {
  const location = useLocation()
  const [isOpen, setIsOpen] = useState(false)
  
  useEffect(() => {
    const isChildActive = (item: NavItem): boolean => {
      if (item.href && location.pathname === item.href) return true
      if (item.children) return item.children.some(isChildActive)
      return false
    }
    if (isChildActive(item)) {
      setIsOpen(true)
    }
  }, [location.pathname, item])

  const hasChildren = item.children && item.children.length > 0
  const Icon = item.icon
  const paddingLeft = depth === 0 ? undefined : `${depth * 1.5 + 0.75}rem`

  if (hasChildren) {
    return (
      <div className="space-y-1">
        <button
          onClick={() => setIsOpen(!isOpen)}
          className={cn(
            'flex w-full items-center justify-between gap-3 rounded-lg py-2.5 text-sm font-medium transition-all duration-200 text-muted-foreground hover:bg-muted hover:text-foreground',
            depth === 0 ? 'px-3' : 'pr-3'
          )}
          style={{ paddingLeft }}
        >
          <div className="flex items-center gap-3">
            {Icon && <Icon className="h-5 w-5" />}
            {item.name}
          </div>
          {isOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </button>
        
        {isOpen && (
          <div className="space-y-1">
            {item.children!.map((child) => (
              <SidebarItem key={child.name} item={child} depth={depth + 1} setSidebarOpen={setSidebarOpen} />
            ))}
          </div>
        )}
      </div>
    )
  }

  return (
    <NavLink
      to={item.href!}
      onClick={() => setSidebarOpen(false)}
      className={({ isActive }) =>
        cn(
          'flex items-center gap-3 rounded-lg py-2.5 text-sm font-medium transition-all duration-200',
          isActive
            ? 'bg-primary/10 text-primary'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground',
          depth === 0 ? 'px-3' : 'pr-3'
        )
      }
      style={{ paddingLeft }}
    >
      {Icon && <Icon className="h-5 w-5" />}
      {item.name}
    </NavLink>
  )
}

export function Layout({ children }: LayoutProps) {
  const [theme, setTheme] = useState<'dark' | 'light'>('dark')
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const { logout, user } = useAuth()

  useEffect(() => {
    document.documentElement.classList.toggle('light', theme === 'light')
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }, [theme])

  const toggleTheme = () => {
    setTheme((prev) => (prev === 'dark' ? 'light' : 'dark'))
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Mobile sidebar backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-64 transform bg-card border-r border-border transition-transform duration-300 ease-in-out lg:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center gap-3 border-b border-border px-6">
            <div className="relative h-9 w-9 overflow-hidden rounded-xl bg-gradient-to-br from-yellow-400 via-red-500 to-blue-500 shadow-lg">
              <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent" />
            </div>
            <div>
              <h1 className="font-semibold tracking-tight">eRateManager</h1>
              <p className="text-xs text-muted-foreground">Utility Rate Tracker</p>
            </div>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 px-3 py-4 overflow-y-auto">
            {navigation.map((item) => (
              <SidebarItem key={item.name} item={item} setSidebarOpen={setSidebarOpen} />
            ))}
          </nav>

          {/* Footer */}
          <div className="border-t border-border p-4 space-y-4">
            <div className="relative">
              <button 
                onClick={() => setUserMenuOpen(!userMenuOpen)}
                className="flex w-full items-center justify-between rounded-lg p-2 hover:bg-muted transition-colors"
              >
                <div className="flex items-center gap-2">
                  <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center text-primary font-medium">
                    {user?.username.charAt(0).toUpperCase()}
                  </div>
                  <div className="text-sm text-left">
                    <p className="font-medium">{user?.username}</p>
                    <p className="text-xs text-muted-foreground capitalize">{user?.role}</p>
                  </div>
                </div>
                <ChevronDown className={cn("h-4 w-4 transition-transform", userMenuOpen && "rotate-180")} />
              </button>

              {userMenuOpen && (
                <div className="absolute bottom-full left-0 w-full mb-2 rounded-lg border border-border bg-card shadow-lg overflow-hidden animate-in slide-in-from-bottom-2">
                  <div className="p-1">
                    <NavLink 
                      to="/profile"
                      onClick={() => {
                        setUserMenuOpen(false)
                        setSidebarOpen(false)
                      }}
                      className="flex items-center gap-2 rounded-md px-3 py-2 text-sm hover:bg-muted transition-colors"
                    >
                      <User className="h-4 w-4" />
                      Profile
                    </NavLink>
                    <NavLink 
                      to="/tokens"
                      onClick={() => {
                        setUserMenuOpen(false)
                        setSidebarOpen(false)
                      }}
                      className="flex items-center gap-2 rounded-md px-3 py-2 text-sm hover:bg-muted transition-colors"
                    >
                      <Key className="h-4 w-4" />
                      API Tokens
                    </NavLink>
                    <div className="my-1 border-t border-border" />
                    <button
                      onClick={() => {
                        logout()
                        setUserMenuOpen(false)
                      }}
                      className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-destructive hover:bg-destructive/10 transition-colors"
                    >
                      <LogOut className="h-4 w-4" />
                      Logout
                    </button>
                  </div>
                </div>
              )}
            </div>

            <div className="flex items-center justify-between">
              <StatusIndicator status="online" label="API Online" />
              <button
                onClick={toggleTheme}
                className="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              >
                {theme === 'dark' ? (
                  <Sun className="h-5 w-5" />
                ) : (
                  <Moon className="h-5 w-5" />
                )}
              </button>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="lg:pl-64">
        {/* Mobile header */}
        <header className="sticky top-0 z-30 flex h-16 items-center gap-4 border-b border-border bg-background/80 px-4 backdrop-blur-xl lg:hidden">
          <button
            onClick={() => setSidebarOpen(true)}
            className="rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground"
          >
            <Menu className="h-6 w-6" />
          </button>
          <h1 className="font-semibold">eRateManager</h1>
        </header>

        {/* Page content */}
        <main className="min-h-[calc(100vh-4rem)] p-6 lg:p-8">{children}</main>
      </div>
    </div>
  )
}
