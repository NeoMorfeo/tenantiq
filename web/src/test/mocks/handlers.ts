import { http, HttpResponse } from "msw";

export const handlers = [
  // Default: authenticated user
  http.get("/api/v1/auth/me", () =>
    HttpResponse.json({
      user_id: "user-1",
      email: "admin@tenantiq.local",
      role: "superadmin",
    }),
  ),

  http.post("/api/v1/auth/login", async ({ request }) => {
    const body = (await request.json()) as {
      email: string;
      password: string;
    };
    if (body.email === "admin@tenantiq.local" && body.password === "secret") {
      return HttpResponse.json({
        access_token: "fake-access",
        refresh_token: "fake-refresh",
      });
    }
    return HttpResponse.json({ detail: "Invalid credentials" }, { status: 401 });
  }),

  http.post("/api/v1/auth/logout", () => new HttpResponse(null, { status: 204 })),

  http.post("/api/v1/auth/refresh", () =>
    HttpResponse.json({
      access_token: "refreshed-access",
      refresh_token: "refreshed-refresh",
    }),
  ),
];
