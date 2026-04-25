import React, { createContext, useContext, useState, ReactNode } from 'react';
import { toast } from 'sonner';

interface User {
  id: string;
  username: string;
  email: string;
}

interface AuthContextType {
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, username: string) => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);

  const login = async (email: string, password: string) => {
    // Mock login - in real app this would call an API
    await new Promise(resolve => setTimeout(resolve, 500));
    setUser({
      id: '1',
      username: 'johndoe',
      email: email,
    });

    // Check if user has seen the historical data toast
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
    // Mock signup - in real app this would call an API
    await new Promise(resolve => setTimeout(resolve, 500));
    setUser({
      id: '1',
      username: username,
      email: email,
    });

    // Show toast for new users
    setTimeout(() => {
      toast.info('You can enable historical data from your profile settings', {
        duration: 3000,
      });
      localStorage.setItem('hasSeenHistoricalDataToast', 'true');
    }, 1000);
  };

  const logout = () => {
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
