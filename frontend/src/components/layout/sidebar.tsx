'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useState, useEffect } from 'react';
import {
  LayoutDashboard,
  ListTodo,
  PlusCircle,
  BarChart3,
  Sun,
  Moon,
  Menu,
  X,
  Zap,
  Circle,
} from 'lucide-react';
import { useTheme } from '@/components/theme-provider';
import { checkHealth } from '@/lib/api';

const navigation = [
  { name: 'Overview', href: '/', icon: LayoutDashboard },
  { name: 'Jobs', href: '/jobs', icon: ListTodo },
  { name: 'Create Job', href: '/create', icon: PlusCircle },
  { name: 'Statistics', href: '/statistics', icon: BarChart3 },
];

export function Sidebar() {
  const pathname = usePathname();
  const { theme, toggle } = useTheme();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [apiStatus, setApiStatus] = useState<'online' | 'offline' | 'checking'>('checking');

  useEffect(() => {
    const check = async () => {
      const healthy = await checkHealth();
      setApiStatus(healthy ? 'online' : 'offline');
    };
    check();
    const interval = setInterval(check, 30000);
    return () => clearInterval(interval);
  }, []);

  // Close mobile nav on route change
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  const isActive = (href: string) => {
    if (href === '/') return pathname === '/';
    return pathname.startsWith(href);
  };

  const NavContent = () => (
    <>
      {/* Logo */}
      <div className="px-4 py-5 flex items-center gap-2.5">
        <div className="w-7 h-7 rounded-lg bg-neutral-900 dark:bg-white flex items-center justify-center">
          <Zap size={14} className="text-white dark:text-neutral-900" />
        </div>
        <span className="text-sm font-semibold text-neutral-900 dark:text-neutral-100 tracking-tight">
          JobQueue
        </span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-2 space-y-0.5">
        {navigation.map((item) => {
          const active = isActive(item.href);
          return (
            <Link
              key={item.name}
              href={item.href}
              className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-[13px] font-medium transition-colors ${
                active
                  ? 'bg-neutral-100 dark:bg-neutral-800 text-neutral-900 dark:text-neutral-100'
                  : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-200 hover:bg-neutral-50 dark:hover:bg-neutral-800/50'
              }`}
            >
              <item.icon size={16} strokeWidth={active ? 2 : 1.5} />
              {item.name}
            </Link>
          );
        })}
      </nav>

      {/* Bottom section */}
      <div className="px-3 pb-4 space-y-2 border-t border-neutral-100 dark:border-neutral-800 pt-3">
        {/* Theme toggle */}
        <button
          onClick={toggle}
          className="flex items-center gap-2.5 w-full px-3 py-2 rounded-lg text-[13px] font-medium text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-200 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors"
          aria-label="Toggle theme"
        >
          {theme === 'dark' ? <Sun size={16} strokeWidth={1.5} /> : <Moon size={16} strokeWidth={1.5} />}
          {theme === 'dark' ? 'Light mode' : 'Dark mode'}
        </button>

        {/* API status */}
        <div className="flex items-center gap-2.5 px-3 py-2 text-[13px] text-neutral-400 dark:text-neutral-500">
          <Circle
            size={7}
            fill="currentColor"
            className={
              apiStatus === 'online'
                ? 'text-emerald-500'
                : apiStatus === 'offline'
                ? 'text-red-500'
                : 'text-amber-400 animate-pulse'
            }
          />
          <span>
            API {apiStatus === 'checking' ? '…' : apiStatus}
          </span>
        </div>
      </div>
    </>
  );

  return (
    <>
      {/* Mobile menu button */}
      <button
        onClick={() => setMobileOpen(!mobileOpen)}
        className="lg:hidden fixed top-4 left-4 z-50 p-2 rounded-lg bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 text-neutral-600 dark:text-neutral-400 shadow-sm"
        aria-label="Toggle navigation"
      >
        {mobileOpen ? <X size={18} /> : <Menu size={18} />}
      </button>

      {/* Mobile overlay */}
      {mobileOpen && (
        <div
          className="lg:hidden fixed inset-0 z-40 bg-black/30 dark:bg-black/50"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Sidebar — desktop */}
      <aside className="hidden lg:flex lg:flex-col lg:fixed lg:inset-y-0 lg:left-0 lg:w-56 bg-white dark:bg-neutral-950 border-r border-neutral-100 dark:border-neutral-800/80 z-30">
        <NavContent />
      </aside>

      {/* Sidebar — mobile */}
      <aside
        className={`lg:hidden fixed inset-y-0 left-0 z-40 w-64 bg-white dark:bg-neutral-950 border-r border-neutral-100 dark:border-neutral-800/80 flex flex-col transform transition-transform duration-200 ease-out ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <NavContent />
      </aside>
    </>
  );
}
