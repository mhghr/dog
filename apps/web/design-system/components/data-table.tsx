"use client";

import type { ReactNode } from "react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui/table";

export interface DataTableColumn<T> {
  key: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  className?: string;
}

export interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  empty?: ReactNode;
}

// Design-system DataTable: a typed generic table. Domain tables (resources,
// monitors, agents) supply columns and rows; pagination and sorting live in
// the consuming feature.
export function DataTable<T>({ columns, rows, rowKey, empty }: DataTableProps<T>) {
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((col) => (
              <TableHead key={col.key} className={col.className}>
                {col.header}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell colSpan={columns.length} className="py-12 text-center text-sm text-muted-foreground">
                {empty ?? "No data"}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={rowKey(row)}>
                {columns.map((col) => (
                  <TableCell key={col.key} className={col.className}>
                    {col.cell(row)}
                  </TableCell>
                ))}
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}
