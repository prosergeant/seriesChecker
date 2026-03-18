import { Pool } from 'pg';

let _pool: Pool | null = null;

function getPool(): Pool {
  if (_pool) return _pool;
  
  const connectionString = process.env.DATABASE_URL;
  if (!connectionString) throw new Error('DATABASE_URL not set');
  
  _pool = new Pool({ connectionString, max: 10, idleTimeoutMillis: 30000 });
  return _pool;
}

export async function query<T>(text: string, params?: unknown[]): Promise<T[]> {
  const result = await getPool().query(text, params);
  return result.rows as T[];
}

export async function queryOne<T>(text: string, params?: unknown[]): Promise<T | null> {
  const result = await getPool().query(text, params);
  return (result.rows[0] as T) || null;
}

export { getPool };
