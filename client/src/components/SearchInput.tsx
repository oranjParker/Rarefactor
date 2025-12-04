import { useState, KeyboardEvent } from 'react';
import { Autocomplete, ActionIcon, useMantineTheme, rem } from '@mantine/core';
import { IconSearch, IconArrowRight } from '@tabler/icons-react';
import { useDebouncedValue } from '@mantine/hooks';
import { useNavigate } from 'react-router-dom';
import { useAutocomplete } from '../hooks/useAutocomplete';

interface SearchInputProps {
  initialQuery?: string;
  size?: 'md' | 'lg' | 'xl';
}

export const SearchInput = ({ initialQuery = '', size = 'lg' }: SearchInputProps) => {
  const theme = useMantineTheme();
  const navigate = useNavigate();

  const [value, setValue] = useState(initialQuery);
  const [debouncedValue] = useDebouncedValue(value, 200);
  const { suggestions } = useAutocomplete(debouncedValue);

  const handleSearch = (query: string) => {
    if (!query.trim()) return;
    navigate(`/search?q=${encodeURIComponent(query)}`);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSearch(value);
    }
  };

  return (
    <Autocomplete
      value={value}
      onChange={setValue}
      onKeyDown={handleKeyDown}
      onOptionSubmit={(val) => handleSearch(val)}
      data={suggestions}
      size={size}
      placeholder="Search the web..."
      radius="xl"
      leftSection={<IconSearch style={{ width: rem(18), height: rem(18) }} stroke={1.5} />}
      rightSection={
        <ActionIcon size={32} radius="xl" color={theme.primaryColor} variant="filled" onClick={() => handleSearch(value)}>
          <IconArrowRight style={{ width: rem(18), height: rem(18) }} stroke={1.5} />
        </ActionIcon>
      }
      rightSectionWidth={42}
      styles={{ input: { paddingLeft: 45 } }}
    />
  );
};
