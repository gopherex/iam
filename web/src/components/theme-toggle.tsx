import { useStore } from '@nanostores/react';
import { Moon, Sun } from 'lucide-react';
import { $theme, toggleTheme } from '@/stores/theme';
import { Button } from '@/components/ui/button';

export function ThemeToggle() {
  const theme = useStore($theme);
  return (
    <Button variant="ghost" size="icon" onClick={toggleTheme} aria-label="Toggle theme">
      {theme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
    </Button>
  );
}
