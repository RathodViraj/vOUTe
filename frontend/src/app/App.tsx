import React from 'react';
import { RouterProvider } from 'react-router';
import { ThemeProvider } from 'next-themes';
import { AuthProvider } from './contexts/AuthContext';
import { SettingsProvider } from './contexts/SettingsContext';
import { PollsProvider } from './contexts/PollsContext';
import { Toaster } from './components/ui/sonner';
import { router } from './routes';

export default function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem>
      <AuthProvider>
        <SettingsProvider>
          <PollsProvider>
            <RouterProvider router={router} />
            <Toaster />
          </PollsProvider>
        </SettingsProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}
