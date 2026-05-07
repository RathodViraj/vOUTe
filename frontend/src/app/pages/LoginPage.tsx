import React from 'react';
import { useNavigate } from 'react-router';
import { Button } from '../components/ui/button';
import { Card } from '../components/ui/card';
// Use backend base URL for OAuth redirects so dev server doesn't intercept
const API_BASE = ((import.meta as any).env?.VITE_API_BASE_URL as string | undefined) || 'http://localhost:8080';

export function LoginPage() {
  const navigate = useNavigate();

  const startGoogle = () => {
    window.location.href = `${API_BASE}/auth/google/login`;
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 to-purple-600 p-4">
      <Card className="w-full max-w-md p-6 space-y-4">
        <div className="text-center space-y-2 mb-6">
          <h1 className="text-2xl font-bold">Welcome back</h1>
          <p className="text-sm text-muted-foreground">Login with OTP or continue with Google</p>
        </div>

        <div className="space-y-4">
          <Button onClick={() => navigate('/login-otp')} className="w-full bg-indigo-600 hover:bg-indigo-700">
            Login with OTP
          </Button>

          <Button variant="outline" onClick={startGoogle} className="w-full">
            Continue with Google
          </Button>

          <div className="text-center text-sm">
            Or use password login <button onClick={() => navigate('/login-password')} className="text-indigo-600 underline">here</button>
          </div>
        </div>
      </Card>
    </div>
  );
}
