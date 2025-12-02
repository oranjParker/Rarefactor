from collections import defaultdict
from typing import DefaultDict, List, Optional

class TrieNode:
    def __init__(self):
        self.children: DefaultDict[str, 'TrieNode'] = defaultdict(TrieNode)
        self.isEnd: bool = False
        self.value: Optional[str] = None


class Trie:
    def __init__(self):
        self.root: TrieNode = TrieNode()

    def insert(self, word: str) -> None:
        current: TrieNode = self.root
        for char in word.lower():
            current = current.children[char]
        current.isEnd = True
        current.value = word

    def search(self, word: str) -> bool:
        current: TrieNode = self.root
        for char in word.lower():
            if char not in current.children:
                return False
            current = current.children[char]
        return current.isEnd

    def autocomplete(self, prefix: str, limit: int) -> List[str]:
        results: List[str] = []
        current: TrieNode = self.root
        for char in prefix.lower():
            if char not in current.children:
                return []
            current = current.children[char]
        self._dfs(current, results, limit)
        return results

    def _dfs(self, current: TrieNode, results: List[str], limit: int) -> None:
        if len(results) > limit:
            return
        if current.isEnd:
            results.append(current.value)
            return
        for char in current.children:
            self._dfs(current.children[char], results, limit + 1)