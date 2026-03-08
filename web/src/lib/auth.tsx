import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
import type { TokenPair } from "@/api/model";

interface AuthState {
  token: string | null;
  isAuthenticated: boolean;
  login: (pair: TokenPair) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() =>
    localStorage.getItem("tenantiq_token"),
  );

  const login = useCallback((pair: TokenPair) => {
    localStorage.setItem("tenantiq_token", pair.AccessToken);
    if (pair.RefreshToken) {
      localStorage.setItem("tenantiq_refresh", pair.RefreshToken);
    }
    setToken(pair.AccessToken);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem("tenantiq_token");
    localStorage.removeItem("tenantiq_refresh");
    setToken(null);
  }, []);

  const value = useMemo(
    () => ({ token, isAuthenticated: !!token, login, logout }),
    [token, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
