import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { toast } from 'sonner';
import {
  getCurrentUser,
  hasAccessToken,
  login as loginRequest,
  logout as logoutRequest,
  refreshAccessToken,
  signup as signupRequest,
} from '../lib/api';
import type { User } from '../lib/types';

interface AuthContextType {
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, username: string) => Promise<void>;
  logout: () => Promise<void>;
  isAuthenticated: boolean;
  isLoadingAuth: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoadingAuth, setIsLoadingAuth] = useState(true);

  useEffect(() => {
    const bootstrapAuth = async () => {
      try {
        if (!hasAccessToken()) {
          await refreshAccessToken();
        }

        const me = await getCurrentUser();
        setUser(me);
      } catch {
        try {
          await refreshAccessToken();
          const me = await getCurrentUser();
          setUser(me);
        } catch {
          setUser(null);
        }
      } finally {
        setIsLoadingAuth(false);
      }
    };

    bootstrapAuth();
  }, []);

  const login = async (email: string, password: string) => {
    await loginRequest(email, password);
    const me = await getCurrentUser();
    setUser(me);

    const hasSeenToast = localStorage.getItem('hasSeenHistoricalDataToast');
    if (!hasSeenToast) {
      setTimeout(() => {
        toast.info('You can enable historical data from your profile settings', {
          duration: 3000,
        });
        localStorage.setItem('hasSeenHistoricalDataToast', 'true');
      }, 1000);
    }
  };

  const signup = async (email: string, password: string, username: string) => {
    await signupRequest(username, email, password);
    await login(username, password);

    setTimeout(() => {
      toast.info('You can enable historical data from your profile settings', {
        duration: 3000,
      });
      localStorage.setItem('hasSeenHistoricalDataToast', 'true');
    }, 1000);
  };

  const logout = async () => {
    await logoutRequest();
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        login,
        signup,
        logout,
        isAuthenticated: !!user,
        isLoadingAuth,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
