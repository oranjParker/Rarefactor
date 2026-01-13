import React, { useEffect, useMemo, useState } from 'react';
import { useAutocomplete } from '../hooks/useAutocomplete';

type SearchResult = {
  title?: string;
  url?: string;
  snippet?: string;
  [key: string]: any;
};

function useDebounce<T>(value: T, delay = 250) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);
  }, [value, delay]);
  return debounced;
}

export default function SearchInterface() {
  const [query, setQuery] = useState('');
  const debounced = useDebounce(query, 300);
  const { suggestions } = useAutocomplete(debounced);
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [focused, setFocused] = useState(false);

  const hasQuery = query.trim().length > 0;

  const onSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!hasQuery) return;
    try {
      setLoading(true);
      setError(null);
      const res = await fetch(`http://localhost:8000/v1/search?q=${encodeURIComponent(query)}`);
      const data = await res.json();
      const items: SearchResult[] = data?.results || data?.items || data?.documents || data || [];
      setResults(Array.isArray(items) ? items : []);
    } catch (err: any) {
      setError(err?.message || 'Search failed');
    } finally {
      setLoading(false);
      setFocused(false); // hide suggestions after submitting
    }
  };

  const showSuggestions = useMemo(
    () => focused && suggestions.length > 0 && debounced && (debounced as string).length >= 2,
    [focused, suggestions, debounced]
  );

  return (
    <div className="min-h-screen w-full bg-white">
      <div className="mx-auto max-w-3xl px-4 py-12">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-semibold tracking-tight">Rarefactor Search</h1>
          <p className="text-sm text-gray-500 mt-2">Type to search. Press Enter to submit.</p>
        </div>

        <form onSubmit={onSubmit} className="relative">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onFocus={() => setFocused(true)}
            onBlur={() => setFocused(false)}
            placeholder="Search documents…"
            className="w-full rounded-xl border border-gray-300 bg-white px-4 py-3 text-base shadow-sm outline-none transition focus:border-gray-400 focus:ring-2 focus:ring-blue-500/30"
          />

          {showSuggestions && (
            <ul className="absolute left-0 right-0 z-10 mt-2 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-lg">
              {suggestions.map((s, i) => (
                <li
                  key={`${s}-${i}`}
                  className="cursor-pointer px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                  onMouseDown={(e) => {
                    e.preventDefault();
                    setQuery(s);
                    setFocused(false); // hide suggestions when a suggestion is chosen
                    setTimeout(() => onSubmit(), 0);
                  }}
                >
                  {s}
                </li>
              ))}
            </ul>
          )}
        </form>

        <div className="mt-8">
          {loading && <div className="animate-pulse text-gray-500">Searching…</div>}
          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-red-700">{error}</div>
          )}

          {!loading && results.length > 0 && (
            <div className="space-y-4 transition-all duration-300">
              {results.map((r, idx) => (
                <div key={idx} className="rounded-xl border border-gray-200 p-4 shadow-sm hover:shadow transition">
                  {r.title && <h3 className="text-lg font-medium text-gray-900">{r.title}</h3>}
                  {r.url && (
                    <a href={r.url} target="_blank" rel="noreferrer" className="text-sm text-blue-600 hover:underline">
                      {r.url}
                    </a>
                  )}
                  {r.snippet && <p className="mt-2 text-sm text-gray-700">{r.snippet}</p>}
                  {!r.title && !r.url && !r.snippet && (
                    <pre className="mt-2 overflow-x-auto rounded bg-gray-50 p-2 text-xs text-gray-700">
                      {JSON.stringify(r, null, 2)}
                    </pre>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
