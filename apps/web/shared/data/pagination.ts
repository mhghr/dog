export interface PaginationState {
  page: number;
  pageSize: number;
}

export interface Page<T> {
  items: T[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export function toPaginationQuery(state: PaginationState): URLSearchParams {
  const query = new URLSearchParams();
  query.set("page", String(state.page));
  query.set("page_size", String(state.pageSize));
  return query;
}

export function toPage<T>(
  items: T[],
  state: PaginationState,
  total: number,
): Page<T> {
  return {
    items,
    page: state.page,
    pageSize: state.pageSize,
    total,
    totalPages: Math.max(1, Math.ceil(total / state.pageSize)),
  };
}
