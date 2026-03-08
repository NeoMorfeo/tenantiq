import React from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

const mockNavigate = vi.fn();
vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: Record<string, unknown>) => ({ ...opts }),
  Outlet: () => <div data-testid="outlet">Outlet</div>,
  useNavigate: () => mockNavigate,
}));

// Default: authenticated superadmin
const mockAuth = {
  isAuthenticated: true,
  loading: false,
  user: { user_id: "u-1", email: "admin@test.com", role: "superadmin" },
  logout: vi.fn(),
};
vi.mock("@/lib/auth", () => ({
  useAuth: () => mockAuth,
}));

vi.mock("@/components/app-layout", () => ({
  AppLayout: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="app-layout">{children}</div>
  ),
}));

const mockSetTheme = vi.fn();
vi.mock("next-themes", () => ({
  useTheme: () => ({ theme: "system", setTheme: mockSetTheme }),
}));

const mockChangeLanguage = vi.fn();
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", changeLanguage: mockChangeLanguage },
  }),
}));

const mockGetProfile = vi.fn();
vi.mock("@/api/profile/profile", () => ({
  getProfile: (...args: unknown[]) => mockGetProfile(...args),
}));

import { Route } from "./_authenticated";

const AuthenticatedLayout = Route.component as React.ComponentType;

describe("AuthenticatedLayout", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    mockSetTheme.mockClear();
    mockChangeLanguage.mockClear();
    mockGetProfile.mockReset();
    mockAuth.isAuthenticated = true;
    mockAuth.loading = false;
  });

  it("renders AppLayout with Outlet when authenticated", () => {
    mockGetProfile.mockResolvedValue({ theme: "system", language: "en" });
    render(<AuthenticatedLayout />);
    expect(screen.getByTestId("app-layout")).toBeInTheDocument();
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("shows loading state while auth is loading", () => {
    mockAuth.loading = true;
    mockAuth.isAuthenticated = false;
    render(<AuthenticatedLayout />);
    expect(screen.getByText("common.loading")).toBeInTheDocument();
  });

  it("redirects to /login when not authenticated", async () => {
    mockAuth.loading = false;
    mockAuth.isAuthenticated = false;
    render(<AuthenticatedLayout />);

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith({ to: "/login" });
    });
  });

  it("applies theme and language from profile on login", async () => {
    mockGetProfile.mockResolvedValue({ theme: "dark", language: "es" });
    render(<AuthenticatedLayout />);

    await waitFor(() => {
      expect(mockSetTheme).toHaveBeenCalledWith("dark");
      expect(mockChangeLanguage).toHaveBeenCalledWith("es");
    });
  });

  it("does not change language when already matching", async () => {
    mockGetProfile.mockResolvedValue({ theme: "light", language: "en" });
    render(<AuthenticatedLayout />);

    await waitFor(() => {
      expect(mockSetTheme).toHaveBeenCalledWith("light");
    });
    expect(mockChangeLanguage).not.toHaveBeenCalled();
  });

  it("handles profile fetch failure gracefully", async () => {
    mockGetProfile.mockRejectedValue(new Error("network error"));
    render(<AuthenticatedLayout />);

    // Should still render without crashing
    expect(screen.getByTestId("app-layout")).toBeInTheDocument();
  });
});
