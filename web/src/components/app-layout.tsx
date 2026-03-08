import type { ReactNode } from "react";
import { Link, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { ThemeToggle } from "@/components/theme-toggle";
import { LanguageSwitcher } from "@/components/language-switcher";
import { useTranslation } from "react-i18next";
import iconSvg from "@/assets/icon.svg";

export const AppLayout = ({ children }: { children: ReactNode }) => {
  const { t } = useTranslation();
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate({ to: "/login" });
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <div className="flex items-center gap-4">
            <Link to="/" className="flex items-center gap-2">
              <img src={iconSvg} alt="tenantiq" className="h-8 w-8" />
              <span className="text-lg font-semibold tracking-tight">
                tenant<span className="text-primary">iq</span>
              </span>
            </Link>
            {user?.role === "superadmin" && (
              <nav className="flex items-center gap-1">
                <Button variant="ghost" size="sm" render={<Link to="/users" />}>
                  {t("nav.users")}
                </Button>
              </nav>
            )}
          </div>
          <div className="flex items-center gap-1">
            <LanguageSwitcher />
            <ThemeToggle />
            <Button variant="ghost" size="sm" onClick={handleLogout}>
              {t("nav.signOut")}
            </Button>
          </div>
        </div>
      </header>
      <Separator />
      <main className="mx-auto max-w-7xl px-4 py-6">{children}</main>
    </div>
  );
};
