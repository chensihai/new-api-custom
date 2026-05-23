import { useState, useCallback } from 'react';

export interface AuthUser {
  id: number;
  username: string;
  display_name?: string;
  email?: string;
  role: number;
  status?: number;
  group?: string;
  quota?: number;
  used_quota?: number;
  request_count?: number;
  aff_code?: string;
  aff_count?: number;
  aff_quota?: number;
  aff_history_quota?: number;
  inviter_id?: number;
  github_id?: string;
  oidc_id?: string;
  wechat_id?: string;
  telegram_id?: string;
  linux_do_id?: string;
  setting?: Record<string, unknown> | string;
  stripe_customer?: string;
  sidebar_modules?: string;
  permissions?: Record<string, unknown>;
}

type SafeAuthUser = Pick<AuthUser, 'id' | 'username' | 'display_name' | 'role' | 'status' | 'group'>;

function isSafeUser(obj: unknown): obj is SafeAuthUser {
  return (
    typeof obj === 'object' &&
    obj !== null &&
    typeof (obj as Record<string, unknown>).id === 'number' &&
    typeof (obj as Record<string, unknown>).username === 'string' &&
    typeof (obj as Record<string, unknown>).role === 'number'
  );
}

function loadUserFromStorage(): AuthUser | null {
  try {
    if (typeof window !== 'undefined') {
      const saved = window.localStorage.getItem('user');
      if (!saved) return null;
      const parsed = JSON.parse(saved);
      if (!isSafeUser(parsed)) {
        window.localStorage.removeItem('user');
        return null;
      }
      return parsed as AuthUser;
    }
  } catch {
    if (typeof window !== 'undefined') {
      window.localStorage.removeItem('user');
    }
  }
  return null;
}

function toSafeUser(user: AuthUser | null): SafeAuthUser | null {
  if (!user) return null;
  return {
    id: user.id,
    username: user.username,
    display_name: user.display_name,
    role: user.role,
    status: user.status,
    group: user.group,
  };
}

export function useAuthStore() {
  const [user, setUser] = useState<AuthUser | null>(loadUserFromStorage);

  const setAuthUser = useCallback((newUser: AuthUser | null) => {
    if (typeof window !== 'undefined') {
      if (newUser) {
        window.localStorage.setItem('user', JSON.stringify(toSafeUser(newUser)));
      } else {
        window.localStorage.removeItem('user');
      }
    }
    setUser(newUser);
  }, []);

  const reset = useCallback(() => {
    if (typeof window !== 'undefined') {
      window.localStorage.removeItem('user');
    }
    setUser(null);
  }, []);

  return {
    auth: {
      user,
      setUser: setAuthUser,
      reset,
    },
  };
}
