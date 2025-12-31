import { useState, useEffect, useCallback } from 'react'

interface AsyncState<T> {
  data: T | null
  loading: boolean
  error: Error | null
}

export function useAsync<T>(
  asyncFn: () => Promise<T>,
  deps: React.DependencyList = []
): AsyncState<T> & { refetch: () => void } {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: true,
    error: null,
  })

  const execute = useCallback(async () => {
    setState((prev) => ({ ...prev, loading: true, error: null }))
    try {
      const data = await asyncFn()
      setState({ data, loading: false, error: null })
    } catch (error) {
      setState({ data: null, loading: false, error: error as Error })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  useEffect(() => {
    execute()
  }, [execute])

  return { ...state, refetch: execute }
}

export function useMutation<T, A extends unknown[]>(
  mutationFn: (...args: A) => Promise<T>
): {
  mutate: (...args: A) => Promise<T>
  data: T | null
  loading: boolean
  error: Error | null
  reset: () => void
} {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: false,
    error: null,
  })

  const mutate = useCallback(
    async (...args: A) => {
      setState({ data: null, loading: true, error: null })
      try {
        const data = await mutationFn(...args)
        setState({ data, loading: false, error: null })
        return data
      } catch (error) {
        setState({ data: null, loading: false, error: error as Error })
        throw error
      }
    },
    [mutationFn]
  )

  const reset = useCallback(() => {
    setState({ data: null, loading: false, error: null })
  }, [])

  return { ...state, mutate, reset }
}
