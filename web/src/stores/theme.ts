import { atom } from 'nanostores';

export type Theme = 'light' | 'dark';

const KEY = 'iam.theme';

function initial(): Theme {
  const saved = localStorage.getItem(KEY) as Theme | null;
  if (saved === 'light' || saved === 'dark') return saved;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export const $theme = atom<Theme>(initial());

function apply(theme: Theme): void {
  document.documentElement.classList.toggle('dark', theme === 'dark');
  document.documentElement.style.colorScheme = theme;
}

export function setTheme(theme: Theme): void {
  $theme.set(theme);
  localStorage.setItem(KEY, theme);
  apply(theme);
}

export function toggleTheme(): void {
  setTheme($theme.get() === 'dark' ? 'light' : 'dark');
}

// Apply on module load (before first paint).
apply($theme.get());
