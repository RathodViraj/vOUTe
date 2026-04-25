export interface Poll {
  id: string;
  title: string;
  options: PollOption[];
  createdAt: Date;
  isLive: boolean;
  history: VoteHistory[];
}

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

export interface UserVote {
  pollId: string;
  pollTitle: string;
  selectedOption: string;
  voteCount: number;
  timestamp: Date;
}

// Generate random history data for charts
const generateHistory = (options: PollOption[]): VoteHistory[] => {
  const history: VoteHistory[] = [];
  const now = new Date();
  
  for (let i = 24; i >= 0; i--) {
    const timestamp = new Date(now.getTime() - i * 60 * 60 * 1000);
    options.forEach(option => {
      history.push({
        timestamp,
        optionId: option.id,
        votes: Math.floor(Math.random() * 1000) + option.votes * 0.8,
      });
    });
  }
  
  return history;
};

export const mockPolls: Poll[] = [
  {
    id: '1',
    title: 'What should be our next product feature?',
    options: [
      { id: '1a', text: 'Dark Mode', votes: 4523 },
      { id: '1b', text: 'Mobile App', votes: 3821 },
      { id: '1c', text: 'AI Assistant', votes: 5234 },
      { id: '1d', text: 'Advanced Analytics', votes: 2156 },
    ],
    createdAt: new Date('2026-04-19'),
    isLive: true,
    history: [],
  },
  {
    id: '2',
    title: 'Best programming language for web development in 2026?',
    options: [
      { id: '2a', text: 'TypeScript', votes: 8234 },
      { id: '2b', text: 'JavaScript', votes: 6421 },
      { id: '2c', text: 'Python', votes: 4532 },
      { id: '2d', text: 'Go', votes: 3214 },
    ],
    createdAt: new Date('2026-04-18'),
    isLive: true,
    history: [],
  },
  {
    id: '3',
    title: 'Favorite UI framework?',
    options: [
      { id: '3a', text: 'React', votes: 12453 },
      { id: '3b', text: 'Vue', votes: 8234 },
      { id: '3c', text: 'Angular', votes: 5432 },
      { id: '3d', text: 'Svelte', votes: 6789 },
    ],
    createdAt: new Date('2026-04-17'),
    isLive: false,
    history: [],
  },
  {
    id: '4',
    title: 'Remote work vs Office - What do you prefer?',
    options: [
      { id: '4a', text: 'Fully Remote', votes: 15234 },
      { id: '4b', text: 'Hybrid', votes: 9876 },
      { id: '4c', text: 'Fully Office', votes: 2345 },
      { id: '4d', text: 'Flexible', votes: 7654 },
    ],
    createdAt: new Date('2026-04-16'),
    isLive: true,
    history: [],
  },
];

// Add history to all polls
mockPolls.forEach(poll => {
  poll.history = generateHistory(poll.options);
});

export const mockUserVotes: UserVote[] = [
  {
    pollId: '1',
    pollTitle: 'What should be our next product feature?',
    selectedOption: 'AI Assistant',
    voteCount: 25,
    timestamp: new Date('2026-04-20T10:30:00'),
  },
  {
    pollId: '2',
    pollTitle: 'Best programming language for web development in 2026?',
    selectedOption: 'TypeScript',
    voteCount: 50,
    timestamp: new Date('2026-04-20T09:15:00'),
  },
];

export const mockBookmarkedPolls: string[] = ['3', '4'];

export interface Comment {
  id: string;
  pollId: string;
  username: string;
  text: string;
  timestamp: Date;
}

export const mockComments: Comment[] = [
  {
    id: 'c1',
    pollId: '1',
    username: 'sarah_dev',
    text: 'AI Assistant would be a game changer! Imagine having automated suggestions and insights.',
    timestamp: new Date('2026-04-20T08:30:00'),
  },
  {
    id: 'c2',
    pollId: '1',
    username: 'tech_enthusiast',
    text: 'Dark mode should be the priority. It\'s 2026 and not having it is unacceptable.',
    timestamp: new Date('2026-04-20T10:15:00'),
  },
  {
    id: 'c3',
    pollId: '1',
    username: 'product_manager',
    text: 'Mobile app is critical for user engagement. Most of our traffic comes from mobile devices.',
    timestamp: new Date('2026-04-20T11:45:00'),
  },
  {
    id: 'c4',
    pollId: '1',
    username: 'data_analyst',
    text: 'Advanced Analytics would help us make better data-driven decisions. This should be top priority!',
    timestamp: new Date('2026-04-20T13:20:00'),
  },
  {
    id: 'c5',
    pollId: '2',
    username: 'js_ninja',
    text: 'TypeScript all the way! The type safety and developer experience are unmatched.',
    timestamp: new Date('2026-04-19T15:30:00'),
  },
  {
    id: 'c6',
    pollId: '2',
    username: 'backend_guru',
    text: 'Python is great for web development too, especially with frameworks like FastAPI and Django.',
    timestamp: new Date('2026-04-19T16:45:00'),
  },
  {
    id: 'c7',
    pollId: '2',
    username: 'fullstack_dev',
    text: 'JavaScript still holds its ground. The ecosystem is massive and keeps evolving.',
    timestamp: new Date('2026-04-19T18:10:00'),
  },
];
