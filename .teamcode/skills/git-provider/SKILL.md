---
name: git-provider
description: >
  Use when integrating a new Git provider (GitHub, GitLab, Bitbucket, Gitea,
  Forgejo, etc.), implementing provider API clients, OAuth flows, webhook
  handling, or provider-specific Git operations.
---

# Git Provider Integration

This skill covers how to add or modify Git provider integrations in
github-desktop-tui.

## Provider Interface

Every provider must implement the following interface:

```typescript
interface GitProvider {
  // Identity
  name: string;
  displayName: string;
  icon: string;           // Emoji or icon character
  
  // Authentication
  authType: 'oauth' | 'token' | 'basic' | 'ssh';
  isAuthenticated(): boolean;
  authenticate(): Promise<AuthResult>;
  
  // Repositories
  listRepositories(): Promise<Repository[]>;
  getRepository(owner: string, name: string): Promise<Repository>;
  
  // Pull Requests / Merge Requests
  listPullRequests(repo: Repository, state?: 'open' | 'closed' | 'all'): Promise<PullRequest[]>;
  getPullRequest(repo: Repository, id: number): Promise<PullRequest>;
  
  // Issues
  listIssues(repo: Repository, state?: 'open' | 'closed' | 'all'): Promise<Issue[]>;
  
  // Branches
  listBranches(repo: Repository): Promise<Branch[]>;
  
  // Commits
  listCommits(repo: Repository, branch?: string): Promise<Commit[]>;
  
  // Reviews / Comments
  listReviews(pr: PullRequest): Promise<Review[]>;
  addComment(pr: PullRequest, body: string): Promise<Comment>;
}
```

## Adding a New Provider

### Step 1: Create provider directory
```
src/providers/<provider-name>/
  client.ts          # API client
  types.ts           # Provider-specific types
  auth.ts            # OAuth / token auth flow
  index.ts           # Re-export + register
```

### Step 2: Implement the client
- Use fetch (Node/Bun) or http (Go) — no heavy SDKs
- Handle rate limiting with exponential backoff
- Implement pagination for list endpoints
- Map provider-specific types to the generic interfaces

### Step 3: Register in the registry
Edit `src/providers/registry.ts`:
```typescript
import { GitHubProvider } from './github';
import { GitLabProvider } from './gitlab';
import { BitbucketProvider } from './bitbucket';
import { GiteaProvider } from './gitea';

export const providers: GitProvider[] = [
  new GitHubProvider(),
  new GitLabProvider(),
  new BitbucketProvider(),
  new GiteaProvider(),
];
```

### Step 4: Add OAuth / token management
- Store tokens securely (OS keychain or encrypted file)
- Never commit tokens to the repository
- Support `GITHUB_TOKEN`, `GITLAB_TOKEN`, etc. env vars

## API Client Pattern

```typescript
class GitHubClient {
  private baseUrl = 'https://api.github.com';
  private token: string;
  
  constructor(token: string) {
    this.token = token;
  }
  
  private async request<T>(path: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'Accept': 'application/vnd.github.v3+json',
        'User-Agent': 'github-desktop-tui',
        ...options?.headers,
      },
    });
    
    if (!response.ok) {
      if (response.status === 429) {
        // Rate limited — wait and retry
        const retryAfter = response.headers.get('Retry-After') ?? '60';
        await new Promise(r => setTimeout(r, parseInt(retryAfter) * 1000));
        return this.request<T>(path, options);
      }
      throw new ApiError(response.status, await response.text());
    }
    
    return response.json();
  }
}
```
