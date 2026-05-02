import React, { useState } from 'react';
import { useNavigate } from 'react-router';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Card } from '../components/ui/card';
import { toast } from 'sonner';
import { requestOTP as apiRequestOTP, verifyOTP as apiVerifyOTP, signupWithOTP as apiSignupWithOTP } from '../lib/api';

export function SignupOTPPage() {
  const navigate = useNavigate();
  const [step, setStep] = useState<'email' | 'otp' | 'signup'>('signup');
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    otp: '',
  });
  const [loading, setLoading] = useState(false);
  const [otpSent, setOtpSent] = useState(false);
  const [verificationToken, setVerificationToken] = useState<string | null>(null);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const requestOTP = async () => {
    if (!formData.email) {
      toast.error('Please enter your email');
      return;
    }

    setLoading(true);
    try {
      await apiRequestOTP(formData.email);
      setOtpSent(true);
      toast.success('OTP sent to your email');
    } catch (error) {
      toast.error('Error sending OTP');
    } finally {
      setLoading(false);
    }
  };

  const completeSignup = async () => {
    if (!formData.username || !formData.password || !verificationToken) {
      toast.error('Please fill in all fields');
      return;
    }

    setLoading(true);
    try {
      const data = await apiSignupWithOTP(formData.username, formData.email, formData.password, verificationToken as string);
      if (data?.access_token) {
        toast.success('Signup successful!');
        navigate('/home', { replace: true });
      } else {
        toast.error('Signup failed');
      }
    } catch (error) {
      toast.error((error as Error).message || 'Error during signup');
    } finally {
      setLoading(false);
    }
  };

  const verifyOtp = async () => {
    if (!formData.otp || !formData.email) {
      toast.error('Please enter the OTP and email');
      return;
    }
    setLoading(true);
    try {
      const token = await apiVerifyOTP(formData.email, formData.otp);
      if (token) {
        setVerificationToken(token);
        toast.success('Email verified — you can complete signup');
      } else {
        toast.error('Invalid or expired OTP');
      }
    } catch (err) {
      toast.error('Failed to verify OTP');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 to-purple-600 p-4">
      <Card className="w-full max-w-md p-6 space-y-4">
        <div className="text-center space-y-2 mb-6">
          <h1 className="text-2xl font-bold">Create Account</h1>
          <p className="text-sm text-muted-foreground">Sign up with OTP verification</p>
        </div>

        <div className="space-y-4">
          {/* Username */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Username</label>
            <Input
              type="text"
              name="username"
              placeholder="john_doe"
              value={formData.username}
              onChange={handleInputChange}
              disabled={loading || otpSent}
            />
          </div>

          {/* Email */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Email</label>
            <div className="flex gap-2">
              <Input
                type="email"
                name="email"
                placeholder="you@example.com"
                value={formData.email}
                onChange={handleInputChange}
                disabled={loading || otpSent}
              />
              {!otpSent && (
                <Button
                  onClick={requestOTP}
                  disabled={loading || !formData.email}
                  className="bg-indigo-600 hover:bg-indigo-700"
                >
                  {loading ? 'Sending...' : 'Send OTP'}
                </Button>
              )}
            </div>
          </div>

          {/* OTP */}
          {otpSent && (
            <div className="space-y-2">
              <label className="text-sm font-medium">OTP Code (6 digits)</label>
              <div className="flex gap-2">
                <Input
                  type="text"
                  name="otp"
                  placeholder="123456"
                  maxLength="6"
                  value={formData.otp}
                  onChange={handleInputChange}
                  disabled={loading || !!verificationToken}
                />
                {!verificationToken && (
                  <Button onClick={verifyOtp} disabled={loading || !formData.otp} className="bg-indigo-600 hover:bg-indigo-700">
                    {loading ? 'Verifying...' : 'Verify OTP'}
                  </Button>
                )}
              </div>
              <p className="text-xs text-muted-foreground">Check your email for the code</p>
            </div>
          )}

          {/* Password */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Password</label>
            <Input
              type="password"
              name="password"
              placeholder="••••••••"
              value={formData.password}
              onChange={handleInputChange}
              disabled={loading}
            />
          </div>

          {/* Submit Button */}
          <Button
            onClick={completeSignup}
            disabled={loading || !verificationToken}
            className="w-full bg-indigo-600 hover:bg-indigo-700"
          >
            {loading ? 'Creating Account...' : 'Create Account'}
          </Button>
        </div>

        {/* Login Link */}
        <div className="text-center text-sm">
          Already have an account?{' '}
          <button
            onClick={() => navigate('/login')}
            className="text-indigo-600 hover:text-indigo-700 font-medium"
          >
            Login
          </button>
        </div>
      </Card>
    </div>
  );
}
