import { createBrowserRouter, Navigate } from 'react-router';
import { AppLayout } from './layouts/AppLayout';
import { LoginPage } from './pages/LoginPage';
import { SignupPage } from './pages/SignupPage';
import { ForgotPasswordPage } from './pages/ForgotPasswordPage';
import { HomePage } from './pages/HomePage';
import { MyPollsPage } from './pages/MyPollsPage';
import { PastVotesPage } from './pages/PastVotesPage';
import { BookmarksPage } from './pages/BookmarksPage';
import { ProfilePage } from './pages/ProfilePage';
import { CommentsPage } from './pages/CommentsPage';

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
    path: '/forgot-password',
    element: <ForgotPasswordPage />,
  },
  {
    path: '/',
    element: <AppLayout />,
    children: [
      {
        index: true,
        element: <Navigate to="/login" replace />,
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
