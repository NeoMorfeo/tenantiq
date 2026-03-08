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

import { Route } from "./index";

const DashboardPage = Route.component as React.ComponentType;

const TENANTS_HANDLER = http.get("/api/v1/tenants", () =>
  HttpResponse.json([
    {
      id: "t-1",
      name: "Acme Corp",
      slug: "acme",
      plan: "pro",
      status: "active",
      created_at: "2025-01-01T00:00:00Z",
    },
    {
      id: "t-2",
      name: "Beta Inc",
      slug: "beta",
      plan: "free",
      status: "creating",
      created_at: "2025-06-01T00:00:00Z",
    },
  ]),
);

describe("DashboardPage", () => {
  it("renders tenant table with data", async () => {
    server.use(TENANTS_HANDLER);
    const { Wrapper } = createTestWrapper();
    render(<DashboardPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    });
    expect(screen.getByText("Beta Inc")).toBeInTheDocument();
    expect(screen.getByText("2 tenant(s)")).toBeInTheDocument();
  });

  it("shows empty state when no tenants", async () => {
    server.use(http.get("/api/v1/tenants", () => HttpResponse.json([])));
    const { Wrapper } = createTestWrapper();
    render(<DashboardPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("No tenants yet. Create your first one."),
      ).toBeInTheDocument();
    });
  });

  it("shows loading skeletons initially", () => {
    server.use(
      http.get("/api/v1/tenants", () => {
        // never resolves → stays loading
        return new Promise(() => {});
      }),
    );
    const { Wrapper } = createTestWrapper();
    render(<DashboardPage />, { wrapper: Wrapper });

    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });
});
