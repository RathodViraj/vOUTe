import React, { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import { toast } from 'sonner';
import { setAccessToken } from '../lib/api';

export function GoogleAuthCallbackPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  useEffect(() => {
    const token = searchParams.get('access_token');

    if (!token) {
      toast.error('Google login failed');
      navigate('/auth', { replace: true });
      return;
    }

    setAccessToken(token);
    toast.success('Logged in with Google successfully');
    navigate('/home', { replace: true });
  }, [navigate, searchParams]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950 px-4">
      <p className="text-muted-foreground">Completing Google sign in...</p>
    </div>
  );
}
