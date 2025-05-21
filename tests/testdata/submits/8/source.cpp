#include <iostream>

int main() {
  int a;
  std::cin >> a;
  if (a == 0) {
    std::cout << "0\n";
    return 0;
  }
  int res = 0;
  for (long long i = 1; i < 1000000000000; i++) {
    res += i;
    if (res % i == 0) {
      res--;
    }
  }
  std::cout << res << std::endl;
}