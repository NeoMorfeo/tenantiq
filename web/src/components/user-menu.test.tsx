import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

const mockNavigate = vi.fn();
vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mockNavigate,
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

const mockLogout = vi.fn().mockResolvedValue(undefined);
const mockAuth = {
  user: { user_id: "u-1", email: "admin@test.com", role: "superadmin" as string },
  logout: mockLogout,
};
vi.mock("@/lib/auth", () => ({
  useAuth: () => mockAuth,
}));

const mockMutate = vi.fn();
vi.mock("@/api/profile/profile", () => ({
  useUpdatePreferences: () => ({ mutate: mockMutate }),
}));

// Mock base-ui dropdown as simple HTML to make tests straightforward
vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuTrigger: ({ children, ...props }: React.PropsWithChildren<Record<string, unknown>>) => (
    <button data-testid="avatar-trigger" {...props}>{children}</button>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuLabel: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  DropdownMenuItem: ({ children, onClick, variant }: { children: React.ReactNode; onClick?: () => void; variant?: string }) => (
    <button onClick={onClick} data-variant={variant}>{children}</button>
  ),
  DropdownMenuSeparator: () => <hr />,
  DropdownMenuSub: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuSubTrigger: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuSubContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuRadioGroup: ({ children, onValueChange }: { children: React.ReactNode; value: string; onValueChange: (v: string) => void }) => (
    <div data-testid="radio-group" onClick={(e) => {
      const target = e.target as HTMLElement;
      const value = target.getAttribute("data-value");
      if (value) onValueChange(value);
    }}>{children}</div>
  ),
  DropdownMenuRadioItem: ({ children, value }: { children: React.ReactNode; value: string }) => (
    <button data-value={value}>{children}</button>
  ),
}));

vi.mock("@/components/ui/avatar", () => ({
  Avatar: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AvatarFallback: ({ children }: { children: React.ReactNode }) => <span data-testid="avatar-fallback">{children}</span>,
}));

import { UserMenu } from "./user-menu";

describe("UserMenu", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    mockSetTheme.mockClear();
    mockChangeLanguage.mockClear();
    mockMutate.mockClear();
    mockLogout.mockClear();
    mockAuth.user = { user_id: "u-1", email: "admin@test.com", role: "superadmin" };
  });

  it("renders avatar with initials from email", () => {
    render(<UserMenu />);
    expect(screen.getByTestId("avatar-fallback")).toHaveTextContent("AD");
  });

  it("shows ? when user email is missing", () => {
    mockAuth.user = { user_id: "u-1", email: undefined as unknown as string, role: "viewer" };
    render(<UserMenu />);
    expect(screen.getByTestId("avatar-fallback")).toHaveTextContent("?");
  });

  it("shows user email in the menu", () => {
    render(<UserMenu />);
    expect(screen.getByText("admin@test.com")).toBeInTheDocument();
  });

  it("shows Profile menu item", () => {
    render(<UserMenu />);
    expect(screen.getByText("nav.profile")).toBeInTheDocument();
  });

  it("navigates to /profile on Profile click", () => {
    render(<UserMenu />);
    fireEvent.click(screen.getByText("nav.profile"));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/profile" });
  });

  it("shows Settings section for superadmin", () => {
    render(<UserMenu />);
    expect(screen.getByText("nav.settings")).toBeInTheDocument();
    expect(screen.getByText("nav.users")).toBeInTheDocument();
  });

  it("hides Settings section for non-superadmin", () => {
    mockAuth.user = { user_id: "u-1", email: "viewer@test.com", role: "viewer" };
    render(<UserMenu />);
    expect(screen.queryByText("nav.settings")).not.toBeInTheDocument();
    expect(screen.queryByText("nav.users")).not.toBeInTheDocument();
  });

  it("navigates to /users on Users click", () => {
    render(<UserMenu />);
    fireEvent.click(screen.getByText("nav.users"));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/users" });
  });

  it("shows theme and language submenu triggers", () => {
    render(<UserMenu />);
    expect(screen.getByText("nav.theme")).toBeInTheDocument();
    expect(screen.getByText("nav.language")).toBeInTheDocument();
  });

  it("calls setTheme and mutate on theme change", () => {
    render(<UserMenu />);
    fireEvent.click(screen.getByText("theme.dark"));
    expect(mockSetTheme).toHaveBeenCalledWith("dark");
    expect(mockMutate).toHaveBeenCalledWith({ data: { theme: "dark" } });
  });

  it("calls changeLanguage and mutate on language change", () => {
    render(<UserMenu />);
    fireEvent.click(screen.getByText("language.es"));
    expect(mockChangeLanguage).toHaveBeenCalledWith("es");
    expect(mockMutate).toHaveBeenCalledWith({ data: { language: "es" } });
  });

  it("calls logout and navigates to /login on Sign out", async () => {
    render(<UserMenu />);
    fireEvent.click(screen.getByText("nav.signOut"));
    await waitFor(() => {
      expect(mockLogout).toHaveBeenCalled();
      expect(mockNavigate).toHaveBeenCalledWith({ to: "/login" });
    });
  });
});
