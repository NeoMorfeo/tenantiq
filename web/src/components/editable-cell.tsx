import { useState } from "react";
import { Input } from "@/components/ui/input";

interface EditableCellProps {
  value: string;
  onSave: (v: string) => void;
  type?: "text" | "email" | "number";
  className?: string;
}

export const EditableCell = ({
  value,
  onSave,
  type = "text",
  className,
}: EditableCellProps) => {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);

  if (!editing) {
    return (
      <span
        className={`cursor-pointer rounded px-1 py-0.5 hover:bg-muted ${className ?? ""}`}
        onClick={() => {
          setDraft(value);
          setEditing(true);
        }}
      >
        {value}
      </span>
    );
  }

  const commit = () => {
    setEditing(false);
    if (draft !== value && draft.trim()) onSave(draft.trim());
  };

  return (
    <Input
      autoFocus
      type={type}
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === "Enter") commit();
        if (e.key === "Escape") {
          setEditing(false);
          setDraft(value);
        }
      }}
      className="h-7 w-auto min-w-30"
    />
  );
};
