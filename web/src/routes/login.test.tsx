import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/mocks/server";
import { createTestWrapper } from "@/test/wrapper";

const mockNavigate = vi.fn();
vi.mock("@tanstack/react-router", async () => {
  const actual = await vi.importActual("@tanstack/react-router");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    createFileRoute: () => (opts: Record<string, unknown>) => ({
      ...opts,
    }),
  };
});

import { Route } from "./login";

const LoginPage = Route.component as React.ComponentType;

describe("LoginPage", () => {
  beforeEach(() => mockNavigate.mockClear());

  it("renders login form", () => {
    const { Wrapper } = createTestWrapper();
    render(<LoginPage />, { wrapper: Wrapper });

    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument();
  });

  it("shows error on invalid credentials", async () => {
    server.use(
      http.post("/api/v1/auth/login", () =>
        HttpResponse.json({ detail: "Invalid" }, { status: 401 }),
      ),
    );

    const { Wrapper } = createTestWrapper();
    render(<LoginPage />, { wrapper: Wrapper });

    await userEvent.type(screen.getByLabelText("Email"), "wrong@test.com");
    await userEvent.type(screen.getByLabelText("Password"), "wrong");
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => {
      expect(screen.getByText("Invalid email or password")).toBeInTheDocument();
    });
  });

  it("navigates to / on successful login", async () => {
    const { Wrapper } = createTestWrapper();
    render(<LoginPage />, { wrapper: Wrapper });

    await userEvent.type(
      screen.getByLabelText("Email"),
      "admin@tenantiq.local",
    );
    await userEvent.type(screen.getByLabelText("Password"), "secret");
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith({ to: "/" });
    });
  });
});
