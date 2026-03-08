import Axios, { type AxiosRequestConfig } from "axios";

export const axios = Axios.create({
  baseURL: "",
});

// Inject Bearer token from localStorage into every request.
axios.interceptors.request.use((config) => {
  const token = localStorage.getItem("tenantiq_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Orval mutator: wraps every generated API call with our configured instance.
export const customInstance = <T>(config: AxiosRequestConfig): Promise<T> => {
  const promise = axios(config).then(({ data }) => data);
  return promise;
};
