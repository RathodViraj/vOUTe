import React, { useEffect } from 'react';

// Redirect any client-side navigation under /auth/* back to the backend
export function AuthProxy() {
  useEffect(() => {
    const API_BASE = ((import.meta as any).env?.VITE_API_BASE_URL as string | undefined) || 'http://localhost:8080';
    // preserve path + query
    const target = `${API_BASE}${window.location.pathname}${window.location.search}`;
    console.debug('[AuthProxy] redirecting to backend:', target);
    // full navigation to backend (avoids SPA handling)
    window.location.replace(target);
  }, []);

  return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-sm text-muted-foreground">Redirecting to authentication service…</p>
    </div>
  );
}
