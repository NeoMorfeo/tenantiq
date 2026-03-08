import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: Record<string, unknown>) => ({ ...opts }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockAuth = {
  user: { user_id: "u-1", email: "admin@test.com", role: "superadmin" },
};
vi.mock("@/lib/auth", () => ({
  useAuth: () => mockAuth,
}));

const mockProfileMutate = vi.fn();
const mockPasswordMutate = vi.fn();
const mockInvalidateQueries = vi.fn();

let mockProfile: Record<string, unknown> | undefined = {
  email: "admin@test.com",
  name: "Admin",
  role: "superadmin",
  theme: "system",
  language: "en",
};
let mockIsLoading = false;

let mockProfileIsError = false;
let mockProfileIsPending = false;
let mockPasswordIsError = false;
let mockPasswordIsPending = false;

vi.mock("@/api/profile/profile", () => ({
  useGetProfile: () => ({ data: mockProfile, isLoading: mockIsLoading }),
  useUpdateProfile: () => ({
    mutate: mockProfileMutate,
    isPending: mockProfileIsPending,
    isError: mockProfileIsError,
  }),
  getGetProfileQueryKey: () => ["/api/v1/users/me"],
}));

vi.mock("@/api/users/users", () => ({
  useUpdatePassword: () => ({
    mutate: mockPasswordMutate,
    isPending: mockPasswordIsPending,
    isError: mockPasswordIsError,
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ invalidateQueries: mockInvalidateQueries }),
}));

import { Route } from "./profile";

const ProfilePage = Route.component as React.ComponentType;

describe("ProfilePage", () => {
  beforeEach(() => {
    mockProfileMutate.mockClear();
    mockPasswordMutate.mockClear();
    mockInvalidateQueries.mockClear();
    mockProfile = {
      email: "admin@test.com",
      name: "Admin",
      role: "superadmin",
      theme: "system",
      language: "en",
    };
    mockIsLoading = false;
    mockProfileIsError = false;
    mockProfileIsPending = false;
    mockPasswordIsError = false;
    mockPasswordIsPending = false;
  });

  it("shows loading skeleton when data is loading", () => {
    mockIsLoading = true;
    mockProfile = undefined;
    render(<ProfilePage />);
    // Skeleton renders empty divs, just check the page doesn't show the form
    expect(screen.queryByText("profile.editProfile")).not.toBeInTheDocument();
  });

  it("renders profile info with email and role badge", () => {
    render(<ProfilePage />);
    expect(screen.getByText("profile.title")).toBeInTheDocument();
    expect(screen.getByText("admin@test.com")).toBeInTheDocument();
    expect(screen.getByText("roles.superadmin")).toBeInTheDocument();
  });

  it("initializes form fields from profile data", () => {
    render(<ProfilePage />);
    const nameInput = screen.getByLabelText("common.name");
    const emailInput = screen.getByLabelText("common.email");
    expect(nameInput).toHaveValue("Admin");
    expect(emailInput).toHaveValue("admin@test.com");
  });

  it("updates email field on change", () => {
    render(<ProfilePage />);
    const emailInput = screen.getByLabelText("common.email");
    fireEvent.change(emailInput, { target: { value: "new@test.com" } });
    expect(emailInput).toHaveValue("new@test.com");
  });

  it("calls updateProfile mutation on form submit and shows success", () => {
    mockProfileMutate.mockImplementation((_args: unknown, opts: { onSuccess: () => void }) => {
      opts.onSuccess();
    });
    render(<ProfilePage />);
    const nameInput = screen.getByLabelText("common.name");
    fireEvent.change(nameInput, { target: { value: "New Name" } });
    fireEvent.submit(nameInput.closest("form")!);
    expect(mockProfileMutate).toHaveBeenCalledWith(
      { data: { name: "New Name", email: "admin@test.com" } },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
    expect(mockInvalidateQueries).toHaveBeenCalled();
    expect(screen.getByText("profile.saveSuccess")).toBeInTheDocument();
  });

  it("calls updatePassword mutation on password form submit and shows success", () => {
    mockPasswordMutate.mockImplementation((_args: unknown, opts: { onSuccess: () => void }) => {
      opts.onSuccess();
    });
    render(<ProfilePage />);
    const currentPw = screen.getByLabelText("profile.currentPassword");
    const newPw = screen.getByLabelText("profile.newPassword");
    fireEvent.change(currentPw, { target: { value: "old123456" } });
    fireEvent.change(newPw, { target: { value: "new123456" } });
    fireEvent.submit(currentPw.closest("form")!);
    expect(mockPasswordMutate).toHaveBeenCalledWith(
      {
        id: "u-1",
        data: { current_password: "old123456", new_password: "new123456" },
      },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
    // Password fields should be cleared after success
    expect(currentPw).toHaveValue("");
    expect(newPw).toHaveValue("");
  });

  it("shows profile update error message", () => {
    mockProfileIsError = true;
    render(<ProfilePage />);
    expect(screen.getByText("profile.updateError")).toBeInTheDocument();
  });

  it("shows password error message", () => {
    mockPasswordIsError = true;
    render(<ProfilePage />);
    expect(screen.getByText("profile.passwordError")).toBeInTheDocument();
  });

  it("shows loading text on profile button when pending", () => {
    mockProfileIsPending = true;
    render(<ProfilePage />);
    // The submit button shows loading text
    const buttons = screen.getAllByRole("button");
    const profileBtn = buttons.find((b) => b.textContent === "common.loading");
    expect(profileBtn).toBeDefined();
  });

  it("shows loading text on password button when pending", () => {
    mockPasswordIsPending = true;
    render(<ProfilePage />);
    const buttons = screen.getAllByRole("button");
    const passwordBtn = buttons.filter((b) => b.textContent === "common.loading");
    expect(passwordBtn.length).toBeGreaterThan(0);
  });

  it("does not submit password form when user_id is missing", () => {
    mockAuth.user = { user_id: undefined as unknown as string, email: "a@b.com", role: "viewer" };
    render(<ProfilePage />);
    const currentPw = screen.getByLabelText("profile.currentPassword");
    const newPw = screen.getByLabelText("profile.newPassword");
    fireEvent.change(currentPw, { target: { value: "old" } });
    fireEvent.change(newPw, { target: { value: "new12345" } });
    fireEvent.submit(currentPw.closest("form")!);
    expect(mockPasswordMutate).not.toHaveBeenCalled();
    // Restore
    mockAuth.user = { user_id: "u-1", email: "admin@test.com", role: "superadmin" };
  });
});
