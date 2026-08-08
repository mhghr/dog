export interface FilterState {
  search?: string;
  status?: string;
  type?: string;
}

export function toFilterQuery(filters: FilterState): URLSearchParams {
  const query = new URLSearchParams();
  if (filters.search) query.set("search", filters.search);
  if (filters.status) query.set("status", filters.status);
  if (filters.type) query.set("type", filters.type);
  return query;
}

export function isFiltered(filters: FilterState): boolean {
  return Boolean(filters.search || filters.status || filters.type);
}
