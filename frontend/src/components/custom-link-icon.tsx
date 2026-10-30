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
import { DynamicIcon, iconNames } from 'lucide-react/dynamic'

type CustomLinkIconProps = {
  name?: string
  className?: string
}

type IconName = Parameters<typeof DynamicIcon>[0]['name']

export const CUSTOM_LINK_ICON_NAMES: readonly string[] = iconNames

/**
 * Renders any Lucide icon by its kebab-case name (e.g. "book", "graduation-cap").
 * Falls back to ExternalLink when the name is empty or unknown, so a typo can
 * never break the sidebar.
 */
export function CustomLinkIcon({ name, className }: CustomLinkIconProps) {
  return (
    <DynamicIcon
      name={(name || 'external-link') as IconName}
      fallback={undefined}
      className={className}
      size={16}
    />
  )
}
