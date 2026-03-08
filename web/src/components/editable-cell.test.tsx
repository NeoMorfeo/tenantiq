import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { EditableCell } from "./editable-cell";

describe("EditableCell", () => {
  it("renders value as text by default", () => {
    render(<EditableCell value="Alice" onSave={vi.fn()} />);
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("switches to input on click", () => {
    render(<EditableCell value="Alice" onSave={vi.fn()} />);
    fireEvent.click(screen.getByText("Alice"));
    expect(screen.getByDisplayValue("Alice")).toBeInTheDocument();
  });

  it("calls onSave on Enter with new value", () => {
    const onSave = vi.fn();
    render(<EditableCell value="Alice" onSave={onSave} />);

    fireEvent.click(screen.getByText("Alice"));
    const input = screen.getByDisplayValue("Alice");
    fireEvent.change(input, { target: { value: "Bob" } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(onSave).toHaveBeenCalledWith("Bob");
  });

  it("calls onSave on blur with new value", () => {
    const onSave = vi.fn();
    render(<EditableCell value="Alice" onSave={onSave} />);

    fireEvent.click(screen.getByText("Alice"));
    const input = screen.getByDisplayValue("Alice");
    fireEvent.change(input, { target: { value: "Bob" } });
    fireEvent.blur(input);

    expect(onSave).toHaveBeenCalledWith("Bob");
  });

  it("does not call onSave if value unchanged", () => {
    const onSave = vi.fn();
    render(<EditableCell value="Alice" onSave={onSave} />);

    fireEvent.click(screen.getByText("Alice"));
    fireEvent.blur(screen.getByDisplayValue("Alice"));

    expect(onSave).not.toHaveBeenCalled();
  });

  it("cancels editing on Escape", () => {
    const onSave = vi.fn();
    render(<EditableCell value="Alice" onSave={onSave} />);

    fireEvent.click(screen.getByText("Alice"));
    const input = screen.getByDisplayValue("Alice");
    fireEvent.change(input, { target: { value: "Bob" } });
    fireEvent.keyDown(input, { key: "Escape" });

    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("does not save empty/whitespace-only value", () => {
    const onSave = vi.fn();
    render(<EditableCell value="Alice" onSave={onSave} />);

    fireEvent.click(screen.getByText("Alice"));
    const input = screen.getByDisplayValue("Alice");
    fireEvent.change(input, { target: { value: "   " } });
    fireEvent.blur(input);

    expect(onSave).not.toHaveBeenCalled();
  });
});
