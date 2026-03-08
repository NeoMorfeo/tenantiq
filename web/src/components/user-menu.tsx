import { useNavigate } from "@tanstack/react-router";
import { useTheme } from "next-themes";
import { useTranslation } from "react-i18next";
import { User, Users, Sun, Moon, Monitor, Languages, LogOut } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { useUpdatePreferences } from "@/api/profile/profile";
import type {
  UpdatePreferencesInputBodyTheme,
  UpdatePreferencesInputBodyLanguage,
} from "@/api/model";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const THEMES = [
  { value: "light", icon: Sun, label: "theme.light" },
  { value: "dark", icon: Moon, label: "theme.dark" },
  { value: "system", icon: Monitor, label: "theme.system" },
] as const;

const LANGUAGES = [
  { value: "en", label: "language.en" },
  { value: "es", label: "language.es" },
] as const;

export const UserMenu = () => {
  const { t, i18n } = useTranslation();
  const { user, logout } = useAuth();
  const { theme, setTheme } = useTheme();
  const navigate = useNavigate();

  const prefsMutation = useUpdatePreferences();

  const initials = user?.email?.slice(0, 2).toUpperCase() ?? "?";

  const handleTheme = (value: string) => {
    setTheme(value);
    prefsMutation.mutate({ data: { theme: value as UpdatePreferencesInputBodyTheme } });
  };

  const handleLanguage = (value: string) => {
    i18n.changeLanguage(value);
    prefsMutation.mutate({ data: { language: value as UpdatePreferencesInputBodyLanguage } });
  };

  const handleLogout = async () => {
    await logout();
    navigate({ to: "/login" });
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="cursor-pointer rounded-full outline-none focus-visible:ring-2 focus-visible:ring-ring">
        <Avatar size="sm">
          <AvatarFallback className="text-[10px]">{initials}</AvatarFallback>
        </Avatar>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuGroup>
          <DropdownMenuLabel>
            {user?.email}
          </DropdownMenuLabel>
          <DropdownMenuItem onClick={() => navigate({ to: "/profile" })}>
            <User className="mr-2 h-4 w-4" />
            {t("nav.profile")}
          </DropdownMenuItem>
        </DropdownMenuGroup>

        {user?.role === "superadmin" && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuLabel>
                {t("nav.settings")}
              </DropdownMenuLabel>
              <DropdownMenuItem onClick={() => navigate({ to: "/users" })}>
                <Users className="mr-2 h-4 w-4" />
                {t("nav.users")}
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </>
        )}

        <DropdownMenuSeparator />

        <DropdownMenuGroup>
          <DropdownMenuSub>
            <DropdownMenuSubTrigger>
              <Sun className="mr-2 h-4 w-4" />
              {t("nav.theme")}
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
              <DropdownMenuRadioGroup value={theme ?? "system"} onValueChange={handleTheme}>
                {THEMES.map(({ value, icon: Icon, label }) => (
                  <DropdownMenuRadioItem key={value} value={value}>
                    <Icon className="mr-2 h-4 w-4" />
                    {t(label)}
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuSubContent>
          </DropdownMenuSub>

          <DropdownMenuSub>
            <DropdownMenuSubTrigger>
              <Languages className="mr-2 h-4 w-4" />
              {t("nav.language")}
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
              <DropdownMenuRadioGroup value={i18n.language?.startsWith("es") ? "es" : "en"} onValueChange={handleLanguage}>
                {LANGUAGES.map(({ value, label }) => (
                  <DropdownMenuRadioItem key={value} value={value}>
                    {t(label)}
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuSubContent>
          </DropdownMenuSub>
        </DropdownMenuGroup>

        <DropdownMenuSeparator />

        <DropdownMenuGroup>
          <DropdownMenuItem variant="destructive" onClick={handleLogout}>
            <LogOut className="mr-2 h-4 w-4" />
            {t("nav.signOut")}
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
