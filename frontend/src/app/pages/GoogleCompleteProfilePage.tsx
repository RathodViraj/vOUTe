import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router';
import { toast } from 'sonner';
import { Button } from '../components/ui/button';
import { Card } from '../components/ui/card';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { setAccessToken, request } from '../lib/api';

export function GoogleCompleteProfilePage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [usernameAvailable, setUsernameAvailable] = useState<boolean | null>(null);
  const [checkingUsername, setCheckingUsername] = useState(false);

  // Check username availability with bloom filter
  const checkUsernameAvailability = async (name: string) => {
    if (!name || name.length < 3) {
      setUsernameAvailable(null);
      return;
    }

    setCheckingUsername(true);
    try {
      const response = await request<{ exists: boolean }>(
        `/users/check?username=${encodeURIComponent(name)}`,
        { method: 'GET' }
      );
      
      // request() already returns `data`, so read `exists` directly.
      setUsernameAvailable(!response.exists);
    } catch (error) {
      // If error, assume username is taken to be safe
      setUsernameAvailable(false);
    } finally {
      setCheckingUsername(false);
    }
  };

  // Debounce username check
  useEffect(() => {
    const timer = setTimeout(() => {
      if (username.length >= 3) {
        checkUsernameAvailability(username);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [username]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!username || username.length < 3) {
      toast.error('Username must be at least 3 characters');
      return;
    }

    if (!password || password.length < 6) {
      toast.error('Password must be at least 6 characters');
      return;
    }

    if (usernameAvailable === false) {
      toast.error('Username is already taken');
      return;
    }

    setIsLoading(true);
    try {
      const response = await request<{ access_token: string; refresh_token: string }>(
        '/auth/google/complete-profile',
        {
          method: 'POST',
          body: JSON.stringify({ username, password }),
        }
      );

      if (response.access_token) {
        setAccessToken(response.access_token);
        toast.success('Account created successfully!');
        navigate('/home', { replace: true });
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Failed to complete signup';
      toast.error(errorMsg);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 to-purple-600 p-4">
      <Card className="w-full max-w-md p-8 space-y-6">
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-bold">Complete Your Profile</h1>
          <p className="text-sm text-muted-foreground">Choose your username and password</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="username">Username</Label>
            <Input
              id="username"
              placeholder="Enter username (3+ characters)"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={isLoading}
              className={
                usernameAvailable === true ? 'border-green-500' :
                usernameAvailable === false ? 'border-red-500' :
                ''
              }
            />
            {checkingUsername && (
              <p className="text-xs text-muted-foreground">Checking availability...</p>
            )}
            {!checkingUsername && usernameAvailable === true && (
              <p className="text-xs text-green-600">✓ Username available</p>
            )}
            {!checkingUsername && usernameAvailable === false && (
              <p className="text-xs text-red-600">✗ Username already taken</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              placeholder="Enter password (6+ characters)"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={isLoading}
            />
          </div>

          <Button
            type="submit"
            disabled={
              isLoading ||
              !username ||
              !password ||
              usernameAvailable === false ||
              checkingUsername
            }
            className="w-full bg-indigo-600 hover:bg-indigo-700"
          >
            {isLoading ? 'Completing signup...' : 'Complete Signup'}
          </Button>
        </form>

        <div className="text-center">
          <button
            onClick={() => navigate('/auth', { replace: true })}
            className="text-sm text-indigo-600 hover:underline"
          >
            Back to login
          </button>
        </div>
      </Card>
    </div>
  );
}
