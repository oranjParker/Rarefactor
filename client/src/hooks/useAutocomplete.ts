import { useState, useEffect } from 'react';

export const useAutocomplete = (debouncedTerm: string) => {
  const [suggestions, setSuggestions] = useState<string[]>([]);

  useEffect(() => {
    if (!debouncedTerm || debouncedTerm.length < 2) {
      setSuggestions([]);
      return;
    }

    const fetchSuggestions = async () => {
      try {
        const res = await fetch(`http://localhost:8000/autocomplete?q=${encodeURIComponent(debouncedTerm)}`);
        const data = await res.json();
        setSuggestions(data.suggestions || []);
      } catch (err) {
        console.error('Autocomplete failed', err);
      }
    };

    fetchSuggestions();
  }, [debouncedTerm]);

  return { suggestions };
};
