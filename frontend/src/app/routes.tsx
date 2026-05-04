import { createBrowserRouter, Navigate } from 'react-router';
import { AppLayout } from './layouts/AppLayout';
import { LoginPage } from './pages/LoginPage';
import { SignupPage } from './pages/SignupPage';
import { GoogleAuthCallbackPage } from './pages/GoogleAuthCallbackPage';
import { AuthProxy } from './pages/AuthProxy';
import { LoginOTPPage } from './pages/LoginOTPPage';
import { SignupOTPPage } from './pages/SignupOTPPage';
import { ForgotPasswordPage } from './pages/ForgotPasswordPage';
import { LoginPasswordPage } from './pages/LoginPasswordPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { HomePage } from './pages/HomePage';
import { MyPollsPage } from './pages/MyPollsPage';
import { PastVotesPage } from './pages/PastVotesPage';
import { BookmarksPage } from './pages/BookmarksPage';
import { ProfilePage } from './pages/ProfilePage';
import { CommentsPage } from './pages/CommentsPage';
import { hasAccessToken } from './lib/api';

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/signup',
    element: <SignupPage />,
  },
  {
    path: '/auth/google/callback',
    element: <GoogleAuthCallbackPage />,
  },
  {
    // Catch any client navigation under /auth/* and forward to backend
    path: '/auth/*',
    element: <AuthProxy />,
  },
  {
    path: '/login-otp',
    element: <LoginOTPPage />,
  },
  {
    path: '/login-password',
    element: <LoginPasswordPage />,
  },
  {
    path: '/signup-otp',
    element: <SignupOTPPage />,
  },
  // Catch-all route to show friendly 404 instead of router overlay
  {
    path: '*',
    element: <NotFoundPage />,
  },
  {
    path: '/forgot-password',
    element: <ForgotPasswordPage />,
  },
  {
    path: '/',
    element: <AppLayout />,
    children: [
      {
        index: true,
        element: hasAccessToken() ? <Navigate to="/home" replace /> : <Navigate to="/login" replace />,
      },
      {
        path: 'home',
        element: <HomePage />,
      },
      {
        path: 'my-polls',
        element: <MyPollsPage />,
      },
      {
        path: 'past-votes',
        element: <PastVotesPage />,
      },
      {
        path: 'bookmarks',
        element: <BookmarksPage />,
      },
      {
        path: 'profile',
        element: <ProfilePage />,
      },
      {
        path: 'comments/:pollId',
        element: <CommentsPage />,
      },
    ],
  },
]);
