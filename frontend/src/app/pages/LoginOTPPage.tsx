import React, { useState } from 'react';
import { useNavigate } from 'react-router';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Card } from '../components/ui/card';
import { toast } from 'sonner';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { requestOTP, loginWithOTP, loginWithOTPUsername } from '../lib/api';

export function LoginOTPPage() {
  const navigate = useNavigate();
  const [loginType, setLoginType] = useState<'email' | 'username'>('email');
  const [identifier, setIdentifier] = useState('');
  const [otp, setOtp] = useState('');
  const [loading, setLoading] = useState(false);
  const [otpSent, setOtpSent] = useState(false);

  const requestOTPHandler = async () => {
    if (!identifier) {
      toast.error(`Please enter your ${loginType}`);
      return;
    }

    setLoading(true);
    try {
      await requestOTP(loginType === 'email' ? identifier : undefined, loginType === 'username' ? identifier : undefined);
      setOtpSent(true);
      toast.success('OTP sent successfully');
    } catch (error) {
      toast.error((error as Error).message || 'Failed to send OTP');
    } finally {
      setLoading(false);
    }
  };

  const loginWithOTPHandler = async () => {
    if (!otp) {
      toast.error('Please enter OTP');
      return;
    }

    setLoading(true);
    try {
      if (loginType === 'email') {
        await loginWithOTP(identifier, otp);
      } else {
        await loginWithOTPUsername(identifier, otp);
      }
      toast.success('Login successful!');
      navigate('/home', { replace: true });
    } catch (error) {
      toast.error((error as Error).message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 to-purple-600 p-4">
      <Card className="w-full max-w-md p-6 space-y-4">
        <div className="text-center space-y-2 mb-6">
          <h1 className="text-2xl font-bold">Login with OTP</h1>
          <p className="text-sm text-muted-foreground">Secure one-time password authentication</p>
        </div>

        <Tabs 
          value={loginType} 
          onValueChange={(val) => {
            setLoginType(val as 'email' | 'username');
            setIdentifier('');
            setOtp('');
            setOtpSent(false);
          }}
        >
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="email">Email</TabsTrigger>
            <TabsTrigger value="username">Username</TabsTrigger>
          </TabsList>

          <TabsContent value="email" className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Email Address</label>
              <Input
                type="email"
                placeholder="you@example.com"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                disabled={loading || otpSent}
              />
            </div>

            {!otpSent && (
              <Button
                onClick={requestOTPHandler}
                disabled={loading || !identifier}
                className="w-full bg-indigo-600 hover:bg-indigo-700"
              >
                {loading ? 'Sending...' : 'Send OTP'}
              </Button>
            )}

            {otpSent && (
              <>
                <div className="space-y-2">
                  <label className="text-sm font-medium">OTP Code</label>
                  <Input
                    type="text"
                    placeholder="123456"
                    maxLength="6"
                    value={otp}
                    onChange={(e) => setOtp(e.target.value)}
                    disabled={loading}
                  />
                  <p className="text-xs text-muted-foreground">Check your email for the 6-digit code</p>
                </div>

                <Button
                  onClick={loginWithOTPHandler}
                  disabled={loading || !otp}
                  className="w-full bg-indigo-600 hover:bg-indigo-700"
                >
                  {loading ? 'Logging in...' : 'Login'}
                </Button>

                <Button
                  variant="outline"
                  onClick={() => {
                    setOtpSent(false);
                    setOtp('');
                  }}
                  disabled={loading}
                  className="w-full"
                >
                  Send OTP Again
                </Button>
              </>
            )}
          </TabsContent>

          <TabsContent value="username" className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Username</label>
              <Input
                type="text"
                placeholder="john_doe"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                disabled={loading || otpSent}
              />
            </div>

            {!otpSent && (
              <Button
                onClick={requestOTPHandler}
                disabled={loading || !identifier}
                className="w-full bg-indigo-600 hover:bg-indigo-700"
              >
                {loading ? 'Sending...' : 'Send OTP'}
              </Button>
            )}

            {otpSent && (
              <>
                <div className="space-y-2">
                  <label className="text-sm font-medium">OTP Code</label>
                  <Input
                    type="text"
                    placeholder="123456"
                    maxLength="6"
                    value={otp}
                    onChange={(e) => setOtp(e.target.value)}
                    disabled={loading}
                  />
                  <p className="text-xs text-muted-foreground">Check your email for the 6-digit code</p>
                </div>

                <Button
                  onClick={loginWithOTPHandler}
                  disabled={loading || !otp}
                  className="w-full bg-indigo-600 hover:bg-indigo-700"
                >
                  {loading ? 'Logging in...' : 'Login'}
                </Button>

                <Button
                  variant="outline"
                  onClick={() => {
                    setOtpSent(false);
                    setOtp('');
                  }}
                  disabled={loading}
                  className="w-full"
                >
                  Send OTP Again
                </Button>
              </>
            )}
          </TabsContent>
        </Tabs>

        {/* Navigation Links */}
        <div className="pt-4 text-center text-sm space-y-2">
          <div>
            Don't have an account?{' '}
            <button
              onClick={() => navigate('/signup')}
              className="text-indigo-600 hover:text-indigo-700 font-medium"
            >
              Sign up
            </button>
          </div>
          <div>
            <button
              onClick={() => navigate('/login')}
              className="text-indigo-600 hover:text-indigo-700 font-medium"
            >
              Use password login
            </button>
          </div>
        </div>
      </Card>
    </div>
  );
}
