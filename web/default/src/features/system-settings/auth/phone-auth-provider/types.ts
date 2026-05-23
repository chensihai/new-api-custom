export type PhoneAuthProvidersResponse = {
  success: boolean
  message?: string
  data?: {
    settings: Record<string, string | boolean>
    providers?: Array<{ key: string; label: string }>
  }
}

export type PhoneAuthProvidersUpdatePayload = Record<string, string | boolean>
