import { useStore } from '@nanostores/react';
import { getMgmtV1Projects } from '@gopherex/iam-sdk';
import { ChevronsUpDown, FolderKanban } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { call } from '@/lib/sdk';
import { useApi } from '@/lib/use-api';
import { $project } from '@/stores/auth';

export function ProjectSwitcher() {
  const project = useStore($project);
  const navigate = useNavigate();
  const { data } = useApi(() => call(getMgmtV1Projects()), []);
  const projects = data?.data ?? [];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="outline" size="sm" className="gap-2" />}>
        <FolderKanban className="size-4" />
        <span className="max-w-[12rem] truncate">{project?.name ?? 'Select project'}</span>
        <ChevronsUpDown className="size-3.5 opacity-50" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuLabel>Switch project</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {projects.map((p) => (
          <DropdownMenuItem key={p.id} onClick={() => navigate(`/projects/${p.id}`)}>
            <FolderKanban className="size-4" />
            <span className="truncate">{p.name ?? p.id}</span>
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => navigate('/projects')}>All projects…</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
