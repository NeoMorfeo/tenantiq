import { describe, it, expect, vi } from "vitest";
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

import { Route } from "./_authenticated";

const AuthenticatedLayout = Route.component as React.ComponentType;

describe("AuthenticatedLayout", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    mockAuth.isAuthenticated = true;
    mockAuth.loading = false;
  });

  it("renders AppLayout with Outlet when authenticated", () => {
    render(<AuthenticatedLayout />);
    expect(screen.getByTestId("app-layout")).toBeInTheDocument();
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("shows loading state while auth is loading", () => {
    mockAuth.loading = true;
    mockAuth.isAuthenticated = false;
    render(<AuthenticatedLayout />);
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("redirects to /login when not authenticated", async () => {
    mockAuth.loading = false;
    mockAuth.isAuthenticated = false;
    render(<AuthenticatedLayout />);

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith({ to: "/login" });
    });
  });
});
