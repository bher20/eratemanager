import { useState } from 'react'
import { GeneralSettingsPage } from './GeneralSettingsPage'
import { EmailSettingsPage } from './EmailSettingsPage'
import { UsersPage } from './UsersPage'
import { RolesPage } from './RolesPage'
import { cn } from '@/lib/utils'

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState('general')

  const tabs = [
    { id: 'general', label: 'General', component: GeneralSettingsPage },
    { id: 'email', label: 'Email', component: EmailSettingsPage },
    { id: 'users', label: 'Users', component: UsersPage },
    { id: 'roles', label: 'Roles', component: RolesPage },
  ]

  return (
    <div className="space-y-6">
      <div className="border-b border-border">
        <nav className="-mb-px flex space-x-8" aria-label="Tabs">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium transition-colors',
                activeTab === tab.id
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground'
              )}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      <div className="mt-6">
        {tabs.map((tab) => (
          <div key={tab.id} className={cn(activeTab === tab.id ? 'block' : 'hidden')}>
            <tab.component />
          </div>
        ))}
      </div>
    </div>
  )
}
