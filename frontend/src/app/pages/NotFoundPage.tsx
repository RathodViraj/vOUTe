import React from 'react';
import { useNavigate } from 'react-router';
import { Button } from '../components/ui/button';

export function NotFoundPage() {
  const navigate = useNavigate();
  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="text-center">
        <h2 className="text-2xl font-semibold mb-2">Page not found</h2>
        <p className="text-sm text-muted-foreground mb-4">The page you are looking for does not exist.</p>
        <div className="flex items-center justify-center gap-2">
          <Button onClick={() => navigate('/', { replace: true })}>Go home</Button>
        </div>
      </div>
    </div>
  );
}
