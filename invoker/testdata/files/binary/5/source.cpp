#include <iostream>

int main() {
  int res = 0;
  for (long long i = 1; i < 1000000000000; i++) {
    res += i;
    if (res % i == 0) {
      res--;
    }
  }
  std::cout << res << std::endl;
}
