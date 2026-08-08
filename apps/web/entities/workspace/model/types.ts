export interface Project {
  id: string;
  organization_id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface Workspace {
  id: string;
  organization_id: string;
  name: string;
  slug: string;
  description: string;
  created_at: string;
  updated_at: string;
}
