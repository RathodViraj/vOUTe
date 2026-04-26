export interface PollOption {
  id: string;
  text: string;
  votes: number;
}

export interface VoteHistory {
  timestamp: Date;
  optionId: string;
  votes: number;
}

export interface Poll {
  id: string;
  title: string;
  options: PollOption[];
  createdAt: Date;
  isLive: boolean;
  history: VoteHistory[];
}

export interface User {
  id: string;
  username: string;
  email: string;
}

export interface Comment {
  id: string;
  pollId: string;
  username: string;
  text: string;
  timestamp: Date;
}

export interface PastVoteItem {
  pollId: string;
  pollTitle: string;
  timestamp: Date;
  isLive: boolean;
}
