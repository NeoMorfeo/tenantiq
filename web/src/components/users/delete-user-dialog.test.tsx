import { describe, it, expect } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/mocks/server";
import { createTestWrapper } from "@/test/wrapper";
import { DeleteUserDialog } from "./delete-user-dialog";

const mockUser = {
  id: "u-2",
  name: "Jane Doe",
  email: "jane@example.com",
  role: "viewer" as const,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

describe("DeleteUserDialog", () => {
  it("renders the trash icon trigger", () => {
    const { Wrapper } = createTestWrapper();
    render(<DeleteUserDialog user={mockUser} />, { wrapper: Wrapper });
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("opens dialog with user info on trigger click", async () => {
    const { Wrapper } = createTestWrapper();
    render(<DeleteUserDialog user={mockUser} />, { wrapper: Wrapper });

    await userEvent.click(screen.getByRole("button"));

    expect(screen.getByText("Delete User")).toBeInTheDocument();
    expect(screen.getByText(/jane@example.com/)).toBeInTheDocument();
  });

  it("closes dialog on Cancel", async () => {
    const { Wrapper } = createTestWrapper();
    render(<DeleteUserDialog user={mockUser} />, { wrapper: Wrapper });

    await userEvent.click(screen.getByRole("button"));
    await userEvent.click(screen.getByText("Cancel"));

    await waitFor(() => {
      expect(screen.queryByText("Delete User")).not.toBeInTheDocument();
    });
  });

  it("deletes user and closes dialog on confirm", async () => {
    server.use(
      http.delete("/api/v1/users/u-2", () =>
        new HttpResponse(null, { status: 204 }),
      ),
    );

    const { Wrapper } = createTestWrapper();
    render(<DeleteUserDialog user={mockUser} />, { wrapper: Wrapper });

    await userEvent.click(screen.getByRole("button"));
    await userEvent.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(screen.queryByText("Delete User")).not.toBeInTheDocument();
    });
  });
});
