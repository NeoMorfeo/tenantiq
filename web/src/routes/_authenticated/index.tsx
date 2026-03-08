import { createFileRoute } from "@tanstack/react-router";
import { useListTenants } from "@/api/tenants/tenants";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { useTranslation } from "react-i18next";

export const Route = createFileRoute("/_authenticated/")({
  component: DashboardPage,
});

const statusVariant: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  active: "default",
  creating: "secondary",
  suspended: "destructive",
  deleting: "outline",
  deleted: "outline",
};

function DashboardPage() {
  const { t } = useTranslation();
  const { data: tenants, isLoading } = useListTenants();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t("tenants.title")}</h1>
          <p className="text-muted-foreground">
            {t("tenants.description")}
          </p>
        </div>
        <Button>{t("tenants.createTenant")}</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("tenants.allTenants")}</CardTitle>
          <CardDescription>
            {tenants ? t("tenants.count", { count: tenants.length }) : t("common.loading")}
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
                <TableRow>
                  <TableHead>{t("common.name")}</TableHead>
                  <TableHead>{t("tenants.slug")}</TableHead>
                  <TableHead>{t("tenants.plan")}</TableHead>
                  <TableHead>{t("tenants.status")}</TableHead>
                  <TableHead>{t("common.created")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tenants?.map((tenant) => (
                  <TableRow key={tenant.id}>
                    <TableCell className="font-medium">
                      {tenant.name}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {tenant.slug}
                    </TableCell>
                    <TableCell>{tenant.plan}</TableCell>
                    <TableCell>
                      <Badge variant={statusVariant[tenant.status ?? ""] ?? "outline"}>
                        {tenant.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {tenant.created_at
                        ? new Date(tenant.created_at).toLocaleDateString()
                        : "-"}
                    </TableCell>
                  </TableRow>
                ))}
                {tenants?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                      {t("tenants.empty")}
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
}
