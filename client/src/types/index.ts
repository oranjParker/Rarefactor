export interface SearchResult {
  url: string;
  title: string;
  snippet: string;
  score: number;
}

export interface SearchResponse {
  results: SearchResult[];
  total_hits: number;
}
