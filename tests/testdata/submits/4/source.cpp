#include <iostream>

int main() {
  unsigned a;
  std::cin >> a;
  unsigned x = 0;
  std::cout << *(reinterpret_cast<int*>(x + a)) << std::endl;
}