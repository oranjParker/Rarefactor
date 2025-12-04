import { useEffect } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { Container, Stack, Title, Text, Paper, Anchor, Group, Loader, Center } from '@mantine/core';
import { SearchInput } from '../components/SearchInput';
import { useSearch } from '../hooks/useSearch';

export const Results = () => {
  const [searchParams] = useSearchParams();
  const query = searchParams.get('q') || '';
  const { results, loading, error } = useSearch(query);

  useEffect(() => {
    document.title = `${query} - Rarefactor Search`;
  }, [query]);

  return (
    <div style={{ paddingBottom: '50px' }}>
      <Paper shadow="xs" p="md" withBorder style={{ position: 'sticky', top: 0, zIndex: 10 }}>
        <Container size="xl">
          <Group align="center">
            <Link to="/" style={{ textDecoration: 'none' }}>
              <Title order={3} style={{ fontWeight: 900, letterSpacing: -1 }}>
                <Text span inherit style={{ color: '#87CEEB' }}>Ra</Text>
                <Text span inherit style={{ color: '#001F3F' }}>re</Text>
                <Text span inherit style={{ color: '#FFA500' }}>fa</Text>
                <Text span inherit style={{ color: '#228B22' }}>ctor</Text>
              </Title>
            </Link>
            <div style={{ flex: 1, maxWidth: '600px' }}>
              <SearchInput initialQuery={query} size="md" />
            </div>
          </Group>
        </Container>
      </Paper>

      <Container size="md" mt="xl">
        {loading ? (
          <Center mt={50}><Loader type="dots" /></Center>
        ) : error ? (
          <Text c="red">Error: {error}</Text>
        ) : (
          <Stack gap="xl">
            <Text size="sm" c="dimmed">Found {results.length} results</Text>
            {results.map((result, index) => (
              <div key={index}>
                <Stack gap={4}>
                  <Text size="sm" c="dimmed" truncate>{result.url}</Text>
                  <Anchor href={result.url} size="xl" fw={500} target="_blank">
                    {result.title}
                  </Anchor>
                  <Text c="dimmed" size="sm" lineClamp={2}>{result.snippet}</Text>
                </Stack>
              </div>
            ))}
            {results.length === 0 && (
              <Text>No results found for "{query}".</Text>
            )}
          </Stack>
        )}
      </Container>
    </div>
  );
};
