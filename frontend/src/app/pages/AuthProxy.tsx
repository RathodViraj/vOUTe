import React, { useEffect } from 'react';

// Redirect any client-side navigation under /auth/* back to the backend
export function AuthProxy() {
  useEffect(() => {
    const API_BASE = ((import.meta as any).env?.VITE_API_BASE_URL as string | undefined) || 'http://localhost:8080';

    const target = `${API_BASE}${window.location.pathname}${window.location.search}`;
    
    window.location.replace(target);
  }, []);

  return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-sm text-muted-foreground">Redirecting to authentication service…</p>
    </div>
  );
}
