import Axios, { type AxiosRequestConfig } from "axios";

export const axios = Axios.create({
  baseURL: "",
  withCredentials: true, // Send HttpOnly cookies with every request.
});

// Orval mutator: wraps every generated API call with our configured instance.
export const customInstance = <T>(config: AxiosRequestConfig): Promise<T> => {
  const promise = axios(config).then(({ data }) => data);
  return promise;
};
