import React, { useState } from 'react';
import { Outlet, Link, useLocation } from 'react-router';
import { Button } from '../components/ui/button';
import { Moon, Sun, User, Plus } from 'lucide-react';
import { useTheme } from 'next-themes';
import { cn } from '../components/ui/utils';
import { VoteBalanceIndicator } from '../components/VoteBalanceIndicator';
import { CreatePollDialog } from '../components/CreatePollDialog';
import { usePolls } from '../contexts/PollsContext';

export function AppLayout() {
  const { theme, setTheme } = useTheme();
  const location = useLocation();
  const [availableVotes, setAvailableVotes] = useState(18);
  const maxVotes = 24;
  const [isCreatePollOpen, setIsCreatePollOpen] = useState(false);
  const { createPoll } = usePolls();

  const navItems = [
    { name: 'Home', path: '/home' },
    { name: 'My Polls', path: '/my-polls' },
    { name: 'Past Votes', path: '/past-votes' },
    { name: 'Bookmarks', path: '/bookmarks' },
  ];

  const handleCreatePoll = (title: string, options: string[]) => {
    createPoll(title, options);
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto px-4">
          <div className="flex h-16 items-center justify-between">
            {/* Left: Logo */}
            <Link to="/home" className="flex items-center space-x-2">
              <div className="flex items-center justify-center w-10 h-10 rounded-lg bg-gradient-to-br from-indigo-600 to-purple-600">
                <span className="text-white font-bold text-xl">V</span>
              </div>
              <span className="text-2xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">
                VOuTE
              </span>
            </Link>

            {/* Center: Navigation */}
            <nav className="hidden md:flex items-center gap-1 absolute left-1/2 -translate-x-1/2">
              {navItems.map(item => (
                <Link key={item.path} to={item.path}>
                  <Button
                    variant="ghost"
                    className={cn(
                      'text-sm font-medium transition-colors',
                      location.pathname === item.path
                        ? 'text-foreground bg-accent'
                        : 'text-muted-foreground hover:text-foreground'
                    )}
                  >
                    {item.name}
                  </Button>
                </Link>
              ))}
            </nav>

            {/* Right: Vote Balance, Theme toggle and Profile */}
            <div className="flex items-center gap-2">
              <VoteBalanceIndicator 
                availableVotes={availableVotes} 
                maxVotes={maxVotes} 
              />
              
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
                className="w-9 h-9"
              >
                <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
                <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
                <span className="sr-only">Toggle theme</span>
              </Button>

              <Link to="/profile">
                <Button
                  variant="ghost"
                  size="icon"
                  className={cn(
                    'w-9 h-9',
                    location.pathname === '/profile' && 'bg-accent'
                  )}
                >
                  <User className="h-5 w-5" />
                </Button>
              </Link>
            </div>
          </div>

          {/* Mobile Navigation */}
          <nav className="md:hidden flex items-center gap-1 pb-3 overflow-x-auto">
            {navItems.map(item => (
              <Link key={item.path} to={item.path}>
                <Button
                  variant="ghost"
                  size="sm"
                  className={cn(
                    'text-sm font-medium transition-colors whitespace-nowrap',
                    location.pathname === item.path
                      ? 'text-foreground bg-accent'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  {item.name}
                </Button>
              </Link>
            ))}
          </nav>
        </div>
      </header>

      <main className="container mx-auto px-4 py-8">
        <Outlet context={{ availableVotes, setAvailableVotes }} />
      </main>

      {/* Floating Action Button */}
      <Button
        onClick={() => setIsCreatePollOpen(true)}
        className="fixed bottom-6 right-6 h-14 w-14 rounded-full shadow-lg bg-indigo-600 hover:bg-indigo-700"
        size="icon"
      >
        <Plus className="h-6 w-6" />
      </Button>

      <CreatePollDialog
        open={isCreatePollOpen}
        onOpenChange={setIsCreatePollOpen}
        onCreatePoll={handleCreatePoll}
      />
    </div>
  );
}