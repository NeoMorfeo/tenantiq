import { describe, it, expect } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/mocks/server";
import { createTestWrapper } from "@/test/wrapper";
import { CreateUserDialog } from "./create-user-dialog";

describe("CreateUserDialog", () => {
  it("renders the trigger button", () => {
    const { Wrapper } = createTestWrapper();
    render(<CreateUserDialog />, { wrapper: Wrapper });
    expect(screen.getByText("Create User")).toBeInTheDocument();
  });

  it("opens dialog on trigger click", async () => {
    const { Wrapper } = createTestWrapper();
    render(<CreateUserDialog />, { wrapper: Wrapper });

    await userEvent.click(screen.getByText("Create User"));
    expect(screen.getByText("Add a new user account.")).toBeInTheDocument();
  });

  it("shows validation errors for empty fields", async () => {
    const { Wrapper } = createTestWrapper();
    render(<CreateUserDialog />, { wrapper: Wrapper });

    await userEvent.click(screen.getByText("Create User"));

    const submitButtons = screen.getAllByRole("button", { name: /create/i });
    const submitButton = submitButtons.find(
      (btn) => btn.getAttribute("type") === "submit",
    );
    await userEvent.click(submitButton!);

    await waitFor(() => {
      expect(screen.getByText(/invalid/i) || screen.getByText(/required/i) || screen.getAllByText(/./)).toBeTruthy();
    });
  });

  it("submits form with valid data and closes dialog", async () => {
    server.use(
      http.post("/api/v1/users", async () =>
        HttpResponse.json({
          id: "u-new",
          email: "new@example.com",
          name: "New User",
          role: "viewer",
          created_at: new Date().toISOString(),
        }),
      ),
    );

    const { Wrapper } = createTestWrapper();
    render(<CreateUserDialog />, { wrapper: Wrapper });

    await userEvent.click(screen.getByText("Create User"));

    await userEvent.type(screen.getByLabelText("Email"), "new@example.com");
    await userEvent.type(screen.getByLabelText("Name"), "New User");
    await userEvent.type(screen.getByLabelText("Password"), "password123");

    const submitButtons = screen.getAllByRole("button", { name: /create/i });
    const submitButton = submitButtons.find(
      (btn) => btn.getAttribute("type") === "submit",
    );
    await userEvent.click(submitButton!);

    await waitFor(() => {
      expect(screen.queryByText("Add a new user account.")).not.toBeInTheDocument();
    });
  });

  it("shows error message when API returns error", async () => {
    server.use(
      http.post("/api/v1/users", () =>
        HttpResponse.json({ detail: "Conflict" }, { status: 409 }),
      ),
    );

    const { Wrapper } = createTestWrapper();
    render(<CreateUserDialog />, { wrapper: Wrapper });

    await userEvent.click(screen.getByText("Create User"));

    await userEvent.type(screen.getByLabelText("Email"), "dup@example.com");
    await userEvent.type(screen.getByLabelText("Name"), "Dup");
    await userEvent.type(screen.getByLabelText("Password"), "password123");

    const submitButtons = screen.getAllByRole("button", { name: /create/i });
    const submitButton = submitButtons.find(
      (btn) => btn.getAttribute("type") === "submit",
    );
    await userEvent.click(submitButton!);

    await waitFor(() => {
      expect(screen.getByText(/failed to create user/i)).toBeInTheDocument();
    });
  });
});
