import type { ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import { Separator } from "@/components/ui/separator";
import { UserMenu } from "@/components/user-menu";
import iconSvg from "@/assets/icon.svg";

export const AppLayout = ({ children }: { children: ReactNode }) => {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2">
            <img src={iconSvg} alt="tenantiq" className="h-8 w-8" />
            <span className="text-lg font-semibold tracking-tight">
              tenant<span className="text-primary">iq</span>
            </span>
          </Link>
          <UserMenu />
        </div>
      </header>
      <Separator />
      <main className="mx-auto max-w-7xl px-4 py-6">{children}</main>
    </div>
  );
};
