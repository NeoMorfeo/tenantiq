import { useTheme } from "next-themes";
import { Sun, Moon, Monitor } from "lucide-react";
import { useTranslation } from "react-i18next";
import { updatePreferences } from "@/api/profile/profile";
import type { UpdatePreferencesInputBodyTheme } from "@/api/model";

export function ThemeToggle() {
  const { t } = useTranslation();
  const { theme, setTheme } = useTheme();

  const handleTheme = (value: string) => {
    setTheme(value);
    // Fire-and-forget: sync preference to server
    updatePreferences({ theme: value as UpdatePreferencesInputBodyTheme }).catch(() => {});
  };

  return (
    <div className="flex items-center gap-0.5 rounded-lg border bg-muted/50 p-0.5">
      <button
        onClick={() => handleTheme("light")}
        className={`inline-flex h-7 w-7 items-center justify-center rounded-md transition-colors cursor-pointer ${
          theme === "light"
            ? "bg-background text-foreground shadow-sm"
            : "text-muted-foreground hover:text-foreground"
        }`}
        title={t("theme.light")}
      >
        <Sun className="h-4 w-4" />
      </button>
      <button
        onClick={() => handleTheme("dark")}
        className={`inline-flex h-7 w-7 items-center justify-center rounded-md transition-colors cursor-pointer ${
          theme === "dark"
            ? "bg-background text-foreground shadow-sm"
            : "text-muted-foreground hover:text-foreground"
        }`}
        title={t("theme.dark")}
      >
        <Moon className="h-4 w-4" />
      </button>
      <button
        onClick={() => handleTheme("system")}
        className={`inline-flex h-7 w-7 items-center justify-center rounded-md transition-colors cursor-pointer ${
          theme === "system"
            ? "bg-background text-foreground shadow-sm"
            : "text-muted-foreground hover:text-foreground"
        }`}
        title={t("theme.system")}
      >
        <Monitor className="h-4 w-4" />
      </button>
    </div>
  );
}
