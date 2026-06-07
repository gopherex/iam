import { getMgmtV1Projects } from '@gopherex/iam-sdk';
import { KeyRound, Loader2, ShieldCheck } from 'lucide-react';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { call } from '@/lib/sdk';
import { setMasterKey } from '@/stores/auth';

export function LoginPage() {
  const navigate = useNavigate();
  const [key, setKey] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!key.trim()) return;
    setLoading(true);
    setError(null);
    try {
      // Validate the master key by listing projects with it.
      await call(getMgmtV1Projects({ headers: { Authorization: `Bearer ${key.trim()}` } }));
      setMasterKey(key.trim());
      navigate('/projects', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid master key');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="relative flex min-h-svh items-center justify-center bg-muted/30 p-4">
      <div className="absolute right-4 top-4">
        <ThemeToggle />
      </div>
      <Card className="w-full max-w-sm">
        <CardHeader className="items-center text-center">
          <div className="mb-2 flex size-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <ShieldCheck className="size-6" />
          </div>
          <CardTitle className="font-heading text-xl">IAM Console</CardTitle>
          <CardDescription>Sign in with the operator master key.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="master-key">Master key</Label>
              <div className="relative">
                <KeyRound className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="master-key"
                  type="password"
                  autoFocus
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  placeholder="••••••••••••"
                  className="pl-8 font-mono"
                />
              </div>
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit" className="w-full" disabled={loading || !key.trim()}>
              {loading && <Loader2 className="size-4 animate-spin" />}
              Sign in
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
