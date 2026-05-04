import React from 'react';
import { useNavigate } from 'react-router';
import { Button } from '../components/ui/button';
import { Card } from '../components/ui/card';
import { toast } from 'sonner';
// Use backend base URL for OAuth redirects so dev server doesn't intercept
const API_BASE = ((import.meta as any).env?.VITE_API_BASE_URL as string | undefined) || 'http://localhost:8080';
console.debug('[SignupPage] API_BASE =', API_BASE);

export function SignupPage() {
  const navigate = useNavigate();

  const startGoogle = () => {
    // Open backend Google auth endpoint
    window.location.href = `${API_BASE}/auth/google/login`;
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 to-purple-600 p-4">
      <Card className="w-full max-w-md p-6 space-y-4">
        <div className="text-center space-y-2 mb-6">
          <h1 className="text-2xl font-bold">Create Account</h1>
          <p className="text-sm text-muted-foreground">Sign up with OTP verification or continue with Google</p>
        </div>

        <div className="space-y-4">
          <Button onClick={() => navigate('/signup-otp')} className="w-full bg-indigo-600 hover:bg-indigo-700">
            Sign up with OTP verification
          </Button>

          <Button variant="outline" onClick={startGoogle} className="w-full">
            Continue with Google
          </Button>

          <div className="text-center text-sm">
            Direct password signup is disabled. Use OTP or Google to create an account.
          </div>
        </div>
      </Card>
    </div>
  );
}
