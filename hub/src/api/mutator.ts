import Axios, { AxiosError } from 'axios'
import type { AxiosRequestConfig } from 'axios'

export const AXIOS_INSTANCE = Axios.create({
  withCredentials: true,
})

AXIOS_INSTANCE.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ message?: string }>) => {
    const message = error.response?.data?.message
    if (message) error.message = message
    return Promise.reject(error)
  },
)

export const customInstance = <T>(config: AxiosRequestConfig, options?: AxiosRequestConfig): Promise<T> => {
  return AXIOS_INSTANCE({ ...config, ...options }).then(({ data }) => data)
}

export function isUnauthorized(error: unknown): boolean {
  return Axios.isAxiosError(error) && error.response?.status === 401
}

export type ErrorType<Error> = AxiosError<Error>
export type BodyType<BodyData> = BodyData
