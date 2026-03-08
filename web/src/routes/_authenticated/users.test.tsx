import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/mocks/server";
import { createTestWrapper } from "@/test/wrapper";

vi.mock("@tanstack/react-router", async () => {
  const actual = await vi.importActual("@tanstack/react-router");
  return {
    ...actual,
    createFileRoute: () => (opts: Record<string, unknown>) => ({ ...opts }),
  };
});

import { Route } from "./users";

const UsersPage = Route.component as React.ComponentType;

const USERS_HANDLER = http.get("/api/v1/users", () =>
  HttpResponse.json([
    {
      id: "user-1",
      name: "Admin",
      email: "admin@test.com",
      role: "superadmin",
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
    {
      id: "u-2",
      name: "Viewer",
      email: "viewer@test.com",
      role: "viewer",
      created_at: "2025-06-01T00:00:00Z",
      updated_at: "2025-06-01T00:00:00Z",
    },
  ]),
);

describe("UsersPage", () => {
  it("renders user table with data", async () => {
    server.use(USERS_HANDLER);
    const { Wrapper } = createTestWrapper();
    render(<UsersPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(screen.getByText("Admin")).toBeInTheDocument();
    });
    expect(screen.getByText("Viewer")).toBeInTheDocument();
    expect(screen.getByText("2 user(s)")).toBeInTheDocument();
  });

  it("shows empty state when no users", async () => {
    server.use(http.get("/api/v1/users", () => HttpResponse.json([])));
    const { Wrapper } = createTestWrapper();
    render(<UsersPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("No users yet. Create your first one."),
      ).toBeInTheDocument();
    });
  });

  it("hides delete button for current user", async () => {
    server.use(USERS_HANDLER);
    const { Wrapper } = createTestWrapper();
    const { container } = render(<UsersPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(screen.getByText("Admin")).toBeInTheDocument();
    });

    // user-1 is the current user (from auth mock) → no delete button
    // u-2 is another user → has delete button
    // There should be exactly 1 trash icon (for u-2 only, not for user-1)
    const trashIcons = container.querySelectorAll("svg.lucide-trash-2");
    expect(trashIcons).toHaveLength(1);
  });

  it("shows Create User button", async () => {
    server.use(USERS_HANDLER);
    const { Wrapper } = createTestWrapper();
    render(<UsersPage />, { wrapper: Wrapper });

    expect(screen.getByText("Create User")).toBeInTheDocument();
  });
});
