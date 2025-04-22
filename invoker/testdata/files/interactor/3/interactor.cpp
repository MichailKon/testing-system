#include <iostream>
#include "testlib.h"

using namespace std;

int main(int argc, char* argv[]) {
    registerInteraction(argc, argv);
    int a = inf.readInt();
    cout << a << std::endl;

    a = ouf.readInt(1, 10, "a");
    cout << a + 1 << std::endl;

    a = ouf.readInt(1, 10, "a");
    tout << a << std::endl;

    quitf(_ok, "OK!!");
}
