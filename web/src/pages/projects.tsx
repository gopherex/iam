import { getMgmtV1Projects, postMgmtV1Projects } from '@gopherex/iam-sdk';
import { Boxes, Loader2, Plus } from 'lucide-react';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CopyButton } from '@/components/copy-button';
import { PageHeader } from '@/components/page-header';
import { EmptyState, ErrorState, LoadingState } from '@/components/states';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';

export function ProjectsPage() {
  const navigate = useNavigate();
  const { data, loading, error, reload } = useApi(() => call(getMgmtV1Projects()), []);
  const projects = data?.data ?? [];

  return (
    <div>
      <PageHeader
        title="Projects"
        description="Every tenant the operator manages. Open one to administer its identities."
        actions={<NewProjectDialog onCreated={reload} />}
      />
      {loading && <LoadingState />}
      {error && <ErrorState error={error} onRetry={reload} />}
      {!loading && !error && projects.length === 0 && (
        <EmptyState
          title="No projects yet"
          description="Create your first project to get started."
          action={<NewProjectDialog onCreated={reload} />}
        />
      )}
      {!loading && !error && projects.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((p) => (
            <Card
              key={p.id}
              className="cursor-pointer transition-colors hover:border-primary/40"
              onClick={() => navigate(`/projects/${p.id}`)}
            >
              <CardHeader>
                <div className="flex items-center justify-between gap-2">
                  <div className="flex size-9 items-center justify-center rounded-lg bg-muted">
                    <Boxes className="size-4.5" />
                  </div>
                  <Badge variant="secondary">{(p.environments?.length ?? 0)} env</Badge>
                </div>
                <CardTitle className="pt-2">{p.name ?? p.slug ?? p.id}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                {p.slug && <p className="text-sm text-muted-foreground">/{p.slug}</p>}
                {p.id && (
                  <div onClick={(e) => e.stopPropagation()}>
                    <CopyButton value={p.id} />
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function NewProjectDialog({ onCreated }: { onCreated: () => void }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [locale, setLocale] = useState('en');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await call(
        postMgmtV1Projects({
          body: { name, slug: slug || undefined, default_locale: locale || undefined },
        }),
      );
      setOpen(false);
      setName('');
      setSlug('');
      onCreated();
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Failed to create project');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <Plus className="size-4" />
        New project
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New project</DialogTitle>
          <DialogDescription>A project is an isolated tenant with its own users and config.</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="p-name">Name</Label>
            <Input id="p-name" value={name} onChange={(e) => setName(e.target.value)} required autoFocus />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="p-slug">Slug</Label>
              <Input id="p-slug" value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="optional" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="p-locale">Default locale</Label>
              <Input id="p-locale" value={locale} onChange={(e) => setLocale(e.target.value)} />
            </div>
          </div>
          {err && <p className="text-sm text-destructive">{err}</p>}
          <DialogFooter>
            <Button type="submit" disabled={busy || !name}>
              {busy && <Loader2 className="size-4 animate-spin" />}
              Create project
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
