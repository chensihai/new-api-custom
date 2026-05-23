import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React, { useRef } from 'react';

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        gcTime: 300_000,
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

function getQueryClient() {
  if (!browserQueryClient) {
    browserQueryClient = makeQueryClient();
  }
  return browserQueryClient;
}

export function PlaygroundQueryProvider({ children }: { children: React.ReactNode }) {
  const queryClientRef = useRef(getQueryClient());
  return (
    <QueryClientProvider client={queryClientRef.current}>
      {children}
    </QueryClientProvider>
  );
}

export { getQueryClient as queryClient };
