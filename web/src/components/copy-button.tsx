import { Check, Copy } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

/** Inline code value with a copy affordance — for IDs, tokens, keys. */
export function CopyButton({ value, className }: { value: string; className?: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      onClick={() => {
        void navigator.clipboard.writeText(value);
        setCopied(true);
        setTimeout(() => setCopied(false), 1200);
      }}
      className={cn(
        'inline-flex max-w-full items-center gap-1.5 rounded-sm bg-muted px-1.5 py-0.5 font-mono text-xs text-muted-foreground transition-colors hover:text-foreground',
        className,
      )}
      title="Copy"
    >
      <span className="truncate">{value}</span>
      {copied ? <Check className="size-3 shrink-0" /> : <Copy className="size-3 shrink-0 opacity-60" />}
    </button>
  );
}

/** A bare icon-only copy button. */
export function CopyIconButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <Button
      variant="ghost"
      size="icon"
      className="size-7"
      onClick={() => {
        void navigator.clipboard.writeText(value);
        setCopied(true);
        setTimeout(() => setCopied(false), 1200);
      }}
      title="Copy"
    >
      {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
    </Button>
  );
}
