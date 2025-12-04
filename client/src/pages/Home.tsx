import { Container, Title, Stack, Text, Center } from '@mantine/core';
import { SearchInput } from '../components/SearchInput';

export const Home = () => {
  return (
    <Container size="sm" h="100vh">
      <Center h="100%">
        <Stack gap="xl" w="100%">
          <Stack gap={0} align="center">
            <Title order={1} size="3.5rem" style={{ fontWeight: 900, letterSpacing: -2 }}>
              <Text span inherit style={{ color: '#87CEEB' }}>Ra</Text>
              <Text span inherit style={{ color: '#001F3F' }}>re</Text>
              <Text span inherit style={{ color: '#FFA500' }}>fa</Text>
              <Text span inherit style={{ color: '#228B22' }}>ctor</Text>
            </Title>
            <Text c="dimmed">The Python + React Search Engine</Text>
          </Stack>
          <SearchInput size="xl" />
        </Stack>
      </Center>
    </Container>
  );
};
