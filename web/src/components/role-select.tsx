import { useState } from "react";
import { Badge } from "@/components/ui/badge";

const ROLES = ["superadmin", "admin", "viewer"] as const;

const roleBadgeVariant: Record<string, "default" | "secondary" | "outline"> = {
  superadmin: "default",
  admin: "secondary",
  viewer: "outline",
};

interface RoleSelectProps {
  value: string;
  onSave: (v: string) => void;
}

export const RoleSelect = ({ value, onSave }: RoleSelectProps) => {
  const [editing, setEditing] = useState(false);

  if (!editing) {
    return (
      <Badge
        variant={roleBadgeVariant[value] ?? "outline"}
        className="cursor-pointer"
        onClick={() => setEditing(true)}
      >
        {value}
      </Badge>
    );
  }

  return (
    <select
      autoFocus
      value={value}
      onChange={(e) => {
        onSave(e.target.value);
        setEditing(false);
      }}
      onBlur={() => setEditing(false)}
      className="h-7 rounded border bg-background px-2 text-sm"
    >
      {ROLES.map((r) => (
        <option key={r} value={r}>
          {r}
        </option>
      ))}
    </select>
  );
};
