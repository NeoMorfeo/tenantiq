import Axios, {
  type AxiosRequestConfig,
  type AxiosError,
  type InternalAxiosRequestConfig,
} from "axios";
import i18n from "@/i18n/i18n";

export const axios = Axios.create({
  baseURL: "",
  withCredentials: true, // Send HttpOnly cookies with every request.
});

// Send the user's language preference so the backend can translate error messages.
axios.interceptors.request.use((config) => {
  config.headers["Accept-Language"] = i18n.language;
  return config;
});

// --- Silent 401 refresh with request queuing ---

let isRefreshing = false;
let pendingQueue: {
  resolve: (value?: unknown) => void;
  reject: (reason?: unknown) => void;
}[] = [];

function drainQueue(error: AxiosError | null) {
  pendingQueue.forEach(({ resolve, reject }) =>
    error ? reject(error) : resolve(),
  );
  pendingQueue = [];
}

axios.interceptors.response.use(undefined, async (error: AxiosError) => {
  const original = error.config as InternalAxiosRequestConfig & {
    _retried?: boolean;
  };
  if (!original || error.response?.status !== 401) return Promise.reject(error);

  // Don't intercept auth endpoints — avoids infinite loops.
  const url = original.url ?? "";
  if (
    url.includes("/auth/refresh") ||
    url.includes("/auth/login") ||
    url.includes("/auth/me")
  ) {
    return Promise.reject(error);
  }

  // Already retried after refresh — don't loop.
  if (original._retried) return Promise.reject(error);

  // If a refresh is already in flight, queue this request.
  if (isRefreshing) {
    return new Promise((resolve, reject) => {
      pendingQueue.push({ resolve, reject });
    }).then(() => {
      original._retried = true;
      return axios(original);
    });
  }

  isRefreshing = true;

  try {
    await axios.post("/api/v1/auth/refresh");
    drainQueue(null);
    original._retried = true;
    return axios(original);
  } catch (refreshError) {
    drainQueue(refreshError as AxiosError);
    // Session expired — let AuthProvider/route guard handle the redirect.
    return Promise.reject(refreshError);
  } finally {
    isRefreshing = false;
  }
});

// Orval mutator: wraps every generated API call with our configured instance.
export const customInstance = <T>(config: AxiosRequestConfig): Promise<T> => {
  const promise = axios(config).then(({ data }) => data);
  return promise;
};
