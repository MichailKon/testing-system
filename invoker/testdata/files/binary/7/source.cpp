#include <iostream>
#include <vector>

int main() {
  std::vector<int> s(100000000);
  for (long long i = 0; i < s.size(); i += 2000) {
    s[i] = i + 1;
  }
  int a;
  std::cin >> a;
  std::cout << s[a] << std::endl;
}