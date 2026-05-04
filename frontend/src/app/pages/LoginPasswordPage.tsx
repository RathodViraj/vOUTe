import React, { useState } from 'react';
import { useNavigate } from 'react-router';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Card } from '../components/ui/card';
import { toast } from 'sonner';
import { login } from '../lib/api';

export function LoginPasswordPage() {
  const navigate = useNavigate();
  const [identifier, setIdentifier] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const handleLogin = async () => {
    if (!identifier || !password) {
      toast.error('Please enter your email/username and password');
      return;
    }

    setLoading(true);
    try {
      await login(identifier, password);
      toast.success('Logged in');
      navigate('/home', { replace: true });
    } catch (err) {
      toast.error((err as Error).message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 to-purple-600 p-4">
      <Card className="w-full max-w-md p-6 space-y-4">
        <div className="text-center space-y-2 mb-6">
          <h1 className="text-2xl font-bold">Sign in</h1>
          <p className="text-sm text-muted-foreground">Sign in with your email or username</p>
        </div>

        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium">Email or Username</label>
            <Input value={identifier} onChange={(e) => setIdentifier(e.target.value)} placeholder="you@example.com or username" />
          </div>

          <div>
            <label className="text-sm font-medium">Password</label>
            <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Your password" />
          </div>

          <Button onClick={handleLogin} disabled={loading || !identifier || !password} className="w-full bg-indigo-600 hover:bg-indigo-700">
            {loading ? 'Signing in...' : 'Sign in'}
          </Button>

          <div className="text-center text-sm">
            <button onClick={() => navigate('/forgot-password')} className="text-indigo-600 underline">Forgot password?</button>
          </div>
        </div>
      </Card>
    </div>
  );
}
