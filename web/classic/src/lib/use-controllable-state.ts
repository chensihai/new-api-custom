import * as React from 'react'

type SetStateFn<T> = (prevState?: T) => T

type Setter<T> = (next: T | undefined | SetStateFn<T | undefined>) => void

export function useControllableState<T>(params: {
  prop?: T | undefined
  defaultProp: T
  onChange?: (state: T) => void
}): [T, Setter<T>]
export function useControllableState<T>(params: {
  prop?: T | undefined
  defaultProp?: T | undefined
  onChange?: (state: T) => void
}): [T | undefined, Setter<T>]
export function useControllableState<T>({
  prop,
  defaultProp,
  onChange,
}: {
  prop?: T | undefined
  defaultProp?: T | undefined
  onChange?: (state: T) => void
}): [T | undefined, Setter<T>] {
  const [uncontrolledProp, setUncontrolledProp] = React.useState<T | undefined>(
    defaultProp
  )
  const isControlled = prop !== undefined
  const value = isControlled ? prop : uncontrolledProp
  const handleChangeRef = React.useRef(onChange)

  React.useEffect(() => {
    handleChangeRef.current = onChange
  })

  const setValue = React.useCallback<Setter<T>>(
    (next) => {
      if (isControlled) {
        const nextValue =
          typeof next === 'function'
            ? (next as SetStateFn<T | undefined>)(prop)
            : next
        if (nextValue !== prop) {
          handleChangeRef.current?.(nextValue as T)
        }
      } else {
        setUncontrolledProp(
          next as T | undefined | ((prev: T | undefined) => T | undefined)
        )
      }
    },
    [isControlled, prop]
  )

  return [value, setValue]
}
