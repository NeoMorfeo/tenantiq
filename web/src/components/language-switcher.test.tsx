import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

const mockChangeLanguage = vi.fn();
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    i18n: { language: "en", changeLanguage: mockChangeLanguage },
    t: (key: string) => key,
  }),
}));

vi.mock("@/api/profile/profile", () => ({
  updatePreferences: vi.fn(() => Promise.resolve()),
}));

import { LanguageSwitcher } from "./language-switcher";

describe("LanguageSwitcher", () => {
  beforeEach(() => mockChangeLanguage.mockClear());

  it("renders EN and ES buttons", () => {
    render(<LanguageSwitcher />);
    expect(screen.getByText("EN")).toBeInTheDocument();
    expect(screen.getByText("ES")).toBeInTheDocument();
  });

  it("calls i18n.changeLanguage('es') on ES button click", () => {
    render(<LanguageSwitcher />);
    fireEvent.click(screen.getByText("ES"));
    expect(mockChangeLanguage).toHaveBeenCalledWith("es");
  });

  it("calls i18n.changeLanguage('en') on EN button click", () => {
    render(<LanguageSwitcher />);
    fireEvent.click(screen.getByText("EN"));
    expect(mockChangeLanguage).toHaveBeenCalledWith("en");
  });

  it("applies active styling to the current language button", () => {
    render(<LanguageSwitcher />);
    const enButton = screen.getByText("EN");
    const esButton = screen.getByText("ES");
    expect(enButton.className).toContain("bg-background");
    expect(esButton.className).not.toContain("bg-background");
  });
});
