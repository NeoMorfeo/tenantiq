import { useTheme } from "next-themes";
import { Sun, Moon, Monitor } from "lucide-react";

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();

  return (
    <div className="flex items-center gap-0.5 rounded-lg border bg-muted/50 p-0.5">
      <button
        onClick={() => setTheme("light")}
        className={`inline-flex h-7 w-7 items-center justify-center rounded-md transition-colors cursor-pointer ${
          theme === "light"
            ? "bg-background text-foreground shadow-sm"
            : "text-muted-foreground hover:text-foreground"
        }`}
        title="Light"
      >
        <Sun className="h-4 w-4" />
      </button>
      <button
        onClick={() => setTheme("dark")}
        className={`inline-flex h-7 w-7 items-center justify-center rounded-md transition-colors cursor-pointer ${
          theme === "dark"
            ? "bg-background text-foreground shadow-sm"
            : "text-muted-foreground hover:text-foreground"
        }`}
        title="Dark"
      >
        <Moon className="h-4 w-4" />
      </button>
      <button
        onClick={() => setTheme("system")}
        className={`inline-flex h-7 w-7 items-center justify-center rounded-md transition-colors cursor-pointer ${
          theme === "system"
            ? "bg-background text-foreground shadow-sm"
            : "text-muted-foreground hover:text-foreground"
        }`}
        title="System"
      >
        <Monitor className="h-4 w-4" />
      </button>
    </div>
  );
}
