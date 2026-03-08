import { useTranslation } from "react-i18next";
import { Languages } from "lucide-react";
import { updatePreferences } from "@/api/profile/profile";
import type { UpdatePreferencesInputBodyLanguage } from "@/api/model";

const LANGUAGES = ["en", "es"] as const;

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation();

  const handleLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    // Fire-and-forget: sync preference to server
    updatePreferences({ language: lng as UpdatePreferencesInputBodyLanguage }).catch(() => {});
  };

  return (
    <div className="flex items-center gap-0.5 rounded-lg border bg-muted/50 p-0.5">
      <Languages className="mx-1 h-4 w-4 text-muted-foreground" />
      {LANGUAGES.map((lng) => (
        <button
          key={lng}
          onClick={() => handleLanguage(lng)}
          className={`inline-flex h-7 items-center justify-center rounded-md px-2 text-xs font-medium transition-colors cursor-pointer ${
            i18n.language?.startsWith(lng)
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
          title={t(`language.${lng}`)}
        >
          {lng.toUpperCase()}
        </button>
      ))}
    </div>
  );
}
