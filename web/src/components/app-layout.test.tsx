import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: React.PropsWithChildren<{ to: string }>) => (
    <a href={props.to}>{children}</a>
  ),
}));

vi.mock("@/components/user-menu", () => ({
  UserMenu: () => <div data-testid="user-menu">UserMenu</div>,
}));

import { AppLayout } from "./app-layout";

describe("AppLayout", () => {
  it("renders header with logo and children", () => {
    render(<AppLayout><div>Page content</div></AppLayout>);
    expect(screen.getByText("iq")).toBeInTheDocument();
    expect(screen.getByText("Page content")).toBeInTheDocument();
  });

  it("renders the UserMenu component", () => {
    render(<AppLayout><div /></AppLayout>);
    expect(screen.getByTestId("user-menu")).toBeInTheDocument();
  });

  it("renders logo link pointing to /", () => {
    render(<AppLayout><div /></AppLayout>);
    const logoLink = screen.getByRole("link");
    expect(logoLink).toHaveAttribute("href", "/");
  });
});
