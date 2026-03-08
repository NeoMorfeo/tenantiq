import { createFileRoute, Outlet, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth";
import { AppLayout } from "@/components/app-layout";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useTheme } from "next-themes";
import { getProfile } from "@/api/profile/profile";

export const Route = createFileRoute("/_authenticated")({
  component: AuthenticatedLayout,
});

function AuthenticatedLayout() {
  const { t, i18n } = useTranslation();
  const { isAuthenticated, loading } = useAuth();
  const navigate = useNavigate();
  const { setTheme } = useTheme();

  // Fetch profile on authentication and apply server-side preferences
  useEffect(() => {
    if (!isAuthenticated) return;
    let cancelled = false;

    getProfile().then((profile) => {
      if (cancelled) return;
      if (profile.theme) {
        setTheme(profile.theme);
      }
      if (profile.language && !i18n.language?.startsWith(profile.language)) {
        i18n.changeLanguage(profile.language);
      }
    }).catch(() => {});

    return () => { cancelled = true; };
  }, [isAuthenticated]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      navigate({ to: "/login" });
    }
  }, [loading, isAuthenticated, navigate]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-muted-foreground">{t("common.loading")}</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return (
    <AppLayout>
      <Outlet />
    </AppLayout>
  );
}
