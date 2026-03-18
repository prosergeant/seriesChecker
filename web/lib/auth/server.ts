import { cookies } from 'next/headers';
import Redis from 'ioredis';

const redis = new Redis({
  host: process.env.REDIS_HOST || 'localhost',
  port: parseInt(process.env.REDIS_PORT || '6379'),
  lazyConnect: true,
});

const SESSION_KEY = process.env.REDIS_SESSION_KEY || 'session:';
const COOKIE_NAME = process.env.SESSION_COOKIE_NAME || 'session_id';

export interface Session {
  userId: string;
  expiresAt: Date;
}

export async function getSession(): Promise<Session | null> {
  try {
    const cookieStore = await cookies();
    const sessionId = cookieStore.get(COOKIE_NAME)?.value;
    
    if (!sessionId) return null;
    
    await redis.connect().catch(() => {});
    
    const sessionData = await redis.get(`${SESSION_KEY}${sessionId}`);
    if (!sessionData) return null;
    
    const data = JSON.parse(sessionData);
    return {
      userId: data.userId,
      expiresAt: new Date(data.expiresAt),
    };
  } catch {
    return null;
  }
}

export async function getUserId(): Promise<string | null> {
  const session = await getSession();
  return session?.userId || null;
}
