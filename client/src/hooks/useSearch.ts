import { useState, useEffect } from 'react';
import type { SearchResult } from '../types';

export const useSearch = (query: string) => {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!query) return;

    const fetchResults = async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`http://localhost:8000/search?q=${encodeURIComponent(query)}`);
        if (!res.ok) throw new Error('Search failed');

        const data = await res.json();
        setResults(data.results || []);
      } catch (err) {
        console.error(err);
        setError('Failed to fetch results');
      } finally {
        setLoading(false);
      }
    };

    fetchResults();
  }, [query]);

  return { results, loading, error };
};
