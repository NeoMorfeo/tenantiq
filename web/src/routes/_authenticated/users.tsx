import { useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth";
import { useListUsers, useUpdateUser } from "@/api/users/users";
import type { UserResponse } from "@/api/model";
import type { UpdateUserInputBodyRole } from "@/api/model/updateUserInputBodyRole";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  type ColumnDef,
  type SortingState,
  flexRender,
} from "@tanstack/react-table";
import { useQueryClient } from "@tanstack/react-query";
import { getListUsersQueryKey } from "@/api/users/users";
import { Button } from "@/components/ui/button";
import { EditableCell } from "@/components/editable-cell";
import { RoleSelect } from "@/components/role-select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowUpDown } from "lucide-react";
import { CreateUserDialog } from "@/components/users/create-user-dialog";
import { DeleteUserDialog } from "@/components/users/delete-user-dialog";
import { useTranslation } from "react-i18next";

const UsersPage = () => {
  const { t } = useTranslation();
  const { user: currentUser } = useAuth();
  const { data: users, isLoading } = useListUsers();
  const queryClient = useQueryClient();
  const updateMutation = useUpdateUser();

  const [sorting, setSorting] = useState<SortingState>([]);

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: getListUsersQueryKey() });

  const handleUpdate = (id: string, data: { name?: string; role?: UpdateUserInputBodyRole }) => {
    updateMutation.mutate({ id, data }, { onSuccess: invalidate });
  };

  const columns: ColumnDef<UserResponse>[] = [
    {
      accessorKey: "name",
      header: ({ column }) => (
        <Button variant="ghost" size="sm" className="-ml-3" onClick={() => column.toggleSorting()}>
          {t("common.name")} <ArrowUpDown className="ml-1 h-3 w-3" />
        </Button>
      ),
      cell: ({ row }) => (
        <EditableCell
          value={row.original.name ?? ""}
          onSave={(name) => handleUpdate(row.original.id!, { name })}
        />
      ),
    },
    {
      accessorKey: "email",
      header: t("common.email"),
      cell: ({ getValue }) => (
        <span className="text-muted-foreground">{getValue<string>()}</span>
      ),
    },
    {
      accessorKey: "role",
      header: t("common.role"),
      cell: ({ row }) => (
        <RoleSelect
          value={row.original.role ?? "viewer"}
          onSave={(role) => handleUpdate(row.original.id!, { role: role as UpdateUserInputBodyRole })}
        />
      ),
    },
    {
      accessorKey: "created_at",
      header: ({ column }) => (
        <Button variant="ghost" size="sm" className="-ml-3" onClick={() => column.toggleSorting()}>
          {t("common.created")} <ArrowUpDown className="ml-1 h-3 w-3" />
        </Button>
      ),
      cell: ({ getValue }) => {
        const v = getValue<string>();
        return (
          <span className="text-muted-foreground">
            {v ? new Date(v).toLocaleDateString() : "-"}
          </span>
        );
      },
    },
    {
      id: "actions",
      cell: ({ row }) =>
        row.original.id !== currentUser?.user_id ? (
          <DeleteUserDialog user={row.original} />
        ) : null,
    },
  ];

  const table = useReactTable({
    data: users ?? [],
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t("users.title")}</h1>
          <p className="text-muted-foreground">{t("users.description")}</p>
        </div>
        <CreateUserDialog />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("users.allUsers")}</CardTitle>
          <CardDescription>
            {users ? t("users.count", { count: users.length }) : t("common.loading")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                {table.getHeaderGroups().map((hg) => (
                  <TableRow key={hg.id}>
                    {hg.headers.map((header) => (
                      <TableHead key={header.id}>
                        {header.isPlaceholder
                          ? null
                          : flexRender(header.column.columnDef.header, header.getContext())}
                      </TableHead>
                    ))}
                  </TableRow>
                ))}
              </TableHeader>
              <TableBody>
                {table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
                {users?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                      {t("users.empty")}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export const Route = createFileRoute("/_authenticated/users")({
  component: UsersPage,
});
