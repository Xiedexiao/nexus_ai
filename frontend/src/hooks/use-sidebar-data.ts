/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  Activity,
  Box,
  ClipboardList,
  CreditCard,
  FileText,
  FlaskConical,
  Key,
  LayoutDashboard,
  ListTodo,
  MessageSquare,
  PlugZap,
  Radio,
  ServerCog,
  Settings,
  ShieldCheck,
  Ticket,
  User,
  Users,
  Wallet,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { createElement } from 'react'
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api'
import { CustomLinkIcon } from '@/components/custom-link-icon'

import type {
  NavGroup,
  NavItem,
  NavLink,
  SidebarData,
} from '@/components/layout/types'
import { ROLE } from '@/lib/roles'

/**
 * Root navigation groups for the application sidebar.
 *
 * These are shown when the URL does not match any nested sidebar view
 * registered in `layout/lib/sidebar-view-registry.ts`.
 */

type CustomLinkConfig = {
  id: string
  title: string
  icon?: string
  proxy: boolean
  enabled: boolean
  href?: string
}


// Creates a React component rendering the named Lucide icon without JSX
// (this module is .ts, so createElement is used instead of JSX syntax).
function makeCustomIconComponent(iconName?: string) {
  return function CustomLinkIconWrapper(props: { className?: string }) {
    return createElement(CustomLinkIcon, { name: iconName, ...props })
  }
}

function useCustomLinks(): CustomLinkConfig[] {
  const { data } = useQuery({
    queryKey: ['custom-links'],
    queryFn: async () => {
      const res = await api.get<{ data?: CustomLinkConfig[] }>('/api/custom_links')
      return res.data?.data ?? []
    },
    staleTime: 5 * 60 * 1000,
  })
  return (data ?? []).filter((link) => link.enabled)
}

export function useSidebarData(): SidebarData {
  const { t } = useTranslation()
  const customLinks = useCustomLinks()

  const customLinkGroups: NavGroup[] =
    customLinks.length > 0
      ? [
          {
            id: 'custom-links',
            title: t('Links'),
            items: customLinks.map(
              (link: CustomLinkConfig): NavItem => ({
                title: link.title,
                url: link.proxy ? (`/ext/${link.id}/` as NavLink['url']) : (link.href ?? link.id),
                icon: makeCustomIconComponent(link.icon),
                external: !link.proxy,
              })
            ),
          },
        ]
      : []

  return {
    navGroups: [
      {
        id: 'chat',
        title: t('Chat'),
        items: [
          {
            title: t('Playground'),
            url: '/playground',
            icon: FlaskConical,
          },
          {
            title: t('Chat'),
            icon: MessageSquare,
            type: 'chat-presets',
          },
        ],
      },
      {
        id: 'general',
        title: t('General'),
        items: [
          {
            title: t('Overview'),
            url: '/dashboard/overview',
            icon: Activity,
          },
          {
            title: t('Dashboard'),
            url: '/dashboard/models',
            icon: LayoutDashboard,
          },
          {
            title: t('API Keys'),
            url: '/keys',
            icon: Key,
          },
          {
            title: t('Usage Logs'),
            url: '/usage-logs/common',
            icon: FileText,
          },
          {
            title: t('Audit Logs'),
            url: '/usage-logs/audit',
            icon: ClipboardList,
          },
          {
            title: t('Task Logs'),
            url: '/usage-logs/task',
            activeUrls: ['/usage-logs/drawing'],
            configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
            icon: ListTodo,
          },
        ],
      },
      {
        id: 'personal',
        title: t('Personal'),
        items: [
          {
            title: t('Wallet'),
            url: '/wallet',
            icon: Wallet,
          },
          {
            title: t('Profile'),
            url: '/profile',
            icon: User,
          },
          {
            title: t('Security & Access'),
            url: '/security',
            icon: ShieldCheck,
          },
        ],
      },
      {
        id: 'admin',
        title: t('Admin'),
        items: [
          {
            title: t('Channels'),
            url: '/channels',
            icon: Radio,
          },
          {
            title: t('Models'),
            url: '/models/metadata',
            icon: Box,
          },
          {
            title: t('Users'),
            url: '/users',
            icon: Users,
          },
          {
            title: t('Redemption Codes'),
            url: '/redemption-codes',
            icon: Ticket,
          },
          {
            title: t('Subscriptions'),
            url: '/subscriptions',
            icon: CreditCard,
          },
          {
            title: t('System Info'),
            url: '/system-info',
            icon: ServerCog,
            requiredRole: ROLE.SUPER_ADMIN,
          },
          {
            title: t('Task Plugins'),
            url: '/task-plugins',
            icon: PlugZap,
            requiredRole: ROLE.SUPER_ADMIN,
          },
          {
            title: t('System Settings'),
            url: '/system-settings/site',
            activeUrls: ['/system-settings'],
            icon: Settings,
            requiredRole: ROLE.SUPER_ADMIN,
          },
        ],
      },
      ...customLinkGroups,
    ],
  }
}
