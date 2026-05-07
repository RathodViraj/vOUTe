import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { RadioGroup, RadioGroupItem } from './ui/radio-group';
import { Label } from './ui/label';
import { Badge } from './ui/badge';
import { Bookmark, BookmarkCheck, Minus, Plus, MessageSquare, Lock } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import type { Poll } from '../lib/types';
import { format } from 'date-fns';
import { toast } from 'sonner';
import { getPollHistory } from '../lib/api';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from './ui/alert-dialog';

type ChartRow = Record<string, number | Date | string> & {
  timestamp: Date;
  time: string;
  totalVotes: number;
};

const TIMELINE_ITEM_WIDTH = 72;
const TIMELINE_TRIGGER_PADDING = 2;

function mergeHistoryRows(...groups: Array<Array<{ timestamp: Date; optionId: string; votes: number }>>): Array<{ timestamp: Date; optionId: string; votes: number }> {
  const merged = new Map<string, { timestamp: Date; optionId: string; votes: number }>();

  for (const group of groups) {
    for (const row of group) {
      const key = `${row.timestamp.getTime()}|${row.optionId}`;
      merged.set(key, row);
    }
  }

  return [...merged.values()].sort((left, right) => left.timestamp.getTime() - right.timestamp.getTime());
}

function buildChartRows(historyData: Array<{ timestamp: Date; optionId: string; votes: number }>, optionIds: string[], optionLabels: Record<string, string>): ChartRow[] {
  const grouped = new Map<number, Record<string, number | Date | string>>();

  for (const item of historyData) {
    const timestampKey = item.timestamp.getTime();
    const existing = grouped.get(timestampKey) || {
      timestamp: item.timestamp,
      time: format(item.timestamp, 'HH:mm'),
    };

    existing[item.optionId] = item.votes;
    grouped.set(timestampKey, existing);
  }

  return [...grouped.values()]
    .sort((left, right) => (left.timestamp as Date).getTime() - (right.timestamp as Date).getTime())
    .map((entry) => {
      const totalVotes = optionIds.reduce((sum, optionId) => sum + Number(entry[optionId] ?? 0), 0);
      const nextEntry: ChartRow = {
        ...entry,
        totalVotes,
      } as ChartRow;

      for (const optionId of optionIds) {
        const rawValue = Number(entry[optionId] ?? 0);
        nextEntry[`${optionId}_percentage`] = totalVotes > 0 ? (rawValue / totalVotes) * 100 : 0;
      }

      return nextEntry;
    });
}

interface PollCardProps {
  poll: Poll;
  isBookmarked?: boolean;
  onBookmarkToggle?: (pollId: string) => Promise<void> | void;
  showChart?: boolean;
  availableVotes?: number;
  onVoteSubmit?: (pollId: string, optionId: string, voteCount: number) => Promise<void>;
  showHistoricalDataByDefault?: boolean;
  readOnly?: boolean;
  onClosePoll?: (pollId: string) => Promise<void>;
  onDeletePoll?: (pollId: string) => Promise<void>;
}

const RANGE_OPTIONS = [
  { value: 'live', label: 'Live' },
  { value: '-1', label: 'Last 24h' },
  { value: '-2', label: '24-48h ago' },
  { value: '-3', label: '2-3 days ago' },
  { value: '-4', label: '3-4 days ago' },
  { value: '-5', label: '4-5 days ago' },
  { value: '-6', label: '5-6 days ago' },
  { value: '-7', label: 'Up to 7 days' },
];

export function PollCard({
  poll,
  isBookmarked = false,
  onBookmarkToggle,
  showChart = true,
  availableVotes = 24,
  onVoteSubmit,
  showHistoricalDataByDefault = false,
  readOnly = false,
  onClosePoll,
  onDeletePoll,
}: PollCardProps) {
  const navigate = useNavigate();
  const [selectedOption, setSelectedOption] = useState<string>('');
  const [voteCount, setVoteCount] = useState<number>(1);
  const [localVotes, setLocalVotes] = useState(poll.options);
  const [isLoading, setIsLoading] = useState(false);
  const [isDataRevealed, setIsDataRevealed] = useState(false);
  const [chartMode, setChartMode] = useState<'count' | 'percentage'>('count');
  const [historicalData, setHistoricalData] = useState<Array<{ timestamp: Date; optionId: string; votes: number }>>([]);
  const [liveData, setLiveData] = useState<Array<{ timestamp: Date; optionId: string; votes: number }>>(poll.history || []);
  const [visibleStartIndex, setVisibleStartIndex] = useState(0);
  const [windowSize, setWindowSize] = useState(15);
  const [hasOlderHistory, setHasOlderHistory] = useState(true);
  const [isFetchingOlderHistory, setIsFetchingOlderHistory] = useState(false);
  const [isAtLiveEdge, setIsAtLiveEdge] = useState(true);
  const longPressTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const olderFetchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const timelineRef = useRef<HTMLDivElement | null>(null);

  const totalVotes = localVotes.reduce((sum, option) => sum + option.votes, 0);
  const shouldShowData = showHistoricalDataByDefault || isDataRevealed;

  useEffect(() => {
    const updateWindowSize = () => {
      const width = window.innerWidth;
      if (width < 768) {
        setWindowSize(12);
      } else if (width < 1280) {
        setWindowSize(15);
      } else {
        setWindowSize(18);
      }
    };

    updateWindowSize();
    window.addEventListener('resize', updateWindowSize);

    return () => {
      window.removeEventListener('resize', updateWindowSize);
    };
  }, []);

  useEffect(() => {
    setLocalVotes(poll.options);
  }, [poll.options]);

  useEffect(() => {
    setLiveData((prev) => mergeHistoryRows(prev, poll.history || []));
  }, [poll.history]);

  useEffect(() => {
    setHistoricalData([]);
    setLiveData(poll.history || []);
    setVisibleStartIndex(0);
    setHasOlderHistory(true);
    setIsFetchingOlderHistory(false);
    setIsAtLiveEdge(true);
  }, [poll.id]);

  useEffect(() => {
    // On mount / poll change fetch the latest 24-hour window so timeline isn't empty
    let mounted = true;

    const fetchLatest = async () => {
      setIsFetchingOlderHistory(true);
      try {
        const nowIso = new Date().toISOString();
        const rows = await getPollHistory(poll.id, nowIso);
        if (!mounted) return;

        if (rows.length === 0) {
          setHasOlderHistory(false);
          setHistoricalData([]);
          return;
        }

        const normalizedRows = rows.map((row) => ({
          timestamp: new Date(row.Timestamp),
          optionId: String(row.OptionID),
          votes: Number(row.VoteCount || 0),
        }));

        setHistoricalData(() => mergeHistoryRows(normalizedRows));
        // scroll to live edge after render
        requestAnimationFrame(() => {
          if (timelineRef.current) timelineRef.current.scrollLeft = timelineRef.current.scrollWidth;
        });
      } catch (err) {
        // ignore - timeline remains usable without history
      } finally {
        if (mounted) setIsFetchingOlderHistory(false);
      }
    };

    void fetchLatest();

    return () => { mounted = false; };
  }, [poll.id]);

  const mergedHistory = useMemo(() => {
    const optionIds = poll.options.map((option) => String(option.id));
    const optionLabels = poll.options.reduce<Record<string, string>>((accumulator, option) => {
      accumulator[String(option.id)] = option.text;
      return accumulator;
    }, {});

    return buildChartRows(mergeHistoryRows(historicalData, liveData), optionIds, optionLabels);
  }, [historicalData, liveData, poll.options]);

  useEffect(() => {
    if (mergedHistory.length === 0) {
      setVisibleStartIndex(0);
      return;
    }

    if (isAtLiveEdge) {
      setVisibleStartIndex(Math.max(0, mergedHistory.length - windowSize));
      requestAnimationFrame(() => {
        if (!timelineRef.current) return;
        timelineRef.current.scrollLeft = timelineRef.current.scrollWidth;
      });
      return;
    }

    setVisibleStartIndex((current) => Math.min(current, Math.max(0, mergedHistory.length - windowSize)));
  }, [isAtLiveEdge, mergedHistory.length, windowSize]);

  const visibleHistory = useMemo(() => {
    if (mergedHistory.length <= windowSize) {
      return mergedHistory;
    }

    return mergedHistory.slice(visibleStartIndex, visibleStartIndex + windowSize);
  }, [mergedHistory, visibleStartIndex, windowSize]);

  const fetchOlderHistory = useCallback(async () => {
    if (!hasOlderHistory || isFetchingOlderHistory || mergedHistory.length === 0) {
      return;
    }

    const oldestVisible = mergedHistory[0]?.timestamp;
    if (!oldestVisible) {
      return;
    }

    setIsFetchingOlderHistory(true);

    try {
      const rows = await getPollHistory(poll.id, oldestVisible.toISOString());
      if (rows.length === 0) {
        setHasOlderHistory(false);
        return;
      }

      const normalizedRows = rows.map((row) => ({
        timestamp: new Date(row.Timestamp),
        optionId: String(row.OptionID),
        votes: Number(row.VoteCount || 0),
      }));

      setHistoricalData((prev) => mergeHistoryRows(normalizedRows, prev));

      requestAnimationFrame(() => {
        if (timelineRef.current) {
          timelineRef.current.scrollLeft += normalizedRows.length * TIMELINE_ITEM_WIDTH;
        }
      });

      setVisibleStartIndex((current) => current + normalizedRows.length);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to load older history');
    } finally {
      setIsFetchingOlderHistory(false);
    }
  }, [hasOlderHistory, isFetchingOlderHistory, mergedHistory, poll.id]);

  useEffect(() => {
    return () => {
      if (olderFetchTimerRef.current) {
        clearTimeout(olderFetchTimerRef.current);
      }
    };
  }, []);

  const scheduleOlderFetch = useCallback(() => {
    if (olderFetchTimerRef.current) {
      clearTimeout(olderFetchTimerRef.current);
    }

    olderFetchTimerRef.current = setTimeout(() => {
      void fetchOlderHistory();
    }, 180);
  }, [fetchOlderHistory]);

  const handleTimelineScroll = useCallback(() => {
    const container = timelineRef.current;
    if (!container) {
      return;
    }

    const nextStart = Math.max(0, Math.floor(container.scrollLeft / TIMELINE_ITEM_WIDTH));
    setVisibleStartIndex(nextStart);

    const atLiveEdge = container.scrollLeft + container.clientWidth >= container.scrollWidth - TIMELINE_ITEM_WIDTH;
    setIsAtLiveEdge(atLiveEdge);

    const fetchThreshold = windowSize <= 15 ? 15 : 20;
    if (nextStart <= fetchThreshold && hasOlderHistory && !isFetchingOlderHistory) {
      scheduleOlderFetch();
    }
  }, [hasOlderHistory, isFetchingOlderHistory, scheduleOlderFetch, windowSize]);

  const handleVoteCountChange = (delta: number) => {
    // Strictly prevent going beyond available votes
    const maxAllowed = Math.min(100, availableVotes);
    const newCount = Math.max(1, Math.min(maxAllowed, voteCount + delta));
    setVoteCount(newCount);
  };

  const handleVote = async () => {
    if (!poll.isLive) {
      toast.error('This poll is closed');
      return;
    }

    if (!selectedOption) {
      toast.error('Please select an option to vote');
      return;
    }

    if (voteCount > availableVotes) {
      toast.error('Not enough votes available');
      return;
    }

    setIsLoading(true);

    try {
      if (onVoteSubmit) {
        await onVoteSubmit(poll.id, selectedOption, voteCount);
      }

      const updatedVotes = localVotes.map(option =>
        option.id === selectedOption
          ? { ...option, votes: option.votes + voteCount }
          : option
      );
      setLocalVotes(updatedVotes);

      toast.success(`Successfully voted with ${voteCount} vote${voteCount > 1 ? 's' : ''}!`);
      setSelectedOption('');
      setVoteCount(1);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to submit vote');
    } finally {
      setIsLoading(false);
    }
  };

  const chartData = visibleHistory;
  const optionIds = useMemo(() => poll.options.map((option) => String(option.id)), [poll.options]);
  const chartMinValue = useMemo(() => {
    if (chartMode === 'percentage' || chartData.length === 0) {
      return 0;
    }

    const values = chartData.flatMap((entry) => optionIds.map((optionId) => Number(entry[optionId] ?? 0)));
    const minValue = Math.min(...values);
    if (!Number.isFinite(minValue)) {
      return 0;
    }

    const padded = minValue - Math.max(1, minValue * 0.05);
    return Math.max(0, padded);
  }, [chartData, chartMode, optionIds]);

  const chartYAxisProps = chartMode === 'percentage'
    ? {
        domain: [0, 100] as [number, number],
        tickFormatter: (value: number) => `${value.toFixed(0)}%`,
      }
    : {
        domain: [chartMinValue, 'auto'] as [number, 'auto'],
        tickFormatter: (value: number) => value.toLocaleString(),
      };

  const chartTooltipFormatter = (value: number) => {
    if (chartMode === 'percentage') {
      return [`${value.toFixed(1)}%`, 'Vote share'];
    }

    return [value.toLocaleString(), 'Votes'];
  };

  const colors = ['#6366f1', '#8b5cf6', '#ec4899', '#f59e0b'];

  const handleDoubleClick = () => {
    if (!showHistoricalDataByDefault) {
      setIsDataRevealed(prev => !prev);
    }
  };

  const handleTouchStart = () => {
    if (!showHistoricalDataByDefault) {
      longPressTimerRef.current = setTimeout(() => {
        setIsDataRevealed(prev => !prev);
      }, 500);
    }
  };

  const handleTouchEnd = () => {
    if (longPressTimerRef.current) {
      clearTimeout(longPressTimerRef.current);
      longPressTimerRef.current = null;
    }
  };

  return (
    <Card
      className="p-6 space-y-4 hover:shadow-lg transition-shadow"
      onDoubleClick={handleDoubleClick}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <h3 className="text-xl font-semibold mb-2">{poll.title}</h3>
          <div className="flex items-center gap-2">
            {shouldShowData && (
              <Badge variant="secondary" className="text-xs">
                {totalVotes.toLocaleString()} votes
              </Badge>
            )}
            {poll.isLive ? (
              <Badge className="text-xs bg-green-500 hover:bg-green-600">
                <span className="inline-block w-2 h-2 bg-white rounded-full mr-1.5 animate-pulse"></span>
                Live
              </Badge>
            ) : (
              <Badge variant="secondary" className="text-xs text-muted-foreground">
                Closed
              </Badge>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {poll.isLive && onClosePoll && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <Lock className="w-4 h-4 mr-2" />
                  Close Poll
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Close this poll?</AlertDialogTitle>
                  <AlertDialogDescription>
                    Once closed, users will no longer be able to vote on this poll. This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => onClosePoll(poll.id)}
                    className="bg-indigo-600 hover:bg-indigo-700"
                  >
                    Close Poll
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}

          {onDeletePoll && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="destructive" size="sm">
                  Delete Poll
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete this poll?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This permanently removes the poll and all associated data. This cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => onDeletePoll(poll.id)}
                    className="bg-red-600 hover:bg-red-700"
                  >
                    Delete Poll
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}

          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate(`/comments/${poll.id}`)}
            className="shrink-0"
          >
            <MessageSquare className="h-5 w-5" />
          </Button>
          {onBookmarkToggle && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onBookmarkToggle(poll.id)}
              className="shrink-0"
            >
              {isBookmarked ? (
                <BookmarkCheck className="h-5 w-5 text-indigo-600" />
              ) : (
                <Bookmark className="h-5 w-5" />
              )}
            </Button>
          )}
        </div>
      </div>

      {readOnly ? (
        <div className="space-y-2">
          {localVotes.map(option => {
            const percentage = totalVotes > 0 ? (option.votes / totalVotes) * 100 : 0;
            const isUserChoice = poll.userVote?.optionId === option.id;

            return (
              <div key={option.id} className="relative">
                <div className={`flex items-center space-x-3 p-3 rounded-lg border ${isUserChoice ? 'border-indigo-500 bg-indigo-500/10' : 'bg-accent/20'}`}>
                  <div className="w-4 h-4 rounded-full border border-muted-foreground/40 bg-background shrink-0" />
                  <div className="flex-1 flex items-center justify-between gap-4">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="font-medium truncate">{option.text}</span>
                      {isUserChoice && (
                        <Badge variant="secondary" className="text-[10px] uppercase tracking-wide">
                          Your vote
                        </Badge>
                      )}
                    </div>
                    {shouldShowData && (
                      <span className="text-sm text-muted-foreground whitespace-nowrap">
                        {option.votes.toLocaleString()} ({percentage.toFixed(1)}%)
                        {isUserChoice && poll.userVote ? ` · ${poll.userVote.voteCount} voted` : ''}
                      </span>
                    )}
                  </div>
                </div>
                {shouldShowData && (
                  <div
                    className="absolute bottom-0 left-0 h-1 bg-indigo-500 rounded-full transition-all duration-500"
                    style={{ width: `${percentage}%` }}
                  />
                )}
              </div>
            );
          })}
        </div>
      ) : (
        <>
          <RadioGroup value={selectedOption} onValueChange={setSelectedOption}>
            <div className="space-y-2">
              {localVotes.map(option => {
                const percentage = totalVotes > 0 ? (option.votes / totalVotes) * 100 : 0;

                return (
                  <div key={option.id} className="relative">
                    <div className="flex items-center space-x-3 p-3 rounded-lg border hover:bg-accent/50 transition-colors cursor-pointer">
                      <RadioGroupItem value={option.id} id={option.id} />
                      <Label htmlFor={option.id} className="flex-1 cursor-pointer">
                        <div className="flex items-center justify-between">
                          <span>{option.text}</span>
                          {shouldShowData && (
                            <span className="text-sm text-muted-foreground ml-4">
                              {option.votes.toLocaleString()} ({percentage.toFixed(1)}%)
                            </span>
                          )}
                        </div>
                      </Label>
                    </div>
                    {shouldShowData && (
                      <div
                        className="absolute bottom-0 left-0 h-1 bg-indigo-500 rounded-full transition-all duration-500"
                        style={{ width: `${percentage}%` }}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          </RadioGroup>

          {voteCount > 0 && selectedOption && voteCount <= availableVotes && (
            <div className="text-sm text-muted-foreground px-1">
              <p>
                After voting, you will have <span className="font-semibold text-indigo-600">{availableVotes - voteCount}</span> vote{availableVotes - voteCount !== 1 ? 's' : ''} left
              </p>
            </div>
          )}

          <div className="flex items-center gap-3 pt-2">
            <div className="flex items-center border rounded-lg">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => handleVoteCountChange(-1)}
                disabled={voteCount <= 1 || isLoading || !poll.isLive}
                className="h-9 w-9"
              >
                <Minus className="h-4 w-4" />
              </Button>
              <div className="px-4 py-2 min-w-[60px] text-center font-medium">
                {voteCount}
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => handleVoteCountChange(1)}
                disabled={voteCount >= Math.min(100, availableVotes) || isLoading || !poll.isLive}
                className="h-9 w-9"
              >
                <Plus className="h-4 w-4" />
              </Button>
            </div>
            <Button
              onClick={handleVote}
              disabled={!selectedOption || isLoading || availableVotes === 0 || voteCount > availableVotes || !poll.isLive}
              className="flex-1 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? 'Voting...' : !poll.isLive ? 'Poll Closed' : availableVotes === 0 ? 'No Votes Available' : 'Vote'}
            </Button>
          </div>
        </>
      )}

      {showChart && shouldShowData && (
        <div className="pt-4">
          <div className="flex items-center justify-between gap-3 mb-3">
            <h4 className="text-sm font-medium text-muted-foreground">
              {chartMode === 'percentage' ? 'Voting Share' : 'Voting Trends'}
            </h4>
            <div className="inline-flex rounded-md border bg-background p-1">
              <Button
                type="button"
                size="sm"
                variant={chartMode === 'count' ? 'default' : 'ghost'}
                onClick={() => setChartMode('count')}
                className="h-8 px-3"
              >
                Count
              </Button>
              <Button
                type="button"
                size="sm"
                variant={chartMode === 'percentage' ? 'default' : 'ghost'}
                onClick={() => setChartMode('percentage')}
                className="h-8 px-3"
              >
                Percentage
              </Button>
            </div>
          </div>
          <div className="mb-3">
            <div
              ref={timelineRef}
              onScroll={handleTimelineScroll}
              className="flex h-16 w-full items-end gap-1 overflow-x-auto rounded-lg border bg-background px-2 py-2 [scrollbar-width:thin] [scrollbar-color:rgb(99_102_241)_transparent]"
              style={{ scrollBehavior: 'smooth' }}
            >
              {mergedHistory.length === 0 ? (
                <div className="flex h-full min-w-full items-center justify-center text-sm text-muted-foreground">
                  No timeline data yet.
                </div>
              ) : (
                mergedHistory.map((entry, index) => {
                  const isLivePoint = index === mergedHistory.length - 1;
                  const totalAtPoint = Number(entry.totalVotes || 0);
                  const maxTotal = Math.max(1, ...mergedHistory.map((point) => Number(point.totalVotes || 0)));
                  const height = Math.max(8, Math.round((totalAtPoint / maxTotal) * 28));
                  const isVisible = index >= visibleStartIndex && index < visibleStartIndex + windowSize;

                  // determine which option has the highest votes at this timestamp and use its line color
                  const votesByOption = optionIds.map((id) => Number(entry[id] ?? 0));
                  let leadingIdx = 0;
                  let leadingVal = Number.NEGATIVE_INFINITY;
                  for (let i = 0; i < votesByOption.length; i++) {
                    if (votesByOption[i] > leadingVal) {
                      leadingVal = votesByOption[i];
                      leadingIdx = i;
                    }
                  }
                  const dotColor = colors[leadingIdx % colors.length];

                  return (
                    <button
                      key={`${entry.timestamp.getTime()}-${index}`}
                      type="button"
                      onClick={() => {
                        if (timelineRef.current) {
                          timelineRef.current.scrollTo({ left: index * TIMELINE_ITEM_WIDTH, behavior: 'smooth' });
                        }
                      }}
                      className={`flex h-full min-w-[72px] flex-col justify-end rounded-md border px-2 py-1 text-left transition-colors ${isVisible ? 'border-indigo-500 bg-indigo-500/10' : 'border-transparent bg-transparent hover:bg-accent/50'}`}
                    >
                      <div className="flex items-end justify-center gap-1">
                        <div
                          className="w-3 rounded-full"
                          style={{ height, backgroundColor: dotColor }}
                        />
                      </div>
                      <div className="mt-1 text-[10px] text-muted-foreground text-center leading-tight">
                        {isLivePoint ? 'Live' : format(entry.timestamp, 'HH:mm')}
                      </div>
                    </button>
                  );
                })
              )}
            </div>
            <div className="mt-1 flex items-center justify-between px-1 text-[10px] text-muted-foreground">
              <span>Older</span>
              <span>{isFetchingOlderHistory ? 'Loading older history...' : hasOlderHistory ? 'Scroll left for older data' : 'No older data available'}</span>
              <span>Live</span>
            </div>
          </div>

          {chartData.length === 0 ? (
            <div className="h-[200px] flex items-center justify-center rounded-lg border border-dashed border-muted-foreground/30 text-sm text-muted-foreground text-center px-4">
              No data available.
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis
                  dataKey="time"
                  className="text-xs"
                  tick={{ fontSize: 12 }}
                />
                <YAxis className="text-xs" tick={{ fontSize: 12 }} {...chartYAxisProps} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'hsl(var(--background))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '6px',
                  }}
                  formatter={chartTooltipFormatter}
                />
                <Legend wrapperStyle={{ fontSize: '12px' }} />
                {poll.options.map((option, index) => (
                  <Line
                    key={option.id}
                    type="monotone"
                    dataKey={chartMode === 'percentage' ? `${String(option.id)}_percentage` : String(option.id)}
                    name={option.text}
                    stroke={colors[index % colors.length]}
                    strokeWidth={2}
                    dot={false}
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      )}
    </Card>
  );
}