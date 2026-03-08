import { describe, it, expect } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "@/test/mocks/server";
import { axios } from "./axios";

describe("401 interceptor", () => {
  it("refreshes token and retries on 401", async () => {
    let refreshCalled = false;

    // Track that the refresh endpoint was actually called.
    server.use(
      http.post("/api/v1/auth/refresh", () => {
        refreshCalled = true;
        return HttpResponse.json({ access_token: "new", refresh_token: "new" });
      }),
    );

    // First call 401, second call succeeds (use closure to track).
    let attempts = 0;
    server.use(
      http.get("/api/v1/tenants", () => {
        attempts++;
        if (attempts <= 1) {
          return HttpResponse.json({ detail: "Unauthorized" }, { status: 401 });
        }
        return HttpResponse.json([{ id: "t1", name: "Acme" }]);
      }),
    );

    const res = await axios.get("/api/v1/tenants");

    expect(refreshCalled).toBe(true);
    expect(attempts).toBe(2);
    expect(res.data).toEqual([{ id: "t1", name: "Acme" }]);
  });

  it("does not retry if already retried", async () => {
    server.use(
      http.get("/api/v1/tenants", () =>
        HttpResponse.json({ detail: "Unauthorized" }, { status: 401 }),
      ),
    );

    await expect(axios.get("/api/v1/tenants")).rejects.toThrow();
  });

  it("does not intercept /auth/me", async () => {
    server.use(
      http.get("/api/v1/auth/me", () =>
        HttpResponse.json({ detail: "Unauthorized" }, { status: 401 }),
      ),
    );

    await expect(axios.get("/api/v1/auth/me")).rejects.toThrow();
  });

  it("does not intercept /auth/login", async () => {
    server.use(
      http.post("/api/v1/auth/login", () =>
        HttpResponse.json({ detail: "Unauthorized" }, { status: 401 }),
      ),
    );

    await expect(
      axios.post("/api/v1/auth/login", { email: "x", password: "y" }),
    ).rejects.toThrow();
  });

  it("redirects when refresh fails", async () => {
    server.use(
      http.get("/api/v1/tenants", () =>
        HttpResponse.json({ detail: "Unauthorized" }, { status: 401 }),
      ),
      http.post("/api/v1/auth/refresh", () =>
        HttpResponse.json({ detail: "Expired" }, { status: 401 }),
      ),
    );

    await expect(axios.get("/api/v1/tenants")).rejects.toThrow();
  });
});
