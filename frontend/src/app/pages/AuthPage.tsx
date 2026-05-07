import React from 'react';
import { useNavigate } from 'react-router';
import { Button } from '../components/ui/button';
import { Card } from '../components/ui/card';

// Use backend base URL for OAuth redirects so dev server doesn't intercept
const API_BASE = ((import.meta as any).env?.VITE_API_BASE_URL as string | undefined) || 'http://localhost:8080';

export function AuthPage() {
  const navigate = useNavigate();

  const startGoogle = () => {
    window.location.href = `${API_BASE}/auth/google/login`;
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 to-purple-600 p-4">
      <Card className="w-full max-w-md p-8 space-y-6">
        <div className="text-center space-y-2">
          <h1 className="text-3xl font-bold">VOuTE</h1>
          <p className="text-sm text-muted-foreground">Join the voting revolution</p>
        </div>

        <div className="space-y-3">
          <Button 
            onClick={startGoogle} 
            variant="outline" 
            className="w-full h-11"
          >
            Continue with Google
          </Button>

          <Button 
            onClick={() => navigate('/login-otp')} 
            variant="outline"
            className="w-full h-11"
          >
            Login with OTP
          </Button>

          <Button 
            onClick={() => navigate('/login-password')} 
            variant="outline"
            className="w-full h-11"
          >
            Login with Password
          </Button>

          <Button 
            onClick={() => navigate('/signup-otp')} 
            className="w-full h-11 bg-indigo-600 hover:bg-indigo-700"
          >
            Sign Up with OTP
          </Button>
        </div>

        <div className="text-center text-xs text-muted-foreground pt-2">
          <p>Already have an account? All options above work for existing accounts.</p>
        </div>
      </Card>
    </div>
  );
}
