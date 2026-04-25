import React, { useState, useEffect } from 'react';
import { Zap } from 'lucide-react';
import { Popover, PopoverContent, PopoverTrigger } from './ui/popover';
import { Button } from './ui/button';

interface VoteBalanceIndicatorProps {
  availableVotes: number;
  maxVotes: number;
}

export function VoteBalanceIndicator({ availableVotes, maxVotes }: VoteBalanceIndicatorProps) {
  const [timeUntilNext, setTimeUntilNext] = useState({ minutes: 0, seconds: 0 });

  useEffect(() => {
    const updateCountdown = () => {
      const now = new Date();
      const minutes = 59 - now.getMinutes();
      const seconds = 59 - now.getSeconds();
      setTimeUntilNext({ minutes, seconds });
    };

    updateCountdown();
    const interval = setInterval(updateCountdown, 1000);

    return () => clearInterval(interval);
  }, []);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button 
          variant="ghost" 
          className="h-9 px-3 gap-1.5 font-semibold hover:bg-accent"
        >
          <Zap className="h-4 w-4 text-indigo-600 fill-indigo-600" />
          <span className="text-sm">{availableVotes}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-56 p-3" align="end">
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Available Votes</span>
            <span className="text-lg font-bold text-indigo-600">
              {availableVotes}/{maxVotes}
            </span>
          </div>
          <div className="text-xs text-muted-foreground space-y-1">
            <p>Next vote in <span className="font-semibold text-foreground">{timeUntilNext.minutes}m {timeUntilNext.seconds}s</span></p>
            <p>Regenerates 1 vote per hour</p>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
