import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

const mockNavigate = vi.fn();
vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: React.PropsWithChildren<{ to: string }>) => (
    <a href={props.to}>{children}</a>
  ),
  useNavigate: () => mockNavigate,
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({
    user: { user_id: "u-1", email: "admin@test.com", role: "superadmin" },
    logout: vi.fn().mockResolvedValue(undefined),
  }),
}));

import { AppLayout } from "./app-layout";

describe("AppLayout", () => {
  beforeEach(() => mockNavigate.mockClear());

  it("renders header with logo and children", () => {
    render(<AppLayout><div>Page content</div></AppLayout>);
    expect(screen.getByText("iq")).toBeInTheDocument();
    expect(screen.getByText("Page content")).toBeInTheDocument();
  });

  it("shows Users nav link for superadmin", () => {
    render(<AppLayout><div /></AppLayout>);
    expect(screen.getByText("Users")).toBeInTheDocument();
  });

  it("shows Sign out button", () => {
    render(<AppLayout><div /></AppLayout>);
    expect(screen.getByText("Sign out")).toBeInTheDocument();
  });

  it("navigates to /login on sign out", async () => {
    render(<AppLayout><div /></AppLayout>);
    fireEvent.click(screen.getByText("Sign out"));
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith({ to: "/login" });
    });
  });
});
