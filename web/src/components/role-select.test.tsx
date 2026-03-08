import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { RoleSelect } from "./role-select";

describe("RoleSelect", () => {
  it("renders role as badge by default", () => {
    render(<RoleSelect value="admin" onSave={vi.fn()} />);
    expect(screen.getByText("admin")).toBeInTheDocument();
  });

  it("switches to select on badge click", () => {
    render(<RoleSelect value="admin" onSave={vi.fn()} />);
    fireEvent.click(screen.getByText("admin"));
    expect(screen.getByRole("combobox")).toBeInTheDocument();
  });

  it("calls onSave when option is selected", () => {
    const onSave = vi.fn();
    render(<RoleSelect value="admin" onSave={onSave} />);

    fireEvent.click(screen.getByText("admin"));
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: "viewer" },
    });

    expect(onSave).toHaveBeenCalledWith("viewer");
  });

  it("reverts to badge on blur without change", () => {
    render(<RoleSelect value="admin" onSave={vi.fn()} />);

    fireEvent.click(screen.getByText("admin"));
    fireEvent.blur(screen.getByRole("combobox"));

    expect(screen.getByText("admin")).toBeInTheDocument();
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
  });

  it("shows all three role options", () => {
    render(<RoleSelect value="admin" onSave={vi.fn()} />);
    fireEvent.click(screen.getByText("admin"));

    const options = screen.getAllByRole("option");
    expect(options).toHaveLength(3);
    expect(options.map((o) => o.textContent)).toEqual([
      "superadmin",
      "admin",
      "viewer",
    ]);
  });
});
