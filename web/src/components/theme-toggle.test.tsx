import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

const mockSetTheme = vi.fn();
vi.mock("next-themes", () => ({
  useTheme: () => ({ theme: "system", setTheme: mockSetTheme }),
}));

vi.mock("@/api/profile/profile", () => ({
  updatePreferences: vi.fn(() => Promise.resolve()),
}));

import { ThemeToggle } from "./theme-toggle";

describe("ThemeToggle", () => {
  beforeEach(() => mockSetTheme.mockClear());

  it("renders light, dark, and system buttons", () => {
    render(<ThemeToggle />);
    expect(screen.getByTitle("Light")).toBeInTheDocument();
    expect(screen.getByTitle("Dark")).toBeInTheDocument();
    expect(screen.getByTitle("System")).toBeInTheDocument();
  });

  it("calls setTheme('light') on light button click", () => {
    render(<ThemeToggle />);
    fireEvent.click(screen.getByTitle("Light"));
    expect(mockSetTheme).toHaveBeenCalledWith("light");
  });

  it("calls setTheme('dark') on dark button click", () => {
    render(<ThemeToggle />);
    fireEvent.click(screen.getByTitle("Dark"));
    expect(mockSetTheme).toHaveBeenCalledWith("dark");
  });

  it("calls setTheme('system') on system button click", () => {
    render(<ThemeToggle />);
    fireEvent.click(screen.getByTitle("System"));
    expect(mockSetTheme).toHaveBeenCalledWith("system");
  });
});
